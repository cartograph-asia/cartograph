package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/photos"
	"github.com/cartograph-asia/cartograph/internal/views"
)

const maxFieldBytes = 1 << 10

// crossOrigin refuses uploads that a browser sends from another origin
// (Sec-Fetch-Site / Origin). With the session cookie this is the endpoint's
// CSRF protection: the map page posts multipart from the same origin.
var crossOrigin = http.NewCrossOriginProtection()

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

// uploadForm is a parsed upload. The photo is spooled to a temporary file,
// so that a slow upload holds disk, not memory; close removes it.
type uploadForm struct {
	photo    *os.File // nil if the request had no photo
	size     int64
	lat, lng string
}

func (f *uploadForm) close() {
	if f.photo == nil {
		return
	}
	name := f.photo.Name()
	_ = f.photo.Close()
	if err := os.Remove(name); err != nil {
		slog.Error("remove spooled upload", "file", name, "err", err)
	}
	f.photo = nil
}

// file returns a reader over the whole spooled photo.
func (f *uploadForm) file() *io.SectionReader { return io.NewSectionReader(f.photo, 0, f.size) }

var (
	errTooLarge = errors.New("too large")
	errSpool    = errors.New("could not store the upload")
)

// createPhoto handles POST /api/photos.
// The Surveyor is the signed-in viewer (internal/auth's middleware); a
// "surveyor" form field is ignored.
func (s *Server) createPhoto(w http.ResponseWriter, r *http.Request) {
	if err := crossOrigin.Check(r); err != nil {
		writeError(w, http.StatusForbidden, "cross-origin uploads are not allowed")
		return
	}
	viewer := views.ViewerFrom(r.Context())
	if !viewer.HasName() {
		writeError(w, http.StatusUnauthorized, "sign in to survey")
		return
	}
	surveyor := viewer.Name

	// One upload at a time per Surveyor (see docs/decisions.md, Security).
	if !s.startUpload(viewer.SurveyorID) {
		writeError(w, http.StatusTooManyRequests, "another upload of yours is still in progress; wait until it has finished")
		return
	}
	defer s.endUpload(viewer.SurveyorID)

	// A 10-20 MB phone photo over mobile data can take longer than the
	// server's ReadTimeout, which stays short for every other route.
	rc := http.NewResponseController(w)
	deadline := time.Now().Add(s.cfg.UploadTimeout) // real time, not s.now: a socket deadline
	if err := errors.Join(rc.SetReadDeadline(deadline), rc.SetWriteDeadline(deadline.Add(time.Minute))); err != nil {
		slog.Warn("upload deadline not extended", "err", err)
	}

	maxBytes := int64(s.cfg.MaxUploadMB) << 20
	tooLarge := fmt.Sprintf("file is larger than %d MB", s.cfg.MaxUploadMB)

	// Allow some slack above the file limit for the other fields and the
	// multipart framing; the file itself is checked exactly below.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
	form, err := readUploadForm(r, maxBytes, s.cfg.SpoolDir)
	if form != nil {
		defer form.close()
	}
	switch {
	case errors.Is(err, errTooLarge):
		writeError(w, http.StatusBadRequest, tooLarge)
		return
	case errors.Is(err, errSpool):
		slog.Error("spool upload", "err", err)
		writeError(w, http.StatusInternalServerError, "could not receive photo")
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if form.photo == nil {
		writeError(w, http.StatusBadRequest, "photo is required")
		return
	}

	head := make([]byte, 16)
	n, _ := io.ReadFull(form.file(), head)
	if _, err := photos.Sniff(head[:n]); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Position: EXIF GPS wins; the form fields are only a fallback.
	meta := photos.ReadMetaFrom(form.file())
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

	// Decoding is what needs memory: the file plus a decoded image several
	// times its size. At most MaxConcurrentProcessing photos are read into
	// memory and decoded at once; the slot is taken only now that the body
	// is on disk, and given back before the small variants are stored.
	select {
	case s.sem <- struct{}{}:
	case <-r.Context().Done():
		return
	}
	variants, err := s.process(form)
	<-s.sem
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

	// The triggers of migration 0051 would fill the ids from the name too;
	// they only act when SurveyorID is null.
	surveyorID, _ := uuid.Parse(viewer.SurveyorID)
	claimed, err := s.repo.ClaimPhoto(ctx, db.NewPhoto{
		ID:             id,
		Cell:           cell,
		Parent5:        parent5,
		Parent6:        parent6,
		Surveyor:       surveyor,
		SurveyorID:     surveyorID,
		KeyLarge:       keyLarge,
		KeyMedium:      keyMedium,
		KeyThumb:       keyThumb,
		Width:          variants.LargeWidth,
		Height:         variants.LargeHeight,
		Lat:            lat,
		Lng:            lng,
		TakenAt:        meta.TakenAt,
		TakenLocal:     meta.TakenLocal,
		TakenUTCOffset: meta.TakenOffset,
		UploadedAt:     now,
		Camera:         meta.Camera,
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

// process reads the spooled photo into memory and makes the variants.
func (s *Server) process(form *uploadForm) (*photos.Variants, error) {
	b := make([]byte, form.size)
	if _, err := io.ReadFull(form.file(), b); err != nil {
		return nil, fmt.Errorf("read spooled upload: %w", err)
	}
	return photos.Process(b)
}

// startUpload marks an upload of Surveyor id as running; false if one
// already is.
func (s *Server) startUpload(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uploading[id] {
		return false
	}
	s.uploading[id] = true
	return true
}

func (s *Server) endUpload(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.uploading, id)
}

// readUploadForm streams the multipart body, spooling the photo to a
// temporary file in dir ("" = os.TempDir). On error it removes the file
// itself; otherwise the caller closes the form.
func readUploadForm(r *http.Request, maxFile int64, dir string) (form *uploadForm, err error) {
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct != "multipart/form-data" {
		return nil, errors.New("request must be multipart/form-data")
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, errors.New("request must be multipart/form-data")
	}
	f := &uploadForm{}
	defer func() {
		if err != nil {
			f.close()
		}
	}()
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
			if f.photo, err = os.CreateTemp(dir, "upload-*"); err != nil {
				return nil, fmt.Errorf("%w: %v", errSpool, err)
			}
			n, err := io.Copy(f.photo, io.LimitReader(part, maxFile+1))
			if err != nil {
				return nil, bodyErr(err)
			}
			if n > maxFile {
				return nil, errTooLarge
			}
			if n == 0 {
				return nil, errors.New("photo is empty")
			}
			f.size = n
		case "lat", "lng":
			b, err := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
			if err != nil {
				return nil, bodyErr(err)
			}
			if len(b) > maxFieldBytes {
				return nil, fmt.Errorf("field %s is too long", part.FormName())
			}
			switch part.FormName() {
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
