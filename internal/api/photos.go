package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/photos"
)

const (
	maxSurveyorLen = 100 // characters
	maxFieldBytes  = 1 << 10
)

type photoURLs struct {
	Large string `json:"large"`
	Thumb string `json:"thumb"`
}

type createPhotoResponse struct {
	ID      string    `json:"id"`
	Cell    string    `json:"cell"`
	Claimed bool      `json:"claimed"`
	URLs    photoURLs `json:"urls"`
}

type uploadForm struct {
	photo    []byte
	surveyor string
	lat, lng string
}

var errTooLarge = errors.New("too large")

// createPhoto handles POST /api/photos.
func (s *Server) createPhoto(w http.ResponseWriter, r *http.Request) {
	maxBytes := int64(s.cfg.MaxUploadMB) << 20
	tooLarge := fmt.Sprintf("file is larger than %d MB", s.cfg.MaxUploadMB)

	// Allow some slack above the file limit for the other fields and the
	// multipart framing; the file itself is checked exactly below.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
	form, err := readUploadForm(r, maxBytes)
	switch {
	case errors.Is(err, errTooLarge):
		writeError(w, http.StatusBadRequest, tooLarge)
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if form.photo == nil {
		writeError(w, http.StatusBadRequest, "photo is required")
		return
	}
	surveyor := strings.TrimSpace(form.surveyor)
	if surveyor == "" {
		writeError(w, http.StatusBadRequest, "surveyor is required")
		return
	}
	if !utf8.ValidString(surveyor) || utf8.RuneCountInString(surveyor) > maxSurveyorLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("surveyor must be valid text of at most %d characters", maxSurveyorLen))
		return
	}

	if _, err := photos.Sniff(form.photo); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Position: EXIF GPS wins; the form fields are only a fallback.
	meta := photos.ReadMeta(form.photo)
	lat, lng := meta.Lat, meta.Lng
	if !meta.HasGPS {
		var ok bool
		lat, lng, ok, err = parseLatLng(form.lat, form.lng)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !ok {
			writeError(w, http.StatusBadRequest, "no GPS position found in the photo; please provide lat and lng")
			return
		}
	}

	cell, err := h3.CellFromLatLng(lat, lng, h3.PlotRes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "position cannot be mapped to a plot")
		return
	}
	parent5, err5 := h3.Parent(cell, 5)
	parent6, err6 := h3.Parent(cell, 6)
	if err5 != nil || err6 != nil {
		slog.Error("compute parents", "cell", cell, "err5", err5, "err6", err6)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Decoding a large photo is memory hungry; bound concurrency.
	select {
	case s.sem <- struct{}{}:
	case <-r.Context().Done():
		return
	}
	variants, err := photos.Process(form.photo)
	<-s.sem
	form.photo = nil
	if err != nil {
		if errors.Is(err, photos.ErrInvalidImage) || errors.Is(err, photos.ErrUnsupported) || errors.Is(err, photos.ErrHEIC) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("process photo", "err", err)
		writeError(w, http.StatusInternalServerError, "could not process photo")
		return
	}

	id := uuid.New()
	now := s.now().UTC()
	prefix := fmt.Sprintf("photos/%04d/%02d/%s", now.Year(), int(now.Month()), id)
	keyLarge, keyMedium, keyThumb := prefix+"_large.jpg", prefix+"_medium.jpg", prefix+"_thumb.jpg"

	ctx := r.Context()
	if err := s.store.Put(ctx, keyLarge, "image/jpeg", bytes.NewReader(variants.Large)); err != nil {
		slog.Error("store photo", "key", keyLarge, "err", err)
		writeError(w, http.StatusInternalServerError, "could not store photo")
		return
	}
	if err := s.store.Put(ctx, keyMedium, "image/jpeg", bytes.NewReader(variants.Medium)); err != nil {
		slog.Error("store photo", "key", keyMedium, "err", err, "orphaned_keys", []string{keyLarge})
		writeError(w, http.StatusInternalServerError, "could not store photo")
		return
	}
	if err := s.store.Put(ctx, keyThumb, "image/jpeg", bytes.NewReader(variants.Thumb)); err != nil {
		slog.Error("store photo", "key", keyThumb, "err", err, "orphaned_keys", []string{keyLarge, keyMedium})
		writeError(w, http.StatusInternalServerError, "could not store photo")
		return
	}

	claimed, err := s.repo.ClaimPhoto(ctx, db.NewPhoto{
		ID:         id,
		Cell:       cell,
		Parent5:    parent5,
		Parent6:    parent6,
		Surveyor:   surveyor,
		KeyLarge:   keyLarge,
		KeyMedium:  keyMedium,
		KeyThumb:   keyThumb,
		Width:      variants.LargeWidth,
		Height:     variants.LargeHeight,
		Lat:        lat,
		Lng:        lng,
		TakenAt:    meta.TakenAt,
		UploadedAt: now,
	})
	if errors.Is(err, db.ErrOutsideRegions) {
		slog.Info("photo outside every region", "cell", cell, "orphaned_keys", []string{keyLarge, keyMedium, keyThumb})
		writeError(w, http.StatusBadRequest, "this photo was taken outside the area Cartograph covers so far (Thailand)")
		return
	}
	if err != nil {
		slog.Error("claim photo", "err", err, "orphaned_keys", []string{keyLarge, keyMedium, keyThumb})
		writeError(w, http.StatusInternalServerError, "could not save photo")
		return
	}

	slog.Info("plot surveyed", "photo_id", id, "cell", cell, "claimed", claimed, "exif_gps", meta.HasGPS)
	writeJSON(w, http.StatusCreated, createPhotoResponse{
		ID:      id.String(),
		Cell:    cell,
		Claimed: claimed,
		URLs:    photoURLs{Large: s.store.URL(keyLarge), Thumb: s.store.URL(keyThumb)},
	})
}

// readUploadForm streams the multipart body, keeping the photo in memory
// (no temp files: the runtime image is distroless).
func readUploadForm(r *http.Request, maxFile int64) (*uploadForm, error) {
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct != "multipart/form-data" {
		return nil, errors.New("request must be multipart/form-data")
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, errors.New("request must be multipart/form-data")
	}
	f := &uploadForm{}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, bodyErr(err)
		}
		switch part.FormName() {
		case "photo":
			if f.photo != nil {
				return nil, errors.New("only one photo per upload")
			}
			var buf bytes.Buffer
			n, err := io.Copy(&buf, io.LimitReader(part, maxFile+1))
			if err != nil {
				return nil, bodyErr(err)
			}
			if n > maxFile {
				return nil, errTooLarge
			}
			if n == 0 {
				return nil, errors.New("photo is empty")
			}
			f.photo = buf.Bytes()
		case "surveyor", "lat", "lng":
			b, err := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
			if err != nil {
				return nil, bodyErr(err)
			}
			if len(b) > maxFieldBytes {
				return nil, fmt.Errorf("field %s is too long", part.FormName())
			}
			switch part.FormName() {
			case "surveyor":
				f.surveyor = string(b)
			case "lat":
				f.lat = string(b)
			case "lng":
				f.lng = string(b)
			}
		default:
			if _, err := io.Copy(io.Discard, part); err != nil {
				return nil, bodyErr(err)
			}
		}
		part.Close()
	}
	return f, nil
}

func bodyErr(err error) error {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return errTooLarge
	}
	return errors.New("could not read upload")
}

// parseLatLng parses the optional form position. ok is false when neither
// field is given.
func parseLatLng(latS, lngS string) (lat, lng float64, ok bool, err error) {
	latS, lngS = strings.TrimSpace(latS), strings.TrimSpace(lngS)
	if latS == "" && lngS == "" {
		return 0, 0, false, nil
	}
	errBad := errors.New("lat and lng must both be given as decimal degrees (lat -90..90, lng -180..180)")
	if latS == "" || lngS == "" {
		return 0, 0, false, errBad
	}
	lat, err1 := strconv.ParseFloat(latS, 64)
	lng, err2 := strconv.ParseFloat(lngS, 64)
	if err1 != nil || err2 != nil || !photos.ValidLatLng(lat, lng) {
		return 0, 0, false, errBad
	}
	return lat, lng, true, nil
}
