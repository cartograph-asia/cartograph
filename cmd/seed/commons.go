package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/commons"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/storage"
)

// commonsSurveyor is a development account that holds photos from Wikimedia
// Commons (docs/SEED_PHOTOS_BRIEF.md), with an example.test address. The photos
// are dealt out over all of Thailand, so a bio names the home, not a route.
type commonsSurveyor struct {
	name, display, bio string
	home               string // region slug, or empty
	joinedAgo          time.Duration
}

var commonsSurveyors = []commonsSurveyor{
	{"north-trails", "Napat Srisuk", "Lives in Chiang Mai, travels the whole country with a camera and no plan. Slow roads, long lunches.", "chiang-mai", 58 * 24 * time.Hour},
	{"isan-rider", "", "Khon Kaen is home; the rest of Thailand is a weekend away by bus or motorbike.", "khon-kaen", 49 * 24 * time.Hour},
	{"andaman-ann", "Ann P.", "Phuket is home. Rainy season is the best season, wherever the trip goes.", "phuket", 41 * 24 * time.Hour},
	{"krung-thep-lek", "", "Bangkok by bicycle and ferry at home, the rest of the country by night train.", "bangkok", 33 * 24 * time.Hour},
	{"gulf-coast-ton", "Ton", "Surat Thani is home. Trips go by train, one plot at every station.", "surat-thani", 24 * 24 * time.Hour},
	{"central-plains-nok", "", "Phitsanulok is home. Temples at dawn, then the fields, in every province.", "phitsanulok", 15 * 24 * time.Hour},
}

// CommonsStats reports what SeedCommons did.
type CommonsStats struct {
	Surveyors, Photos, Skipped, Outside int
}

// SeedCommons creates the Commons Surveyors and gives each of them photos from
// the manifest in dir, dealt out one after the other so that every Surveyor
// has photos from many places. Existing rows (by deterministic id) are left
// alone; a photo outside every region is skipped. The regions of Thailand
// must be imported first.
func SeedCommons(ctx context.Context, d *db.DB, store storage.Store, now time.Time, dir string) (*CommonsStats, error) {
	m, err := commons.Read(dir)
	if err != nil {
		return nil, err
	}
	now = now.UTC().Truncate(time.Second)
	st := &CommonsStats{}
	ids := make([]uuid.UUID, len(commonsSurveyors))
	joined := make([]time.Time, len(commonsSurveyors))
	for i, s := range commonsSurveyors {
		ids[i] = uuid.NewSHA1(namespace, []byte("surveyor/"+s.name))
		tag, err := d.Pool.Exec(ctx, `
			insert into surveyors (id, name, display_name, email, bio, home_region_id, joined_at)
			values ($1, $2, nullif($3, ''), $4, $5, (select id from regions where slug = nullif($6, '')), $7)
			on conflict do nothing`, ids[i], s.name, s.display, s.name+"@example.test", s.bio, s.home, now.Add(-s.joinedAgo))
		if err != nil {
			return nil, fmt.Errorf("seed: surveyor %s: %w", s.name, err)
		}
		st.Surveyors += int(tag.RowsAffected())
		if err := d.Pool.QueryRow(ctx, `select id, joined_at from surveyors where lower(name) = lower($1)`, s.name).Scan(&ids[i], &joined[i]); err != nil {
			return nil, fmt.Errorf("seed: surveyor %s: %w", s.name, err)
		}
	}

	r := rand.New(rand.NewPCG(31, 2026))
	first := now.Add(-60 * 24 * time.Hour)
	for i, p := range m.Photos {
		id := uuid.NewSHA1(namespace, []byte("commons/"+p.File))
		var exists bool
		if err := d.Pool.QueryRow(ctx, `select exists (select 1 from photos where id = $1)`, id).Scan(&exists); err != nil {
			return nil, fmt.Errorf("seed: %s: %w", p.File, err)
		}
		if exists {
			st.Skipped++
			continue
		}
		j := i % len(commonsSurveyors)
		// Spread uploads over 60 days, never before the Surveyor joined.
		at := first.Add(time.Duration(float64(now.Sub(first)-time.Hour) * float64(i) / float64(len(m.Photos)))).Add(time.Duration(r.IntN(3600)) * time.Second)
		if at.Before(joined[j]) {
			at = joined[j].Add(time.Duration(1+r.IntN(48)) * time.Hour)
		}
		taken := at.Add(-time.Duration(3+r.IntN(40)) * time.Hour)
		if p.Taken != nil && p.Taken.Before(at) {
			taken = *p.Taken
		}
		raw, err := os.ReadFile(filepath.Join(dir, p.File))
		if err != nil {
			return nil, err
		}
		if raw, err = withExif(raw, p.Lat, p.Lng, taken); err != nil {
			return nil, fmt.Errorf("seed: %s: %w", p.File, err)
		}
		err = ingest(ctx, d, store, id, raw, at, ids[j], commonsSurveyors[j].name)
		if errors.Is(err, db.ErrOutsideRegions) {
			slog.Warn("photo is outside every region, skipped (are the regions of Thailand imported?)", "file", p.File, "place", p.Place)
			st.Outside++
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("seed: %s: %w", p.File, err)
		}
		st.Photos++
	}
	return st, nil
}
