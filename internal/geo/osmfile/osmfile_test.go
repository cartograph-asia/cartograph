package osmfile

import (
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo"
)

var fixture = filepath.Join("..", "..", "..", "testdata", "pattaya.osm.gz")

func TestFixtureBoundary(t *testing.T) {
	th := countries.Imports["th"]
	raw, err := ReadBoundaryData(fixture, func(r *osm.Relation) bool { return MatchLevel(r, th.Province) })
	if err != nil {
		t.Fatal(err)
	}
	bs, skipped, err := BuildBoundaries(raw)
	if err != nil || len(skipped) != 0 || len(bs) != 1 {
		t.Fatalf("boundaries %d skipped %v err %v", len(bs), skipped, err)
	}
	n := LevelNames(bs[0].Tags, th, th.Province)
	if n.En != "Chon Buri" || n.Local != "ชลบุรี" || Slug(n.En) != "chon-buri" {
		t.Fatalf("names %+v", n)
	}
	pattaya := orb.Point{100.8826, 12.9346}
	if !planar.MultiPolygonContains(bs[0].Geometry, pattaya) {
		t.Fatal("Pattaya is not in Chon Buri")
	}
	simple := geo.Simplify(bs[0].Geometry, geo.SimplifyTolerance)
	if !planar.MultiPolygonContains(simple, pattaya) {
		t.Fatal("Pattaya is not in simplified Chon Buri")
	}
}

func TestFixturePlaces(t *testing.T) {
	ps, err := ReadPlaces(fixture, orb.Bound{Min: orb.Point{1, 1}}, countries.Imports["th"])
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Place{}
	for _, p := range ps {
		byName[p.NameEn] = p
	}
	if p, ok := byName["Pattaya"]; !ok || p.Kind != "city" || p.NameLocal != "พัทยา" || p.OSMType != "n" {
		t.Fatalf("places %+v", ps)
	}
}

func TestNamesAndSlug(t *testing.T) {
	th := countries.Imports["th"]
	en, local := NamesFor(osm.Tags{{Key: "name", Value: "บ้านนา"}}, th)
	if en != "" || local != "บ้านนา" {
		t.Fatalf("%q %q", en, local)
	}
	en, _ = NamesFor(osm.Tags{{Key: "name", Value: "บ้านนา"}, {Key: "name:th-Latn", Value: "Ban Na"}}, th)
	if en != "Ban Na" {
		t.Fatal(en)
	}
	for in, want := range map[string]string{"Chon Buri": "chon-buri", "Phra Nakhon Si Ayutthaya": "phra-nakhon-si-ayutthaya", " Bueng  Kan ": "bueng-kan", "Ko-Chang!": "ko-chang"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q", in, got)
		}
	}
}

func TestDistrictName(t *testing.T) {
	for _, c := range []struct {
		en, th, wantEn, wantTh string
	}{
		{"Bang Kapi District", "เขตบางกะปิ", "Bang Kapi", "บางกะปิ"},
		{"Ban Khok District", "อำเภอบ้านโคก", "Ban Khok", "บ้านโคก"},
		{"Khet Watthana", "", "Watthana", ""},
		{"Mueang Roi Et", "อำเภอเมืองร้อยเอ็ด", "Mueang Roi Et", "เมืองร้อยเอ็ด"},
	} {
		en, th := StripName(c.en, c.th, countries.Imports["th"].District)
		if en != c.wantEn || th != c.wantTh {
			t.Errorf("StripName(%q, %q) = %q, %q", c.en, c.th, en, th)
		}
	}
}
