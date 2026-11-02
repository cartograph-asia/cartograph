package countries

// importBN opens Brunei (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H, decision
// 6), from malaysia-singapore-brunei-latest.osm.pbf (2026-11-01), imported
// after Malaysia and before Singapore. Read from the tags of its cut:
//
//   - Provinces: the 4 level-4 relations with ISO 3166-2 BN-xx, Brunei's
//     districts (daerah: Belait, Brunei-Muara, Temburong, Tutong); the word
//     is "district". name:en ends in " District" on all four and name:ms
//     starts with "Daerah " on all four; both go, so name and name:ms equal
//     the English name and the local name is NULL ("Belait",
//     belait-brunei). name is already the bare name.
//   - No district level: the level-6 mukim cover 25 % of the land plots in
//     the extract and the level-8 villages 5 %; no level below 4 tiles the
//     country.
//   - English: name:en, int_name, name; never a name in Arabic (Jawi), Han,
//     Tamil or Thai script. Local: name:ms, else a Latin name; LocalLang
//     "ms". Every name is Latin; Jawi is in name:ms-Arab, not read.
//   - Places: the default six kinds; 23 hamlets stay out as in Thailand.
//
// LandPlots: docs/countries/brunei.md, scanned 2026-11-02 from the
// malaysia-singapore-brunei-latest.osm.pbf downloaded 2026-11-01.
var importBN = Import{
	Extract: "malaysia-singapore-brunei",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "BN",
		Count:       [2]int{4, 4},
		Kind:        "district",
		Strip:       Affixes{EnSuffix: []string{" District"}, LocalPrefix: []string{"Daerah "}},
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Arabic", "Han", "Tamil", "Thai"},
	Local:        []string{"name:ms", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "ms",
	SlugSuffix:   true,
	LandPlots:    7_864,
}
