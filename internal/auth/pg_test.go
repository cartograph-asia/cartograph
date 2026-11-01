package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/migrations"
)

// testPool connects to TEST_DATABASE_URL in a fresh schema. The test is
// skipped when the variable is not set.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop schema "+schema+" cascade")
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// migrate applies the embedded migrations whose names sort before stop
// ("" = all of them).
func migrate(t *testing.T, pool *pgxpool.Pool, stop string) {
	t.Helper()
	sub := fstest.MapFS{}
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if stop != "" && n >= stop {
			continue
		}
		b, err := fs.ReadFile(migrations.FS, n)
		if err != nil {
			t.Fatal(err)
		}
		sub[n] = &fstest.MapFile{Data: b}
	}
	if err := (&db.DB{Pool: pool}).Migrate(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
}

// protoPhoto inserts a prototype-style photo (and its plot, if new) the way
// the prototype's ClaimPhoto did, without the Home columns.
func protoPhoto(t *testing.T, pool *pgxpool.Pool, cell, surveyor string, at time.Time) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		insert into plots (cell, first_photo, first_surveyor, claimed_at, photo_count, parent5, parent6)
		values ($1, $2, $3, $4, 1, 'p5', 'p6')
		on conflict (cell) do update set photo_count = plots.photo_count + 1`, cell, id, surveyor, at)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		insert into photos (id, cell, surveyor, key_large, key_thumb, lat, lng, uploaded_at)
		values ($1, $2, $3, 'l', 't', 12.9, 100.9, $4)`, id, cell, surveyor, at)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPlaceholderMigration(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	migrate(t, pool, "0050")

	t0 := time.Date(2025, 3, 1, 8, 0, 0, 0, time.UTC)
	at := func(h int) time.Time { return t0.Add(time.Duration(h) * time.Hour) }
	// Awkward prototype names, in order of their first upload.
	protoPhoto(t, pool, "88658bd11dfffff", "Ada Lovelace", at(0))
	protoPhoto(t, pool, "88658bd11dfffff", "ada-lovelace", at(1)) // collides with the first after slugifying
	protoPhoto(t, pool, "88658bd13bfffff", "สมชาย", at(2))        // Thai script only
	protoPhoto(t, pool, "88658bd139fffff", "J", at(3))            // one character
	protoPhoto(t, pool, "88658bd131fffff", "  Zoë  & Björn  ", at(4))
	protoPhoto(t, pool, "88658bd135fffff", "Ada Lovelace", at(5)) // second photo of the first name
	protoPhoto(t, pool, "88658bd137fffff", "นก", at(6))           // Thai again: collides with "surveyor"
	protoPhoto(t, pool, "88658bd133fffff", strings.Repeat("Long name ", 5), at(7))
	protoPhoto(t, pool, "88658bd1abfffff", "J", at(8))

	migrate(t, pool, "")

	got := map[string]string{} // prototype name -> account name
	rows, err := pool.Query(ctx, `
		select distinct ph.surveyor, s.name, s.placeholder, s.email is null, s.joined_at = (
		  select min(uploaded_at) from photos where surveyor = ph.surveyor)
		from photos ph join surveyors s on s.id = ph.surveyor_id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var proto, name string
		var placeholder, noEmail, joined bool
		if err := rows.Scan(&proto, &name, &placeholder, &noEmail, &joined); err != nil {
			t.Fatal(err)
		}
		if !placeholder || !noEmail || !joined {
			t.Errorf("%q -> %q: placeholder %v, no e-mail %v, joined at first upload %v", proto, name, placeholder, noEmail, joined)
		}
		got[proto] = name
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	wantNames := map[string]string{
		"Ada Lovelace":                  "Ada-Lovelace",
		"ada-lovelace":                  "ada-lovelace-2",
		"สมชาย":                         "surveyor",
		"J":                             "J-1",
		"  Zoë  & Björn  ":              "Zoe-Bjorn",
		"นก":                            "surveyor-2",
		strings.Repeat("Long name ", 5): "Long-name-Long-name-Long-name",
	}
	if len(got) != len(wantNames) {
		t.Errorf("got %d accounts, want %d: %v", len(got), len(wantNames), got)
	}
	for proto, name := range wantNames {
		if got[proto] != name {
			t.Errorf("%q -> %q, want %q", proto, got[proto], name)
		}
	}

	var unattached, plotsWithout, wrongPlot int
	if err := pool.QueryRow(ctx, "select count(*) from photos where surveyor_id is null").Scan(&unattached); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "select count(*) from plots where first_surveyor_id is null").Scan(&plotsWithout); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		select count(*) from plots pl join photos ph on ph.id = pl.first_photo
		where pl.first_surveyor_id is distinct from ph.surveyor_id`).Scan(&wrongPlot); err != nil {
		t.Fatal(err)
	}
	if unattached != 0 || plotsWithout != 0 || wrongPlot != 0 {
		t.Fatalf("unattached photos %d, plots without first surveyor %d, mismatched %d", unattached, plotsWithout, wrongPlot)
	}
	var first string
	if err := pool.QueryRow(ctx, `
		select s.name from plots pl join surveyors s on s.id = pl.first_surveyor_id
		where pl.cell = '88658bd11dfffff'`).Scan(&first); err != nil || first != "Ada-Lovelace" {
		t.Fatalf("first surveyor of the shared plot: %q %v", first, err)
	}

	// New uploads under an account name get the ids from the triggers (0051).
	id := protoPhoto(t, pool, "88658bd1a9fffff", "Ada-Lovelace", at(20))
	var photoOwner, plotOwner string
	if err := pool.QueryRow(ctx, `
		select s1.name, s2.name from photos ph join surveyors s1 on s1.id = ph.surveyor_id
		join plots pl on pl.cell = ph.cell join surveyors s2 on s2.id = pl.first_surveyor_id
		where ph.id = $1`, id).Scan(&photoOwner, &plotOwner); err != nil || photoOwner != "Ada-Lovelace" || plotOwner != "Ada-Lovelace" {
		t.Fatalf("trigger: %q %q %v", photoOwner, plotOwner, err)
	}
}

func TestPGStore(t *testing.T) {
	pool := testPool(t)
	migrate(t, pool, "")
	ctx := context.Background()
	s := NewPGStore(pool)
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

	// Login tokens: single use, expiry, per-address count.
	tok := func(email string, created time.Time) []byte {
		h := hashToken(newToken())
		if err := s.CreateLoginToken(ctx, LoginToken{Hash: h, Email: email, Next: "/map", CreatedAt: created, ExpiresAt: created.Add(LinkTTL)}); err != nil {
			t.Fatal(err)
		}
		return h
	}
	h1 := tok("new@example.org", now)
	tok("new@example.org", now.Add(-2*time.Hour))
	if n, err := s.CountLoginTokens(ctx, "new@example.org", now.Add(-time.Hour)); err != nil || n != 1 {
		t.Fatalf("count: %d %v", n, err)
	}
	if _, _, err := s.ConsumeLoginToken(ctx, h1, now.Add(LinkTTL)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired token: %v", err)
	}
	email, next, err := s.ConsumeLoginToken(ctx, h1, now.Add(time.Minute))
	if err != nil || email != "new@example.org" || next != "/map" {
		t.Fatalf("consume: %q %q %v", email, next, err)
	}
	if _, _, err := s.ConsumeLoginToken(ctx, h1, now.Add(time.Minute)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reuse: %v", err)
	}

	// A new address: a session without a Surveyor, then the name.
	sh := hashToken(newToken())
	v, err := s.CreateSession(ctx, sh, "new@example.org", now, now.Add(SessionTTL))
	if err != nil || v.HasName() || v.Email != "new@example.org" {
		t.Fatalf("new session: %+v %v", v, err)
	}
	sh2 := hashToken(newToken()) // a second browser, also before the name
	if _, err := s.CreateSession(ctx, sh2, "new@example.org", now, now.Add(SessionTTL)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "insert into surveyors (id, name, placeholder) values ($1, 'Taken', true)", uuid.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChooseName(ctx, sh, "TAKEN", now); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("taken: %v", err)
	}
	v, err = s.ChooseName(ctx, sh, "Nok", now.Add(time.Minute))
	if err != nil || v.Name != "Nok" || v.SurveyorID == "" {
		t.Fatalf("choose: %+v %v", v, err)
	}
	if v2, err := s.ChooseName(ctx, sh, "Other", now); err != nil || v2.Name != "Nok" {
		t.Fatalf("second choice: %+v %v", v2, err)
	}
	if ses, err := s.Session(ctx, sh2, now); err != nil || ses.Viewer.Name != "Nok" {
		t.Fatalf("other browser: %+v %v", ses, err)
	}
	var joined time.Time
	if err := pool.QueryRow(ctx, "select joined_at from surveyors where name = 'Nok'").Scan(&joined); err != nil || !joined.Equal(now.Add(time.Minute)) {
		t.Fatalf("joined_at %v %v", joined, err)
	}
	if _, err := s.ChooseName(ctx, hashToken(newToken()), "Nobody", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown session: %v", err)
	}

	// Session expiry and touch.
	if _, err := s.Session(ctx, sh, now.Add(SessionTTL)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session: %v", err)
	}
	later := now.Add(SessionTTL - time.Hour)
	if err := s.TouchSession(ctx, sh, later, later.Add(SessionTTL)); err != nil {
		t.Fatal(err)
	}
	if ses, err := s.Session(ctx, sh, now.Add(SessionTTL)); err != nil || !ses.LastSeen.Equal(later) {
		t.Fatalf("touched session: %+v %v", ses, err)
	}
	if err := s.DeleteSession(ctx, sh); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Session(ctx, sh, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted session: %v", err)
	}

	// Claiming a placeholder: Adrian sets the e-mail (any case) in SQL.
	if _, err := pool.Exec(ctx, "insert into surveyors (id, name, email, placeholder) values ($1, 'beach-walker', 'Adrian@Example.org', true)", uuid.New()); err != nil {
		t.Fatal(err)
	}
	v, err = s.CreateSession(ctx, hashToken(newToken()), "adrian@example.org", now, now.Add(SessionTTL))
	if err != nil || v.Name != "beach-walker" {
		t.Fatalf("placeholder sign-in: %+v %v", v, err)
	}
	var placeholder bool
	if err := pool.QueryRow(ctx, "select placeholder from surveyors where name = 'beach-walker'").Scan(&placeholder); err != nil || placeholder {
		t.Fatalf("placeholder flag: %v %v", placeholder, err)
	}

	// Profile.
	if _, err := pool.Exec(ctx, `
		insert into regions (admin_level, country, slug, name_en, boundary, min_lat, min_lng, max_lat, max_lng)
		values (4, 'th', 'chon-buri', 'Chon Buri', '{}', 0, 0, 0, 0)`); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateProfile(ctx, v.SurveyorID, "Walking the coast.", "chon-buri"); err != nil {
		t.Fatal(err)
	}
	if bio, home, err := s.Profile(ctx, v.SurveyorID); err != nil || bio != "Walking the coast." || home != "chon-buri" {
		t.Fatalf("profile: %q %q %v", bio, home, err)
	}
	if err := s.UpdateProfile(ctx, v.SurveyorID, "", ""); err != nil {
		t.Fatal(err)
	}
	if bio, home, err := s.Profile(ctx, v.SurveyorID); err != nil || bio != "" || home != "" {
		t.Fatalf("cleared profile: %q %q %v", bio, home, err)
	}
	if err := s.UpdateProfile(ctx, uuid.NewString(), "", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown surveyor: %v", err)
	}
}
