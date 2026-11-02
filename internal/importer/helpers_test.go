package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
)

// rect is an axis-parallel ring, minLng, minLat, maxLng, maxLat.
type rect [4]float64

// testRel is a boundary relation for writeTestOSM: one closed outer way per
// rect, plus missing member ways that are not written.
type testRel struct {
	id      int64
	tags    []string // key, value, …
	outer   []rect
	missing int
}

// testNode is a place node for writeTestOSM.
type testNode struct {
	id       int64
	lng, lat float64
	tags     []string
}

// writeTestOSM writes an OSM XML file with the relations and nodes and
// returns its path.
func writeTestOSM(t *testing.T, rels []testRel, nodes []testNode) string {
	t.Helper()
	var ns, ws, rs strings.Builder
	nodeID, wayID := int64(800_000_000), int64(810_000_000)
	tagXML := func(b *strings.Builder, kv []string) {
		for i := 0; i+1 < len(kv); i += 2 {
			fmt.Fprintf(b, "  <tag k=%q v=%q/>\n", kv[i], kv[i+1])
		}
	}
	for _, r := range rels {
		var members []int64
		for _, q := range r.outer {
			var ids []int64
			for _, p := range [][2]float64{{q[0], q[1]}, {q[2], q[1]}, {q[2], q[3]}, {q[0], q[3]}} {
				nodeID++
				fmt.Fprintf(&ns, " <node id=\"%d\" visible=\"true\" version=\"1\" lat=\"%.7f\" lon=\"%.7f\"/>\n", nodeID, p[1], p[0])
				ids = append(ids, nodeID)
			}
			wayID++
			fmt.Fprintf(&ws, " <way id=\"%d\" visible=\"true\" version=\"1\">\n", wayID)
			for _, id := range append(ids, ids[0]) {
				fmt.Fprintf(&ws, "  <nd ref=\"%d\"/>\n", id)
			}
			ws.WriteString(" </way>\n")
			members = append(members, wayID)
		}
		for i := 0; i < r.missing; i++ {
			members = append(members, 890_000_000+r.id*10+int64(i))
		}
		fmt.Fprintf(&rs, " <relation id=\"%d\" visible=\"true\" version=\"1\">\n", r.id)
		for _, m := range members {
			fmt.Fprintf(&rs, "  <member type=\"way\" ref=\"%d\" role=\"outer\"/>\n", m)
		}
		tagXML(&rs, append([]string{"boundary", "administrative", "type", "boundary"}, r.tags...))
		rs.WriteString(" </relation>\n")
	}
	for _, n := range nodes {
		fmt.Fprintf(&ns, " <node id=\"%d\" visible=\"true\" version=\"1\" lat=\"%.7f\" lon=\"%.7f\">\n", n.id, n.lat, n.lng)
		tagXML(&ns, n.tags)
		ns.WriteString(" </node>\n")
	}
	path := filepath.Join(t.TempDir(), "test.osm")
	doc := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<osm version=\"0.6\" generator=\"test\">\n" + ns.String() + ws.String() + rs.String() + "</osm>\n"
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// zzSettings are the settings of the synthetic country zz
// (testdata/synthetic-zz.osm): it is not in countries.All, so tests inject
// them through Options.Settings.
func zzSettings() countries.Import {
	return countries.Import{
		Extract: "synthetic-zz",
		Province: countries.Level{AdminLevels: []int{4}, ISOPrefix: "ZZ", Count: [2]int{1, 1}, Kind: "province",
			KindByAffix: map[string]string{"Prefecture": "prefecture"}, Strip: countries.Affixes{EnSuffix: []string{" Prefecture"}}},
		District:     countries.Level{AdminLevels: []int{6}, Kind: "district", Strip: countries.Affixes{EnSuffix: []string{" District"}}},
		English:      []string{"name:en", "name"},
		Local:        []string{"name:zz", "name"},
		LocalScripts: []string{"Latin"},
		SlugSuffix:   true,
	}
}

// syntheticZZ is testdata/synthetic-zz.osm.
func syntheticZZ() string { return filepath.Join(filepath.Dir(goldenPath("x")), "..", "synthetic-zz.osm") }

// importThai imports the Pattaya fixture as docker compose does.
func importThai(ctx context.Context, t *testing.T, d *db.DB) {
	t.Helper()
	osmPath, waterPath := Fixture()
	if _, err := Regions(ctx, d, RegionOptions{PBF: osmPath, Water: waterPath, BBox: FixtureBBox, Country: "th"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Places(ctx, d, PlaceOptions{PBF: osmPath, BBox: FixtureBBox}); err != nil {
		t.Fatal(err)
	}
}

// sums returns "count:md5" of every table an import writes, in a fixed
// order (ids included, so a renumbering shows), for one country's rows or,
// with country "", all rows and schema_migrations.
func sums(t *testing.T, d *db.DB, country string) map[string]string {
	t.Helper()
	ctx := context.Background()
	qs := map[string]string{
		"regions":      `select count(*), md5(coalesce(string_agg(to_jsonb(x)::text, E'\n' order by x.id), '')) from regions x where $1 = '' or x.country = $1`,
		"places":       `select count(*), md5(coalesce(string_agg(to_jsonb(x)::text, E'\n' order by x.osm_type, x.osm_id), '')) from places x where $1 = '' or x.country = $1`,
		"plot_numbers": `select count(*), md5(coalesce(string_agg(to_jsonb(x)::text, E'\n' order by x.cell), '')) from plot_numbers x where $1 = '' or x.region_id in (select id from regions where country = $1)`,
		"plots":        `select count(*), md5(coalesce(string_agg(to_jsonb(x)::text, E'\n' order by x.cell), '')) from plots x where $1 = '' or x.region_id in (select id from regions where country = $1)`,
	}
	if country == "" {
		qs["schema_migrations"] = `select count(*), md5(coalesce(string_agg(version, E'\n' order by version), '')) from schema_migrations where $1 = ''`
	}
	out := map[string]string{}
	for name, sql := range qs {
		var n int
		var sum string
		if err := d.Pool.QueryRow(ctx, sql, country).Scan(&n, &sum); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out[name] = fmt.Sprintf("%d:%s", n, sum)
	}
	return out
}

func sameSums(t *testing.T, what string, want, got map[string]string) {
	t.Helper()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %s changed: %s → %s", what, k, v, got[k])
		}
	}
}
