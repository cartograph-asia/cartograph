package importer

import (
	"context"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
)

// Two synthetic countries from one extract, as Malaysia, Brunei and
// Singapore share one: zz1's province (ZZ1-01) is 100.00–100.10 E,
// 10.00–10.10 N, zz2's (ZZ2-01) lies east of it, 100.10–100.20 E. At 10° N
// 0.01° is about 1.1 km.
func twoCountriesOSM(t *testing.T) string {
	t.Helper()
	return writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ1-01", "name", "Zed One"}, outer: []rect{{100.00, 10.00, 100.10, 10.10}}},
		{id: 2, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ2-01", "name", "Zed Two"}, outer: []rect{{100.10, 10.00, 100.20, 10.10}}},
	}, []testNode{
		{id: 501, lng: 100.05, lat: 10.05, tags: []string{"place", "town", "name", "Alpha Town"}},          // inside zz1
		{id: 502, lng: 100.105, lat: 10.05, tags: []string{"place", "village", "name", "Border Kampong"}},  // inside zz2, 0.55 km from zz1
		{id: 503, lng: 100.05, lat: 10.115, tags: []string{"place", "village", "name", "Beach Hamlet"}},    // in no province, 1.7 km from zz1, 5 km from zz2
		{id: 504, lng: 100.05, lat: 10.40, tags: []string{"place", "village", "name", "Far Village"}},      // 33 km from both
		{id: 505, lng: 100.10, lat: 10.115, tags: []string{"place", "village", "name", "Between Village"}}, // in no province, 1.7 km from both
		{id: 506, lng: 100.15, lat: 10.05, tags: []string{"place", "town", "name", "Bravo Town"}},          // inside zz2, 5.5 km from zz1
	})
}

// zzN are the settings of zz1 or zz2: one province, no districts.
func zzN(n string) countries.Import {
	return countries.Import{
		Extract:      "synthetic-two",
		Province:     countries.Level{AdminLevels: []int{4}, ISOPrefix: "ZZ" + n, Count: [2]int{1, 1}, Kind: "province"},
		English:      []string{"name:en", "name"},
		Local:        []string{"name"},
		LocalScripts: []string{"Latin"},
		SlugSuffix:   true,
	}
}

// importN imports country code from path (clipped, so the count and size
// checks are skipped) and returns the report; dry runs only report.
func importN(ctx context.Context, t *testing.T, d *db.DB, code string, imp countries.Import, path string, dry bool) string {
	t.Helper()
	var report strings.Builder
	_, waterPath := Fixture()
	o := Options{Country: code, Settings: &imp, PBF: path, Water: waterPath, BBox: "99.5,9.5,100.7,10.7", DryRun: dry, Report: &report}
	if sum, err := Import(ctx, d, o); err != nil {
		t.Fatalf("%s: %+v %v\n%s", code, sum, err, report.String())
	}
	return report.String()
}

// placeRows returns the stored places as "id country strong|weak".
func placeRows(ctx context.Context, t *testing.T, d *db.DB) map[int64]string {
	t.Helper()
	rows, err := d.Pool.Query(ctx, `select p.osm_id, p.country, coalesce(r.country, '') from places p left join regions r on r.id = p.region_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var country, regionCountry string
		if err := rows.Scan(&id, &country, &regionCountry); err != nil {
			t.Fatal(err)
		}
		switch regionCountry {
		case "":
			out[id] = country + " weak"
		case country:
			out[id] = country + " strong"
		default:
			out[id] = country + " in a province of " + regionCountry
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func samePlaces(t *testing.T, what string, got, want map[int64]string) {
	t.Helper()
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: place %d is %q, want %q", what, id, got[id], w)
		}
	}
	for id, g := range got {
		if _, ok := want[id]; !ok {
			t.Errorf("%s: place %d stored as %q, want none", what, id, g)
		}
	}
}

func wantLines(t *testing.T, what, report string, lines ...string) {
	t.Helper()
	for _, l := range lines {
		if !strings.Contains(report, l) {
			t.Errorf("%s: the report lacks %q:\n%s", what, l, report)
		}
	}
}

// Strong, weak and far places; a weak place passes to the country whose
// province holds it, a place weak for both stays with the first; re-runs of
// either change nothing.
func TestPlacesStrongWeakFar(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	path := twoCountriesOSM(t)

	report := importN(ctx, t, d, "zz1", zzN("1"), path, false)
	wantLines(t, "zz1", report,
		"stored for zz1: 4 (1 strong, inside one of its provinces; 3 weak, outside every province and within 2 km of one of its provinces)",
		"Places not stored: 0 inside another country's province, 0 stored for another country inside its provinces, 0 weak and stored weak for another country, 2 farther than 2 km")
	samePlaces(t, "after zz1", placeRows(ctx, t, d), map[int64]string{
		501: "zz1 strong", 502: "zz1 weak", 503: "zz1 weak", 505: "zz1 weak"})

	report = importN(ctx, t, d, "zz2", zzN("2"), path, false)
	wantLines(t, "zz2", report,
		"stored for zz2: 2 (2 strong, inside one of its provinces; 0 weak",
		"Places not stored: 1 inside another country's province, 0 stored for another country inside its provinces, 1 weak and stored weak for another country, 2 farther than 2 km",
		"Places taken over (weak for another country, strong for zz2): 1\n  - n502 \"Border Kampong\" from zz1",
		"Places added: 1", "Places updated: 1")
	after := map[int64]string{501: "zz1 strong", 502: "zz2 strong", 503: "zz1 weak", 505: "zz1 weak", 506: "zz2 strong"}
	samePlaces(t, "after zz2", placeRows(ctx, t, d), after)

	// Re-runs: nothing to change, dry or real, in either order.
	all := sums(t, d, "")
	for _, code := range []string{"zz1", "zz2", "zz1"} {
		report := importN(ctx, t, d, code, zzN(code[2:]), path, true)
		wantLines(t, code+" again", report, "Places taken over (weak for another country, strong for "+code+"): 0",
			"Places added: 0", "Places updated: 0", "Places removed: 0")
		importN(ctx, t, d, code, zzN(code[2:]), path, false)
		sameSums(t, code+" again", all, sums(t, d, ""))
	}
	samePlaces(t, "after the re-runs", placeRows(ctx, t, d), after)
}

// A place another country stored inside one of its provinces is never
// taken, also when it lies in the run's: zz1 with the legacy rule stores
// Border Kampong in its province through geo.NearTolerance.
func TestPlacesLegacyAndStrongNeverTaken(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	path := twoCountriesOSM(t)
	legacy := zzN("1")
	legacy.LegacyPlaces = true
	legacy.PlaceReachKm = 50 // not read under the legacy rule

	report := importN(ctx, t, d, "zz1", legacy, path, false)
	wantLines(t, "zz1 legacy", report, "stored for zz1: 6 (2 in or within 0.01° of one of its provinces, 4 outside every province; legacy rule)")
	samePlaces(t, "after zz1", placeRows(ctx, t, d), map[int64]string{
		501: "zz1 strong", 502: "zz1 strong", 503: "zz1 weak", 504: "zz1 weak", 505: "zz1 weak", 506: "zz1 weak"})

	report = importN(ctx, t, d, "zz2", zzN("2"), path, false)
	wantLines(t, "zz2", report,
		"Places not stored: 1 inside another country's province, 1 stored for another country inside its provinces, 1 weak and stored weak for another country, 2 farther than 2 km",
		"Places taken over (weak for another country, strong for zz2): 1\n  - n506 \"Bravo Town\" from zz1")
	after := map[int64]string{501: "zz1 strong", 502: "zz1 strong", 503: "zz1 weak", 504: "zz1 weak", 505: "zz1 weak", 506: "zz2 strong"}
	samePlaces(t, "after zz2", placeRows(ctx, t, d), after)

	// The legacy run again, as before WP L: Border Kampong lies strictly in
	// zz2's stored province now, so zz1 drops it; zz2's next run takes it.
	report = importN(ctx, t, d, "zz1", legacy, path, false)
	wantLines(t, "zz1 legacy again", report, "Places not stored: 2 inside another country's province",
		"Places added: 0", "Places updated: 0", "Places removed: 1\n  - n502")
	importN(ctx, t, d, "zz2", zzN("2"), path, false)
	after[502] = "zz2 strong"
	samePlaces(t, "after zz1 and zz2 again", placeRows(ctx, t, d), after)

	// From here on, nothing changes.
	all := sums(t, d, "")
	for _, code := range []string{"zz1", "zz2"} {
		imp := zzN(code[2:])
		if code == "zz1" {
			imp = legacy
		}
		report := importN(ctx, t, d, code, imp, path, true)
		wantLines(t, code+" once more", report, "Places added: 0", "Places updated: 0", "Places removed: 0")
		importN(ctx, t, d, code, imp, path, false)
		sameSums(t, code+" once more", all, sums(t, d, ""))
	}
}

// PlaceReachKm decides how far a weak place may lie; a place no longer in
// reach is removed from the run's country.
func TestPlacesReach(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	path := twoCountriesOSM(t)
	imp := zzN("1")

	imp.PlaceReachKm = 40
	importN(ctx, t, d, "zz1", imp, path, false)
	samePlaces(t, "reach 40 km", placeRows(ctx, t, d), map[int64]string{
		501: "zz1 strong", 502: "zz1 weak", 503: "zz1 weak", 504: "zz1 weak", 505: "zz1 weak", 506: "zz1 weak"})

	imp.PlaceReachKm = 1
	report := importN(ctx, t, d, "zz1", imp, path, false)
	wantLines(t, "reach 1 km", report, "within 1 km of one of its provinces", "4 farther than 1 km", "Places removed: 4")
	samePlaces(t, "reach 1 km", placeRows(ctx, t, d), map[int64]string{501: "zz1 strong", 502: "zz1 weak"})
}
