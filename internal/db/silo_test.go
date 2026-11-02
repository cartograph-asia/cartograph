package db

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/migrations"
)

// The silo tests use a synthetic second country, "zz", which is not in
// countries.All: its province overlaps the test region (Chon Buri,
// 100.5–101.5 E) by 0.1°.
var zzSquare = orb.MultiPolygon{{{{101.4, 12.5}, {102.4, 12.5}, {102.4, 13.5}, {101.4, 13.5}, {101.4, 12.5}}}}

func zzProvince() Region {
	return Region{OSMID: 2, Kind: "province", KindName: "prefecture", AdminLevel: 4, Country: "zz", Slug: "zz-one",
		NameEn: "Zed One", Boundary: zzSquare, LandCells: 50}
}

// rowsOf returns every row of table matching where as text, in order and
// without the generated id, so that two calls compare a country's rows
// column by column.
func rowsOf(t *testing.T, d *DB, table, where, order string) string {
	t.Helper()
	var s *string
	err := d.Pool.QueryRow(context.Background(),
		`select string_agg((to_jsonb(x) - 'id')::text, E'\n' order by `+order+`) from `+table+` x where `+where).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		return ""
	}
	return *s
}

func TestRegionsSilo(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	thID := testRegion(t, d)
	thRows := func() string { return rowsOf(t, d, "regions", "country = 'th'", "id") }
	zzRows := func() string { return rowsOf(t, d, "regions", "country = 'zz'", "id") }
	th0 := thRows()

	// A second country's province leaves the first untouched.
	zz := zzProvince()
	ids, err := d.UpsertRegions(ctx, []Region{zz})
	if err != nil {
		t.Fatal(err)
	}
	zzID := ids[0]
	if got := thRows(); got != th0 {
		t.Fatalf("Thai regions changed by zz:\n%s\n%s", th0, got)
	}
	var kindName *string
	if err := d.Pool.QueryRow(ctx, `select kind_name from regions where id = $1`, thID).Scan(&kindName); err != nil || kindName != nil {
		t.Fatalf("Thai kind_name %v %v, want NULL", kindName, err)
	}
	if err := d.Pool.QueryRow(ctx, `select kind_name from regions where id = $1`, zzID).Scan(&kindName); err != nil || kindName == nil || *kindName != "prefecture" {
		t.Fatalf("zz kind_name %v %v", kindName, err)
	}

	// The region page shows the kind word, else the kind.
	v := d.Views()
	if p, err := v.Region(ctx, "zz-one", 1); err != nil || p.Kind != "prefecture" || p.KindName() != "Prefecture" {
		t.Fatalf("zz page: %+v %v", p, err)
	}
	if p, err := v.Region(ctx, "chon-buri", 1); err != nil || p.Kind != "province" {
		t.Fatalf("Chon Buri page: %+v %v", p, err)
	}

	// A relation stored for another country fails the whole call: the new
	// zz district before it is not written either.
	z0 := zzRows()
	stolen := Region{OSMID: 1, Kind: "province", AdminLevel: 4, Country: "zz", Slug: "zz-stolen", NameEn: "Stolen",
		Boundary: zzSquare, LandCells: 1}
	district := Region{OSMID: 3, Kind: "district", AdminLevel: 6, Country: "zz", ParentID: &zzID, Slug: "zz-one-a",
		NameEn: "A", Boundary: zzSquare}
	_, err = d.UpsertRegions(ctx, []Region{district, stolen})
	if !errors.Is(err, ErrRegionOtherCountry) || !strings.Contains(err.Error(), `"th"`) {
		t.Fatalf("foreign relation: %v", err)
	}
	if got := thRows(); got != th0 {
		t.Fatalf("Thai regions changed by a refused run:\n%s\n%s", th0, got)
	}
	if got := zzRows(); got != z0 {
		t.Fatalf("zz regions changed by a refused run:\n%s\n%s", z0, got)
	}
	// The other way round as well.
	foreign := zz
	foreign.Country = "th"
	if _, err := d.UpsertRegions(ctx, []Region{foreign}); !errors.Is(err, ErrRegionOtherCountry) {
		t.Fatalf("zz relation as th: %v", err)
	}

	// Re-importing Thailand with a renamed relation keeps the stored slug,
	// updates the name and leaves zz alone.
	renamed := Region{OSMID: 1, Kind: "province", AdminLevel: 4, Country: "th", Slug: "chonburi", NameEn: "Chonburi",
		NameLocal: "ชลบุรี", Boundary: orb.MultiPolygon{{{{100.5, 12.5}, {101.5, 12.5}, {101.5, 13.5}, {100.5, 13.5}, {100.5, 12.5}}}}, LandCells: 100}
	ids, err = d.UpsertRegions(ctx, []Region{renamed})
	if err != nil || ids[0] != thID {
		t.Fatalf("re-import: %v %v", ids, err)
	}
	var slug, name string
	if err := d.Pool.QueryRow(ctx, `select slug, name_en from regions where id = $1`, thID).Scan(&slug, &name); err != nil ||
		slug != "chon-buri" || name != "Chonburi" {
		t.Fatalf("after rename: slug %q name %q %v", slug, name, err)
	}
	if got := zzRows(); got != z0 {
		t.Fatalf("zz regions changed by the Thai re-import:\n%s\n%s", z0, got)
	}

	// Re-importing zz without a kind word clears it (the word follows the
	// settings); its slug stays too.
	zz.KindName, zz.Slug = "", "zz-renamed"
	if _, err := d.UpsertRegions(ctx, []Region{zz}); err != nil {
		t.Fatal(err)
	}
	if err := d.Pool.QueryRow(ctx, `select slug, kind_name from regions where id = $1`, zzID).Scan(&slug, &kindName); err != nil ||
		slug != "zz-one" || kindName != nil {
		t.Fatalf("zz re-import: slug %q kind_name %v %v", slug, kindName, err)
	}
}

func TestPlacesSilo(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	thID := testRegion(t, d)
	ids, err := d.UpsertRegions(ctx, []Region{zzProvince()})
	if err != nil {
		t.Fatal(err)
	}
	zzID := ids[0]
	thRows := func() string { return rowsOf(t, d, "places", "country = 'th'", "osm_type, osm_id") }
	zzRows := func() string { return rowsOf(t, d, "places", "country = 'zz'", "osm_type, osm_id") }
	country := func(id int64) string {
		t.Helper()
		var c string
		if err := d.Pool.QueryRow(ctx, `select country from places where osm_type = 'n' and osm_id = $1`, id).Scan(&c); err != nil {
			t.Fatalf("place %d: %v", id, err)
		}
		return c
	}
	count := func() int {
		var n int
		if err := d.Pool.QueryRow(ctx, `select count(*) from places`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	pattaya := Place{OSMType: "n", OSMID: 10, Kind: "city", NameEn: "Pattaya", NameLocal: "พัทยา", Lat: 12.93, Lng: 100.88, RegionID: &thID}
	beyond := Place{OSMType: "n", OSMID: 11, Kind: "village", NameEn: "Ban Beyond", Lat: 14, Lng: 102} // in no province
	if err := d.ReplacePlaces(ctx, "th", []Place{pattaya, beyond}); err != nil {
		t.Fatal(err)
	}
	if country(10) != "th" || country(11) != "th" {
		t.Fatal("Thai places not stored as th")
	}
	th0 := thRows()

	// zz's run: its own place, Pattaya (which the zz extract holds as well,
	// inside zz's province by its shape) and the Thai place outside every
	// province. Pattaya stays Thai; the village outside is taken over.
	zzTown := Place{OSMType: "n", OSMID: 20, Kind: "town", NameEn: "Zed Town", Lat: 13, Lng: 102, RegionID: &zzID, Country: "zz"}
	pattayaZZ := pattaya
	pattayaZZ.NameEn, pattayaZZ.RegionID = "Pattaya (zz)", &zzID
	beyondZZ := beyond
	beyondZZ.RegionID = &zzID
	if err := d.ReplacePlaces(ctx, "zz", []Place{zzTown, pattayaZZ, beyondZZ}); err != nil {
		t.Fatal(err)
	}
	if country(10) != "th" || country(11) != "zz" || country(20) != "zz" {
		t.Fatalf("after zz: pattaya %s, beyond %s, zed town %s", country(10), country(11), country(20))
	}
	var name string
	if err := d.Pool.QueryRow(ctx, `select name_en from places where osm_id = 10`).Scan(&name); err != nil || name != "Pattaya" {
		t.Fatalf("Pattaya renamed by zz: %q %v", name, err)
	}
	z0 := zzRows()

	// Thailand's re-import: Pattaya only. It deletes nothing of zz's, and
	// cannot take the village back, since zz stored it inside a province.
	if err := d.ReplacePlaces(ctx, "th", []Place{pattaya, beyond}); err != nil {
		t.Fatal(err)
	}
	if got := zzRows(); got != z0 {
		t.Fatalf("zz places changed by the Thai re-import:\n%s\n%s", z0, got)
	}
	if err := d.ReplacePlaces(ctx, "th", []Place{pattaya}); err != nil {
		t.Fatal(err)
	}
	if got := zzRows(); got != z0 || count() != 3 {
		t.Fatalf("zz places changed by a Thai run without them (%d places):\n%s\n%s", count(), z0, got)
	}

	// An empty zz run deletes only zz's places.
	if err := d.ReplacePlaces(ctx, "zz", nil); err != nil {
		t.Fatal(err)
	}
	if zzRows() != "" || count() != 1 || country(10) != "th" {
		t.Fatalf("after the empty zz run: %d places, zz %q", count(), zzRows())
	}
	if err := d.ReplacePlaces(ctx, "th", []Place{pattaya, beyond}); err != nil {
		t.Fatal(err)
	}
	if got := thRows(); got != th0 {
		t.Fatalf("Thai places differ after the round trip:\n%s\n%s", th0, got)
	}

	// A run's places are its own; a missing country is refused.
	if err := d.ReplacePlaces(ctx, "th", []Place{zzTown}); err == nil || !strings.Contains(err.Error(), `"zz"`) {
		t.Fatalf("zz place in a Thai run: %v", err)
	}
	if err := d.ReplacePlaces(ctx, "", []Place{pattaya}); err == nil {
		t.Fatal("empty country accepted")
	}
	if got := thRows(); got != th0 {
		t.Fatalf("Thai places changed by refused runs:\n%s\n%s", th0, got)
	}
}

// TestMigrationPlacesCountry applies the migrations before 0150 on rows a
// Thai import (and, synthetically, a second country) left, then the rest.
func TestMigrationPlacesCountry(t *testing.T) {
	d := emptyDB(t)
	ctx := context.Background()
	before := fstest.MapFS{}
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f < "0150" {
			b, err := fs.ReadFile(migrations.FS, f)
			if err != nil {
				t.Fatal(err)
			}
			before[f] = &fstest.MapFile{Data: b}
		}
	}
	if err := d.Migrate(ctx, before); err != nil {
		t.Fatal(err)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := d.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	box := `'{"type":"Polygon","coordinates":[[[100,12],[101,12],[101,13],[100,13],[100,12]]]}'`
	exec(`insert into regions (id, osm_id, kind, admin_level, country, slug, name_en, boundary, min_lat, min_lng, max_lat, max_lng)
		values (1, 1, 'province', 4, 'th', 'chon-buri', 'Chon Buri', ` + box + `, 12, 100, 13, 101),
		       (2, 2, 'province', 4, 'zz', 'zz-one', 'Zed One', ` + box + `, 12, 100, 13, 101)`)
	exec(`insert into places (osm_type, osm_id, kind, name_en, lat, lng, region_id) values
		('n', 10, 'city', 'Pattaya', 12.9, 100.9, 1),
		('n', 11, 'village', 'Ban Beyond', 14, 102, null),
		('n', 20, 'town', 'Zed Town', 12.5, 100.5, 2)`)

	if err := d.Migrate(ctx, migrations.FS); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Pool.Query(ctx, `select osm_id, country from places order by osm_id`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var id int64
		var c string
		if err := rows.Scan(&id, &c); err != nil {
			t.Fatal(err)
		}
		got = append(got, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"th", "th", "zz"}; !slices.Equal(got, want) {
		t.Fatalf("backfilled countries %v, want %v", got, want)
	}
	var idx bool
	if err := d.Pool.QueryRow(ctx, `select exists (select 1 from pg_indexes
		where schemaname = current_schema() and indexname = 'places_country_idx')`).Scan(&idx); err != nil || !idx {
		t.Fatalf("places_country_idx: %v %v", idx, err)
	}
	if _, err := d.Pool.Exec(ctx, `insert into places (osm_type, osm_id, kind, name_en, lat, lng) values ('n', 30, 'town', 'X', 0, 0)`); err == nil {
		t.Fatal("a place without country was accepted")
	}
	// 0151 adds kind_name and leaves every stored region NULL.
	var set int
	if err := d.Pool.QueryRow(ctx, `select count(*) from regions where kind_name is not null`).Scan(&set); err != nil || set != 0 {
		t.Fatalf("kind_name set on %d regions: %v", set, err)
	}
}

func TestOpenCountries(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	codes := func() []string {
		t.Helper()
		cs, err := d.OpenCountries(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range cs {
			out = append(out, c.Code)
		}
		return out
	}
	// Nothing imported: loaded on first use, empty.
	if got := codes(); len(got) != 0 {
		t.Fatalf("no regions: %v", got)
	}
	sq := func(lng float64) orb.MultiPolygon {
		return orb.MultiPolygon{{{{lng, 10}, {lng + 1, 10}, {lng + 1, 11}, {lng, 11}, {lng, 10}}}}
	}
	if _, err := d.UpsertRegions(ctx, []Region{
		{OSMID: 11, Kind: "province", AdminLevel: 4, Country: "vn", Slug: "ha-noi", NameEn: "Ha Noi", Boundary: sq(105)},
		{OSMID: 12, Kind: "province", AdminLevel: 4, Country: "th", Slug: "chon-buri", NameEn: "Chon Buri", Boundary: sq(100)},
		{OSMID: 13, Kind: "province", AdminLevel: 4, Country: "la", Slug: "vientiane-laos", NameEn: "Vientiane", Boundary: sq(102)},
		{OSMID: 14, Kind: "province", AdminLevel: 4, Country: "zz", Slug: "zz-one", NameEn: "Zed One", Boundary: sq(120)},
		{OSMID: 15, Kind: "country", AdminLevel: 2, Country: "kh", Slug: "cambodia", NameEn: "Cambodia", Boundary: sq(104)},
	}); err != nil {
		t.Fatal(err)
	}
	// Cached until the regions are reloaded.
	if got := codes(); len(got) != 0 {
		t.Fatalf("before reload: %v", got)
	}
	if err := d.ReloadRegions(ctx); err != nil {
		t.Fatal(err)
	}
	// List order; zz (not in the list) and kh (no province) left out.
	if got, want := codes(), []string{"la", "th", "vn"}; !slices.Equal(got, want) {
		t.Fatalf("open countries %v, want %v", got, want)
	}
	if _, ok := countries.ByCode("zz"); ok {
		t.Fatal("zz must not be in countries.All")
	}
}
