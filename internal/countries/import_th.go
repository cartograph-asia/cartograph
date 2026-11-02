package countries

// importTH reproduces the import Thailand had before the settings existed
// (osmfile.Names, RegionName, DistrictName, IsProvince, IsDistrict and
// PlaceKinds); internal/importer's golden test holds it to that.
//
// LandPlots: docs/countries/thailand.md, scanned 2026-11-02 from the
// thailand-latest.osm.pbf downloaded 2026-11-01.
var importTH = Import{
	Extract: "thailand",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "TH",
		Count:       [2]int{77, 77},
		Kind:        "province",
		Strip:       Affixes{EnSuffix: []string{" Province"}, LocalPrefix: []string{"จังหวัด"}},
	},
	District: Level{
		AdminLevels: []int{6},
		Kind:        "district",
		Strip:       Affixes{EnSuffix: []string{" District"}, EnPrefix: []string{"Khet ", "Amphoe "}, LocalPrefix: []string{"เขต", "อำเภอ"}},
	},
	English:      []string{"name:en", "int_name", "name:th-Latn", "name"},
	EnglishNot:   []string{"Thai"},
	Local:        []string{"name:th", "name"},
	LocalScripts: []string{"Thai"},
	SlugSuffix:   false,
	LandPlots:    598_411,
	// The two "Bang Sai" districts of Phra Nakhon Si Ayutthaya were stored
	// as …-bang-sai and …-bang-sai-11024893; a fresh import keeps them so.
	LegacyDuplicateSlug: true,
	LegacyPlaces:        true, // places by geo.NearTolerance, all of the extract's (WP L)
}
