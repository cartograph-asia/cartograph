package osmfile

import (
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/geo"
)

var fixture = filepath.Join("..", "..", "..", "testdata", "pattaya.osm.gz")

func TestFixtureBoundary(t *testing.T) {
	raw, err := ReadBoundaryData(fixture, func(r *osm.Relation) bool { return IsProvince(r, "th") })
	if err != nil {
		t.Fatal(err)
	}
	bs, skipped, err := BuildBoundaries(raw)
	if err != nil || len(skipped) != 0 || len(bs) != 1 {
		t.Fatalf("boundaries %d skipped %v err %v", len(bs), skipped, err)
	}
	en, th := RegionName(Names(bs[0].Tags))
	if en != "Chon Buri" || th != "ชลบุรี" || Slug(en) != "chon-buri" {
		t.Fatalf("names %q %q", en, th)
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
	ps, err := ReadPlaces(fixture, orb.Bound{Min: orb.Point{1, 1}})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Place{}
	for _, p := range ps {
		byName[p.NameEn] = p
	}
	if p, ok := byName["Pattaya"]; !ok || p.Kind != "city" || p.NameTh != "พัทยา" || p.OSMType != "n" {
		t.Fatalf("places %+v", ps)
	}
}

func TestNamesAndSlug(t *testing.T) {
	en, th := Names(osm.Tags{{Key: "name", Value: "บ้านนา"}})
	if en != "" || th != "บ้านนา" {
		t.Fatalf("%q %q", en, th)
	}
	en, _ = Names(osm.Tags{{Key: "name", Value: "บ้านนา"}, {Key: "name:th-Latn", Value: "Ban Na"}})
	if en != "Ban Na" {
		t.Fatal(en)
	}
	for in, want := range map[string]string{"Chon Buri": "chon-buri", "Phra Nakhon Si Ayutthaya": "phra-nakhon-si-ayutthaya", " Bueng  Kan ": "bueng-kan", "Ko-Chang!": "ko-chang"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q", in, got)
		}
	}
}
