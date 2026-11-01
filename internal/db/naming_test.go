package db

import "testing"

func TestChoosePrefersOwnDistrict(t *testing.T) {
	th := &PlaceNaming{
		UrbanKinds: []string{"quarter", "neighbourhood", "suburb"}, UrbanRadiusKm: 3,
		Tiers: [][]string{{"village", "town"}, {"city"}}, RadiusKm: 5,
	}
	// Candidates with lat 1 lie in the plot's district, lat 2 outside it.
	own := func(lat, lng float64) bool { return lat == 1 }
	c := func(id int64, kind, name string, lat, km float64) placeCandidate {
		return placeCandidate{id: id, kind: kind, nameEn: name, lat: lat, km: km}
	}
	for _, tc := range []struct {
		name  string
		cs    []placeCandidate
		urban bool
		want  string
	}{
		{"nearer khwaeng of the next khet loses", []placeCandidate{
			c(1, "quarter", "Khlong Tan", 2, 0.3),
			c(2, "quarter", "Khlong Tan Nuea", 1, 1.4),
		}, true, "Khlong Tan Nuea"},
		{"own district only within the urban radius", []placeCandidate{
			c(1, "quarter", "Next Door", 2, 0.5),
			c(2, "quarter", "Own Khwaeng", 1, 3.2),
		}, true, "Next Door"},
		{"kind order holds inside the district", []placeCandidate{
			c(1, "suburb", "Own Khet", 1, 0.2),
			c(2, "neighbourhood", "Estate", 1, 0.4),
			c(3, "quarter", "Own Khwaeng", 1, 2.5),
		}, true, "Own Khwaeng"},
		{"a neighbourhood of the own district beats a quarter outside", []placeCandidate{
			c(1, "quarter", "Next Door", 2, 0.5),
			c(2, "neighbourhood", "Estate", 1, 1.0),
		}, true, "Estate"},
		{"no candidate in the district: nearest outside", []placeCandidate{
			c(1, "quarter", "Far Quarter", 2, 2.8),
			c(2, "suburb", "Near Suburb", 2, 0.4),
		}, true, "Far Quarter"},
		{"outside fallback keeps the urban radius", []placeCandidate{
			c(1, "quarter", "Too Far", 2, 3.5),
			c(2, "village", "Ban Near", 2, 1),
		}, true, "Ban Near"},
		{"outside a city the district does not matter", []placeCandidate{
			c(1, "quarter", "Own Khwaeng", 1, 0.1),
			c(2, "village", "Ban Next", 2, 2),
		}, false, "Ban Next"},
	} {
		got := th.choose(tc.cs, tc.urban, own)
		name := ""
		if got != nil {
			name = got.nameEn
		}
		if name != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, name, tc.want)
		}
	}
}
