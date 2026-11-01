package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/photos"
	"github.com/cartograph-asia/cartograph/internal/storage"
)

// namespace makes the fixture ids deterministic, so a second run finds
// everything it would create and does nothing.
var namespace = uuid.MustParse("6f1c2a52-3b8e-4c55-9d2e-7a0c6f2b9e11")

// seedSurveyor is a fixture account: real (not a placeholder), with an
// example.test address so nobody receives mail.
type seedSurveyor struct {
	name     string
	joinedAt time.Duration // before the first run
	bio      string
	home     string // region slug, or empty
}

var seedSurveyors = []seedSurveyor{
	{"noi-pattaya", 45 * 24 * time.Hour, "Born in Naklua, back after twenty years in Bangkok. Mostly early mornings, before the heat.", "chon-buri"},
	{"somchai-k", 30 * 24 * time.Hour, "Motorbike, old Fujifilm, Sukhumvit Road from Bang Lamung to Sattahip.", ""},
	{"mali", 12 * 24 * time.Hour, "", "chon-buri"},
	{"ploy88", 5 * 24 * time.Hour, "Jomtien, on foot.", ""},
	{"jomtien-jo", 26 * time.Hour, "", ""},
}

// FeaturedText is the plot of the week set by the seed.
const featuredText = "Second Road at six in the morning: the songthaews are still parked and the beach belongs to the joggers."

// featuredIndex is the seed photo set as plot of the week.
const featuredIndex = 9

// spot is a seed photo position.
type spot struct{ lat, lng float64 }

// plan returns n photo positions over Pattaya's land (east of Beach Road,
// inside the fixture box): the first `plots` are distinct plots, the rest
// revisit them. Deterministic.
func plan(n, plots int) []spot {
	r := rand.New(rand.NewPCG(8, 2026))
	base := make([]spot, 0, plots)
	seen := map[string]bool{}
	for len(base) < plots {
		s := spot{12.875 + r.Float64()*0.09, 100.889 + r.Float64()*0.055}
		c, _ := h3.CellFromLatLng(s.lat, s.lng, h3.PlotRes)
		if seen[c] {
			continue
		}
		seen[c] = true
		base = append(base, s)
	}
	out := append([]spot(nil), base...)
	for len(out) < n {
		b := base[r.IntN(len(base))]
		// A few metres off, so the photo stays in the same plot.
		out = append(out, spot{b.lat + (r.Float64()-0.5)*0.0004, b.lng + (r.Float64()-0.5)*0.0004})
	}
	return out
}

// Stats reports what Seed did.
type Stats struct {
	Surveyors, Photos, Skipped int
	Featured                   bool
}

// Seed creates the fixture surveyors, n synthetic photos and one featured
// row. Existing rows (by deterministic id) are left alone. The regions must
// already be imported (cmd/regions), otherwise claims fail.
func Seed(ctx context.Context, d *db.DB, store storage.Store, now time.Time, n int) (*Stats, error) {
	now = now.UTC().Truncate(time.Second)
	st := &Stats{}
	ids := make([]uuid.UUID, len(seedSurveyors))
	joined := make([]time.Time, len(seedSurveyors))
	for i, s := range seedSurveyors {
		ids[i] = uuid.NewSHA1(namespace, []byte("surveyor/"+s.name))
		joined[i] = now.Add(-s.joinedAt)
		tag, err := d.Pool.Exec(ctx, `
			insert into surveyors (id, name, email, bio, home_region_id, joined_at)
			values ($1, $2, $3, $4, (select id from regions where slug = nullif($5, '')), $6)
			on conflict do nothing`, ids[i], s.name, s.name+"@example.test", s.bio, s.home, joined[i])
		if err != nil {
			return nil, fmt.Errorf("seed: surveyor %s: %w", s.name, err)
		}
		st.Surveyors += int(tag.RowsAffected())
		// On a second run the stored join date counts.
		if err := d.Pool.QueryRow(ctx, `select id, joined_at from surveyors where lower(name) = lower($1)`, s.name).Scan(&ids[i], &joined[i]); err != nil {
			return nil, fmt.Errorf("seed: surveyor %s: %w", s.name, err)
		}
	}

	r := rand.New(rand.NewPCG(30, 2026))
	spots := plan(n, max(1, n*3/5))
	first := now.Add(-44 * 24 * time.Hour)
	var featuredID uuid.UUID
	for i, sp := range spots {
		id := uuid.NewSHA1(namespace, []byte(fmt.Sprintf("photo/%d", i)))
		if i == featuredIndex {
			featuredID = id
		}
		var exists bool
		if err := d.Pool.QueryRow(ctx, `select exists (select 1 from photos where id = $1)`, id).Scan(&exists); err != nil {
			return nil, fmt.Errorf("seed: photo %d: %w", i, err)
		}
		if exists {
			st.Skipped++
			continue
		}
		// Spread uploads from 44 days ago until an hour ago; each photo
		// goes to a surveyor who had joined by then.
		at := first.Add(time.Duration(float64(now.Sub(first)-time.Hour) * float64(i) / float64(len(spots)))).Add(time.Duration(r.IntN(3600)) * time.Second)
		var who []int
		for j := range seedSurveyors {
			if !joined[j].After(at) {
				who = append(who, j)
			}
		}
		if len(who) == 0 {
			who = []int{0}
		}
		sj := who[r.IntN(len(who))]
		if err := addPhoto(ctx, d, store, id, uint64(i), sp, at, ids[sj], seedSurveyors[sj].name); err != nil {
			return nil, fmt.Errorf("seed: photo %d: %w", i, err)
		}
		st.Photos++
	}

	if featuredID != uuid.Nil {
		tag, err := d.Pool.Exec(ctx, `
			insert into featured (photo_id, body, starts_at)
			select $1, $2, $3 where not exists (select 1 from featured)
			  and exists (select 1 from photos where id = $1)`, featuredID, featuredText, now.Add(-24*time.Hour))
		if err != nil {
			return nil, fmt.Errorf("seed: featured: %w", err)
		}
		st.Featured = tag.RowsAffected() == 1
	}
	return st, nil
}

// addPhoto runs one synthetic photo through the upload pipeline: EXIF,
// variants, storage, claim.
func addPhoto(ctx context.Context, d *db.DB, store storage.Store, id uuid.UUID, seed uint64, sp spot, at time.Time, surveyorID uuid.UUID, name string) error {
	raw, err := synthPhoto(seed, sp.lat, sp.lng, at.Add(-time.Duration(20+seed)*time.Minute))
	if err != nil {
		return err
	}
	meta := photos.ReadMeta(raw)
	if !meta.HasGPS {
		return errors.New("synthetic photo has no GPS")
	}
	cell, err := h3.CellFromLatLng(meta.Lat, meta.Lng, h3.PlotRes)
	if err != nil {
		return err
	}
	p5, err := h3.Parent(cell, 5)
	if err != nil {
		return err
	}
	p6, err := h3.Parent(cell, 6)
	if err != nil {
		return err
	}
	v, err := photos.Process(raw)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("photos/%04d/%02d/%s", at.Year(), int(at.Month()), id)
	keys := map[string][]byte{prefix + "_large.jpg": v.Large, prefix + "_medium.jpg": v.Medium, prefix + "_thumb.jpg": v.Thumb}
	for k, b := range keys {
		if err := store.Put(ctx, k, "image/jpeg", bytes.NewReader(b)); err != nil {
			return fmt.Errorf("store %s: %w", k, err)
		}
	}
	claimed, err := d.ClaimPhoto(ctx, db.NewPhoto{
		ID: id, Cell: cell, Parent5: p5, Parent6: p6, Surveyor: name, SurveyorID: surveyorID,
		KeyLarge: prefix + "_large.jpg", KeyMedium: prefix + "_medium.jpg", KeyThumb: prefix + "_thumb.jpg",
		Width: v.LargeWidth, Height: v.LargeHeight, Lat: meta.Lat, Lng: meta.Lng, TakenAt: meta.TakenAt, TakenLocal: meta.TakenLocal, TakenUTCOffset: meta.TakenOffset, UploadedAt: at,
	})
	if errors.Is(err, db.ErrOutsideRegions) {
		return fmt.Errorf("%w: import the regions first (cmd/regions)", err)
	}
	if err != nil {
		return err
	}
	slog.Debug("seeded photo", "id", id, "cell", cell, "claimed", claimed, "surveyor", name)
	return nil
}
