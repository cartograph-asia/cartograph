package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/views"
)

const (
	maxPlotCells     = 2000
	maxCoverageCells = 500
)

type plotSummary struct {
	Cell          string    `json:"cell"`
	PhotoCount    int       `json:"photo_count"`
	FirstSurveyor string    `json:"first_surveyor"`
	FirstLabel    string    `json:"first_surveyor_label"`
	ClaimedAt     time.Time `json:"claimed_at"`
	Thumb         string    `json:"thumb"`
}

type plotPhoto struct {
	ID         string     `json:"id"`
	Surveyor   string     `json:"surveyor"`
	Label      string     `json:"surveyor_label"`
	TakenAt    *time.Time `json:"taken_at"`
	UploadedAt time.Time  `json:"uploaded_at"`
	URLs       photoURLs  `json:"urls"`
}

type plotDetail struct {
	Cell          string      `json:"cell"`
	PhotoCount    int         `json:"photo_count"`
	FirstSurveyor string      `json:"first_surveyor"`
	FirstLabel    string      `json:"first_surveyor_label"`
	ClaimedAt     time.Time   `json:"claimed_at"`
	Name          string      `json:"name"`
	Path          string      `json:"path"`
	Number        int         `json:"number"`
	Place         string      `json:"place"`
	Region        string      `json:"region"`
	Photos        []plotPhoto `json:"photos"`
}

type coverageCell struct {
	Cell     string `json:"cell"`
	Surveyed int64  `json:"surveyed"`
	Total    int64  `json:"total"`
}

// parseCells parses a comma-separated list of H3 cells at resolution res,
// returning canonical, de-duplicated cells in request order.
func parseCells(raw string, res, max int) ([]string, error) {
	parts := strings.Split(raw, ",")
	cells := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	n := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n++
		if n > max {
			return nil, fmt.Errorf("too many cells: at most %d per request", max)
		}
		c, r, err := h3.Parse(p)
		if err != nil || r != res {
			return nil, fmt.Errorf("invalid cell %q: expected an H3 index at resolution %d", p, res)
		}
		if !seen[c] {
			seen[c] = true
			cells = append(cells, c)
		}
	}
	return cells, nil
}

// listPlots handles GET /api/plots?cells=...
func (s *Server) listPlots(w http.ResponseWriter, r *http.Request) {
	cells, err := parseCells(r.URL.Query().Get("cells"), h3.PlotRes, maxPlotCells)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out := []plotSummary{}
	if len(cells) > 0 {
		rows, err := s.repo.Plots(r.Context(), cells)
		if err != nil {
			slog.Error("list plots", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		for _, p := range rows {
			thumb := ""
			if p.ThumbKey != "" {
				thumb = s.store.URL(p.ThumbKey)
			}
			out = append(out, plotSummary{
				Cell:          p.Cell,
				PhotoCount:    p.PhotoCount,
				FirstSurveyor: p.FirstSurveyor,
				FirstLabel:    p.FirstLabel,
				ClaimedAt:     p.ClaimedAt.UTC(),
				Thumb:         thumb,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plots": out})
}

// getPlot handles GET /api/plots/{cell}.
func (s *Server) getPlot(w http.ResponseWriter, r *http.Request) {
	cell, res, err := h3.Parse(chi.URLParam(r, "cell"))
	if err != nil || res != h3.PlotRes {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid cell: expected an H3 index at resolution %d", h3.PlotRes))
		return
	}
	p, err := s.repo.Plot(r.Context(), cell)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "plot is not surveyed yet")
		return
	}
	if err != nil {
		slog.Error("get plot", "cell", cell, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := plotDetail{
		Cell:          p.Cell,
		PhotoCount:    p.PhotoCount,
		FirstSurveyor: p.FirstSurveyor,
		FirstLabel:    p.FirstLabel,
		ClaimedAt:     p.ClaimedAt.UTC(),
		Name:          plotRef(p).Name(),
		Path:          plotRef(p).Path(),
		Number:        p.Number,
		Place:         p.Place,
		Region:        p.Region,
		Photos:        make([]plotPhoto, 0, len(p.Photos)),
	}
	for _, ph := range p.Photos {
		var taken *time.Time
		if ph.TakenAt != nil {
			t := ph.TakenAt.UTC()
			taken = &t
		}
		out.Photos = append(out.Photos, plotPhoto{
			ID:         ph.ID.String(),
			Surveyor:   ph.Surveyor,
			Label:      ph.Label,
			TakenAt:    taken,
			UploadedAt: ph.UploadedAt.UTC(),
			URLs:       photoURLs{Large: s.store.URL(ph.KeyLarge), Thumb: s.store.URL(ph.KeyThumb)},
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// plotRef is p as the pages name it ("Pattaya · Chon Buri 17").
func plotRef(p *db.PlotDetail) views.PlotRef {
	return views.PlotRef{Cell: p.Cell, Place: p.Place, Number: p.Number, District: p.District,
		Region: views.RegionRef{Slug: p.RegionSlug, NameEn: p.Region}}
}

// coverage handles GET /api/coverage?res=5|6&cells=...
// Only parents with at least one surveyed plot are returned.
func (s *Server) coverage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := strconv.Atoi(q.Get("res"))
	if err != nil || (res != 5 && res != 6) {
		writeError(w, http.StatusBadRequest, "res must be 5 or 6")
		return
	}
	cells, err := parseCells(q.Get("cells"), res, maxCoverageCells)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]coverageCell, 0, len(cells))
	if len(cells) > 0 {
		counts, err := s.repo.Coverage(r.Context(), res, cells)
		if err != nil {
			slog.Error("coverage", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		for _, c := range cells {
			if counts[c] == 0 {
				continue
			}
			total, err := h3.ChildrenCount(c, h3.PlotRes)
			if err != nil {
				slog.Error("children count", "cell", c, "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			out = append(out, coverageCell{Cell: c, Surveyed: counts[c], Total: total})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"cells": out})
}
