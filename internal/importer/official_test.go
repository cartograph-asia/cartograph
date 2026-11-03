package importer

import (
	"context"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
)

// regencies are zz's settings with Indonesia's district words and
// Level.OfficialName: the words of two siblings named alike are only in
// official_name.
func regencies() countries.Import {
	imp := zzSettings()
	imp.District = countries.Level{AdminLevels: []int{6}, Kind: "regency",
		KindByAffix:  map[string]string{"Kota ": "city", "Kabupaten ": "regency"},
		Strip:        countries.Affixes{EnPrefix: []string{"Kota ", "Kabupaten "}, LocalPrefix: []string{"Kota ", "Kabupaten "}},
		OfficialName: []string{"official_name:en", "official_name"}}
	return imp
}

// Two districts that collide and carry their words only in official_name
// are stored with those words, in name and slug; the city's kind word comes
// from official_name too, and a second run changes nothing. Without the
// setting the pair ends with the ids, as before.
func TestImportOfficialNames(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	province := testRel{id: 900000201, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "Zed One"}, outer: []rect{{101.10, 12.90, 101.40, 12.98}}}
	district := func(id int64, x float64, kv ...string) testRel {
		return testRel{id: id, tags: append([]string{"admin_level", "6"}, kv...), outer: []rect{{x, 12.91, x + 0.04, 12.97}}}
	}
	rels := []testRel{province,
		district(900000211, 101.11, "name", "Bogor", "official_name", "Kota Bogor"),
		district(900000212, 101.16, "name", "Bogor", "official_name", "Kabupaten Bogor"),
		district(900000213, 101.21, "name", "Surabaya", "official_name", "Kota Surabaya"),
	}
	pbf := writeTestOSM(t, rels, nil)

	// Without the setting: bare names, ids, every district a regency.
	plain := regencies()
	plain.District.OfficialName = nil
	_, report := regionsRun(ctx, t, d, &plain, pbf, true)
	for _, w := range []string{"zed-one-zz-bogor-900000211", "zed-one-zz-bogor-900000212", `district "regency" 3, province NULL 1`} {
		if !strings.Contains(report, w) {
			t.Errorf("without the setting, the report lacks %q:\n%s", w, report)
		}
	}

	imp := regencies()
	regionsRun(ctx, t, d, &imp, pbf, false)
	want := map[int64]string{
		900000211: "zed-one-zz-kota-bogor Kota Bogor NULL in r900000201",
		900000212: "zed-one-zz-kabupaten-bogor Kabupaten Bogor NULL in r900000201",
		900000213: "zed-one-zz-surabaya Surabaya NULL in r900000201",
	}
	got := storedDistricts(ctx, t, d)
	for id, w := range want {
		if got[id] != w {
			t.Errorf("r%d: %q, want %q", id, got[id], w)
		}
	}
	kinds := map[int64]string{}
	rows, err := d.Pool.Query(ctx, `select osm_id, coalesce(kind_name, 'NULL') from regions where country = 'zz' and kind = 'district'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		var k string
		if err := rows.Scan(&id, &k); err != nil {
			t.Fatal(err)
		}
		kinds[id] = k
	}
	rows.Close()
	for id, w := range map[int64]string{900000211: "city", 900000212: "regency", 900000213: "city"} {
		if kinds[id] != w {
			t.Errorf("r%d: kind_name %q, want %q", id, kinds[id], w)
		}
	}

	_, report = regionsRun(ctx, t, d, &imp, pbf, true)
	for _, w := range []string{"Provinces: 1 (0 added, 0 updated, 1 unchanged)", "Districts: 3 (0 added, 0 updated, 3 unchanged", "New slugs: 0"} {
		if !strings.Contains(report, w) {
			t.Errorf("second run's report lacks %q:\n%s", w, report)
		}
	}
	all := sums(t, d, "")
	regionsRun(ctx, t, d, &imp, pbf, false)
	sameSums(t, "after the second run", all, sums(t, d, ""))
}
