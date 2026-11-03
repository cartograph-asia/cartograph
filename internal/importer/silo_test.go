package importer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
)

// plotAt inserts a surveyed plot without region at (lat, lng), as a claim
// before the import leaves it, and returns its cell.
func plotAt(ctx context.Context, t *testing.T, d *db.DB, lat, lng float64) string {
	t.Helper()
	cell, err := h3.CellFromLatLng(lat, lng, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Pool.Exec(ctx, `insert into surveyors (id, name, email) values ('00000000-0000-4000-8000-0000000000b1', 'silo-sam', 'silo@example.test')
		on conflict do nothing`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Pool.Exec(ctx, `insert into plots (cell, first_photo, first_surveyor, first_surveyor_id, claimed_at, photo_count, parent5, parent6)
		values ($1, gen_random_uuid(), 'silo-sam', '00000000-0000-4000-8000-0000000000b1', now(), 1, 'x', 'x')`, cell); err != nil {
		t.Fatal(err)
	}
	return cell
}

// importZZ imports the synthetic country zz over d: a dry run first, whose
// land plots become the settings' LandPlots, then the real run.
func importZZ(ctx context.Context, t *testing.T, d *db.DB) *Summary {
	t.Helper()
	_, waterPath := Fixture()
	zz := zzSettings()
	o := Options{Country: "zz", Settings: &zz, PBF: syntheticZZ(), Water: waterPath, DryRun: true}
	sum, err := Import(ctx, d, o)
	if !errors.Is(err, ErrChecks) || !strings.Contains(err.Error(), "4 land plots") || sum.LandPlots == 0 {
		t.Fatalf("dry run without LandPlots: %+v %v", sum, err)
	}
	zz.LandPlots = sum.LandPlots
	o.DryRun = false
	if sum, err = Import(ctx, d, o); err != nil {
		t.Fatalf("zz: %+v %v", sum, err)
	}
	return sum
}

func TestSiloSyntheticCountry(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	insertGoldenPlots(ctx, t, d)
	zzPlot := plotAt(ctx, t, d, 12.95, 101.30) // in zz's own part
	importThai(ctx, t, d)
	th0 := sums(t, d, "th")

	// The shared cell: inside the fixture's bbox, Chon Buri and zz.
	shared, err := h3.CellFromLatLng(12.94, 100.95, 8)
	if err != nil {
		t.Fatal(err)
	}
	var sharedNumber int
	var sharedCountry string
	if err := d.Pool.QueryRow(ctx, `select n.number, r.country from plot_numbers n join regions r on r.id = n.region_id where n.cell = $1`,
		shared).Scan(&sharedNumber, &sharedCountry); err != nil || sharedCountry != "th" {
		t.Fatalf("shared cell %s before zz: %d %q %v", shared, sharedNumber, sharedCountry, err)
	}

	sum := importZZ(ctx, t, d)
	if sum.Provinces != 1 || sum.Districts != 1 || sum.BorderSkipped == 0 || sum.Places != 1 || sum.NewNumbers != sum.LandPlots {
		t.Errorf("zz summary %+v", sum)
	}
	sameSums(t, "Thai rows after importing zz", th0, sums(t, d, "th"))

	var slug, nameEn, kindName string
	var nameLocal *string
	if err := d.Pool.QueryRow(ctx, `select slug, name_en, name_local, kind_name from regions where osm_id = 900000001`).
		Scan(&slug, &nameEn, &nameLocal, &kindName); err != nil || slug != "zed-one-zz" || nameEn != "Zed One" || nameLocal != nil || kindName != "prefecture" {
		t.Errorf("zz province: %q %q %v %q %v", slug, nameEn, nameLocal, kindName, err)
	}
	if err := d.Pool.QueryRow(ctx, `select slug from regions where osm_id = 900000011 and country = 'zz'`).Scan(&slug); err != nil || slug != "zed-one-zz-zed-east" {
		t.Errorf("zz district: %q %v", slug, err)
	}
	var n int
	if err := d.Pool.QueryRow(ctx, `select count(*) from plot_numbers n join regions r on r.id = n.region_id where n.cell = $1`, shared).Scan(&n); err != nil || n != 1 {
		t.Errorf("shared cell numbered %d times %v", n, err)
	}
	var country string
	if err := d.Pool.QueryRow(ctx, `select p.country, r.country from places p join regions r on r.id = p.region_id where p.osm_id = 900000002`).
		Scan(&country, &sharedCountry); err != nil || country != "zz" || sharedCountry != "zz" {
		t.Errorf("Zed Town: %q in %q %v", country, sharedCountry, err)
	}
	if err := d.Pool.QueryRow(ctx, `select country, name_en from places where osm_id = 7366568645`).Scan(&country, &nameEn); err != nil ||
		country != "th" || nameEn != "Nong Prue" {
		t.Errorf("Nong Prue: %q %q %v", country, nameEn, err)
	}
	if err := d.Pool.QueryRow(ctx, `select r.country from plots p join regions r on r.id = p.region_id where p.cell = $1`, zzPlot).Scan(&country); err != nil || country != "zz" {
		t.Errorf("plot in zz: %q %v", country, err)
	}

	// Thailand re-imported: zz unchanged, the shared cell still Thai with
	// its number, and Thailand unchanged as well.
	zz0 := sums(t, d, "zz")
	importThai(ctx, t, d)
	sameSums(t, "zz rows after re-importing Thailand", zz0, sums(t, d, "zz"))
	sameSums(t, "Thai rows after re-importing Thailand", th0, sums(t, d, "th"))
	var again int
	if err := d.Pool.QueryRow(ctx, `select n.number, r.country from plot_numbers n join regions r on r.id = n.region_id where n.cell = $1`,
		shared).Scan(&again, &sharedCountry); err != nil || again != sharedNumber || sharedCountry != "th" {
		t.Errorf("shared cell after re-import: %d %q %v, want %d th", again, sharedCountry, err, sharedNumber)
	}

	// zz again: nothing changes for either.
	all := sums(t, d, "")
	zz := zzSettings()
	zz.LandPlots = sum.LandPlots
	_, waterPath := Fixture()
	if _, err := Import(ctx, d, Options{Country: "zz", Settings: &zz, PBF: syntheticZZ(), Water: waterPath}); err != nil {
		t.Fatal(err)
	}
	sameSums(t, "all rows after re-importing zz", all, sums(t, d, ""))
}

func TestDryRunWritesNothing(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	osmPath, waterPath := Fixture()
	insertGoldenPlots(ctx, t, d)
	importThai(ctx, t, d)
	before := sums(t, d, "")

	var report bytes.Buffer
	th := Options{PBF: osmPath, Water: waterPath, BBox: FixtureBBox, Country: "th", DryRun: true, Report: &report}
	sum, err := Import(ctx, d, th)
	if err != nil || sum.Result != "dry run ok" || sum.NewNumbers != 0 {
		t.Fatalf("Thai dry run: %+v %v", sum, err)
	}
	for _, want := range []string{"Provinces: 1 (0 added, 0 updated, 1 unchanged)", "Districts: 3 (0 added, 0 updated, 3 unchanged",
		"Places added: 0", "Places updated: 0", "Places removed: 0", "New plot numbers: 0", "2. province count                             skipped"} {
		if !strings.Contains(report.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, report.String())
		}
	}
	if _, err := Regions(ctx, d, th); err != nil {
		t.Fatal(err)
	}
	if _, err := Places(ctx, d, th); err != nil {
		t.Fatal(err)
	}
	zz := zzSettings()
	zz.LandPlots = 400
	report.Reset()
	if _, err := Import(ctx, d, Options{Country: "zz", Settings: &zz, PBF: syntheticZZ(), Water: waterPath, DryRun: true, Report: &report}); err != nil &&
		!errors.Is(err, ErrChecks) {
		t.Fatal(err)
	}
	if !strings.Contains(report.String(), "Provinces added: 1") || !strings.Contains(report.String(), "Places added: 1") {
		t.Errorf("zz report:\n%s", report.String())
	}
	sameSums(t, "after dry runs", before, sums(t, d, ""))

	// With a migration pending, a dry run refuses instead of migrating.
	if _, err := d.Pool.Exec(ctx, `delete from schema_migrations where version = (select max(version) from schema_migrations)`); err != nil {
		t.Fatal(err)
	}
	before = sums(t, d, "")
	for _, run := range []func() error{
		func() error { _, err := Import(ctx, d, th); return err },
		func() error { _, err := Regions(ctx, d, th); return err },
		func() error { _, err := Places(ctx, d, th); return err },
	} {
		if err := run(); err == nil || !strings.Contains(err.Error(), "pending") {
			t.Errorf("pending migration: %v", err)
		}
	}
	sameSums(t, "after refused dry runs", before, sums(t, d, ""))
}

func TestChecksFail(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	importThai(ctx, t, d)
	_, waterPath := Fixture()
	east := rect{101.10, 12.90, 101.40, 12.98}   // outside Chon Buri
	inside := rect{100.92, 12.92, 100.96, 12.95} // inside Chon Buri
	province := func(id int64, iso string, r rect, kv ...string) testRel {
		return testRel{id: id, tags: append([]string{"admin_level", "4", "ISO3166-2", iso}, kv...), outer: []rect{r}}
	}
	dry := func(t *testing.T, imp *countries.Import, pbf string) (*RegionStats, error) {
		t.Helper()
		return Regions(ctx, d, Options{Country: "zz", Settings: imp, PBF: pbf, Water: waterPath, DryRun: true, Report: io.Discard})
	}
	base := zzSettings()
	basePBF := writeTestOSM(t, []testRel{province(900000001, "ZZ-01", east, "name", "Zed One")}, nil)
	st, err := dry(t, &base, basePBF)
	if !errors.Is(err, ErrChecks) || st.LandCells == 0 {
		t.Fatalf("base without LandPlots: %+v %v", st, err)
	}
	base.LandPlots = st.LandCells
	if st, err := dry(t, &base, basePBF); err != nil {
		t.Fatalf("base: %+v %v", st, err)
	}
	failed := func(t *testing.T, st *RegionStats, err error, n int) {
		t.Helper()
		if !errors.Is(err, ErrChecks) || st == nil || len(st.Checks) != 7 || !st.Checks[n-1].Failed() {
			t.Fatalf("check %d did not fail: %v %+v", n, err, st)
		}
	}

	t.Run("1 hold", func(t *testing.T) {
		imp := base
		imp.Hold = "not opened yet"
		st, err := dry(t, &imp, "does-not-exist.osm") // refused before anything is read
		failed(t, st, err, 1)
		if st.Checks[1].Status != "not run" {
			t.Errorf("after a hold: %+v", st.Checks[1])
		}
		if _, err := Places(ctx, d, Options{Country: "zz", Settings: &imp, PBF: "does-not-exist.osm"}); !errors.Is(err, ErrChecks) {
			t.Errorf("places of a held country: %v", err)
		}
	})
	t.Run("2 count", func(t *testing.T) {
		imp := base
		imp.Province.Count = [2]int{2, 3}
		st, err := dry(t, &imp, basePBF)
		failed(t, st, err, 2)
	})
	t.Run("3 slug stored for another relation", func(t *testing.T) {
		imp := base
		imp.SlugSuffix = false
		st, err := dry(t, &imp, writeTestOSM(t, []testRel{province(900000001, "ZZ-01", east, "name", "Chon Buri")}, nil))
		failed(t, st, err, 3)
	})
	t.Run("3 no English name", func(t *testing.T) {
		imp := base
		imp.SlugSuffix = false
		st, err := dry(t, &imp, writeTestOSM(t, []testRel{province(900000001, "ZZ-01", east)}, nil))
		failed(t, st, err, 3)
	})
	t.Run("4 land plots", func(t *testing.T) {
		imp := base
		imp.LandPlots = 2 * base.LandPlots
		st, err := dry(t, &imp, basePBF)
		failed(t, st, err, 4)
	})
	t.Run("5 province without land plots", func(t *testing.T) {
		imp := base
		imp.Province.Count = [2]int{2, 2}
		st, err := dry(t, &imp, writeTestOSM(t, []testRel{
			province(900000001, "ZZ-01", east, "name", "Zed One"),
			province(900000002, "ZZ-02", inside, "name", "Zed Two"), // all its cells are Thai
		}, nil))
		failed(t, st, err, 5)
		if st.BorderSkipped == 0 {
			t.Errorf("no border cells: %+v", st)
		}
	})
	t.Run("6 not assembled", func(t *testing.T) {
		imp := base
		imp.Province.Count = [2]int{2, 2}
		rels := []testRel{
			province(900000001, "ZZ-01", east, "name", "Zed One"),
			{id: 900000002, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-02", "name", "Zed Two"}, outer: []rect{{101.5, 12.9, 101.6, 13.0}}, missing: 1},
		}
		st, err := dry(t, &imp, writeTestOSM(t, rels, nil))
		failed(t, st, err, 6)
		// Listed in Partial, it is built from its closed ring.
		imp.Partial = []int64{900000002}
		imp.LandPlots = 0
		var report bytes.Buffer
		st, err = Regions(ctx, d, Options{Country: "zz", Settings: &imp, PBF: writeTestOSM(t, rels, nil), Water: waterPath, DryRun: true, Report: &report})
		if st == nil || st.Checks[5].Failed() || st.Regions != 2 || !strings.Contains(report.String(), "r900000002: 1 rings built, 0 fragments dropped, 1 member ways missing") {
			t.Errorf("partial: %v %+v\n%s", err, st, report.String())
		}
	})
	t.Run("7 stored for another country", func(t *testing.T) {
		st, err := dry(t, &base, writeTestOSM(t, []testRel{province(1908773, "ZZ-01", east, "name", "Zed One")}, nil))
		failed(t, st, err, 7)
		// A real run refuses as well, before writing.
		all := sums(t, d, "")
		_, err = Regions(ctx, d, Options{Country: "zz", Settings: &base, PBF: writeTestOSM(t, []testRel{province(1908773, "ZZ-01", east, "name", "Zed One")}, nil), Water: waterPath})
		if !errors.Is(err, ErrChecks) {
			t.Errorf("real run: %v", err)
		}
		sameSums(t, "after a refused run", all, sums(t, d, ""))
	})
}
