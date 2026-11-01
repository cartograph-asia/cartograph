package db

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/migrations"
)

// TestSchema applies every migration on a fresh schema and checks the
// constraints the Home brief relies on.
func TestSchema(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	var applied int
	if err := d.Pool.QueryRow(ctx, "select count(*) from schema_migrations").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != len(files) {
		t.Fatalf("applied %d of %d migrations", applied, len(files))
	}

	exec := func(sql string, args ...any) error {
		_, err := d.Pool.Exec(ctx, sql, args...)
		return err
	}
	mustOK := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	mustFail := func(what, constraint string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), constraint) {
			t.Fatalf("%s: want violation of %q, got %v", what, constraint, err)
		}
	}
	addSurveyor := func(name string, email any, placeholder bool) error {
		return exec("insert into surveyors (id, name, email, placeholder) values ($1, $2, $3, $4)",
			uuid.New(), name, email, placeholder)
	}

	// Regions.
	var region int
	mustOK("insert region", d.Pool.QueryRow(ctx, `
		insert into regions (osm_id, admin_level, country, slug, name_en, name_th, boundary,
		                     min_lat, min_lng, max_lat, max_lng, land_cells)
		values (1, 4, 'th', 'chon-buri', 'Chon Buri', 'ชลบุรี', '{"type":"Polygon","coordinates":[[[100.68,12.5],[101.73,12.5],[101.73,13.59],[100.68,13.59],[100.68,12.5]]]}',
		        12.5, 100.68, 13.59, 101.73, 5921)
		returning id`).Scan(&region))
	mustFail("bad slug", "regions_slug_check", exec(`
		insert into regions (admin_level, country, slug, name_en, boundary, min_lat, min_lng, max_lat, max_lng)
		values (4, 'th', 'Chon Buri', 'x', '{}', 0, 0, 0, 0)`))
	mustOK("insert place", exec(`insert into places (osm_type, osm_id, kind, name_en, name_th, lat, lng, region_id)
		values ('n', 42, 'city', 'Pattaya', 'พัทยา', 12.93, 100.88, $1)`, region))

	// Surveyors: names, case-insensitive uniqueness, nullable unique e-mail, bio.
	mustOK("surveyor", addSurveyor("Adrian", "adrian@example.org", false))
	mustFail("name case", "surveyors_name_lower_idx", addSurveyor("adrian", nil, true))
	mustFail("e-mail case", "surveyors_email_lower_idx", addSurveyor("someone", "ADRIAN@example.org", false))
	mustOK("placeholder 1", addSurveyor("beach-walker", nil, true))
	mustOK("placeholder 2", addSurveyor("Grace", nil, true))
	for _, bad := range []string{"ab", strings.Repeat("a", 31), "has space", "under_score", "ä-umlaut"} {
		mustFail("name "+bad, "surveyors_name_check", addSurveyor(bad, nil, false))
	}
	mustOK("name 30", addSurveyor(strings.Repeat("a", 30), nil, false))
	mustOK("bio 500", exec("update surveyors set bio = $1, home_region_id = $2 where name = 'Adrian'", strings.Repeat("ก", 500), region))
	mustFail("bio 501", "surveyors_bio_check", exec("update surveyors set bio = $1 where name = 'Adrian'", strings.Repeat("a", 501)))

	// Tokens and sessions store 32-byte hashes; a session may exist before a name is chosen.
	hash := make([]byte, 32)
	mustOK("login token", exec("insert into login_tokens (token_hash, email, expires_at) values ($1, 'new@example.org', now() + interval '15 minutes')", hash))
	mustFail("short hash", "login_tokens_token_hash_check", exec("insert into login_tokens (token_hash, email, expires_at) values ($1, 'x', now())", []byte("raw")))
	mustOK("nameless session", exec("insert into sessions (token_hash, email, expires_at) values ($1, 'new@example.org', now() + interval '30 days')", hash))

	// A prototype photo can be attached to a placeholder account later
	// (task 2's 0050+ migration), and new photos carry the new columns.
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	p := newPhoto("88658bd11dfffff", "beach walker", t0)
	if _, err := d.ClaimPhoto(ctx, p); err != nil {
		t.Fatal(err)
	}
	mustOK("attach placeholder", exec(`
		update photos ph set surveyor_id = s.id from surveyors s where s.name = 'beach-walker' and ph.id = $1`, p.ID))
	mustOK("attach plot", exec(`
		update plots pl set first_surveyor_id = ph.surveyor_id, region_id = $2, place_name = 'Pattaya', place_name_th = 'พัทยา'
		from photos ph where ph.id = pl.first_photo and pl.cell = $1`, p.Cell, region))
	var licence string
	var medium *string
	mustOK("read photo", d.Pool.QueryRow(ctx, "select licence, key_medium from photos where id = $1", p.ID).Scan(&licence, &medium))
	if licence != "CC BY-SA 4.0" || medium != nil {
		t.Fatalf("licence %q, key_medium %v", licence, medium)
	}

	// Featured.
	mustOK("featured", exec("insert into featured (photo_id, body) values ($1, 'Second Road at six')", p.ID))
	mustFail("featured window", "featured_check", exec("insert into featured (photo_id, starts_at, ends_at) values ($1, $2, $2)", p.ID, t0))
}
