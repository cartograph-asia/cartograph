package countries

// importLA opens Laos, the first country after Thailand
// (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP G). Read from the tags of
// laos-latest.osm.pbf (2026-11-01) and its cut:
//
//   - Provinces: the 18 level-4 relations with ISO 3166-2 LA-xx, every one
//     with name:en and name:lo. Nine name:en end in " Province", which goes;
//     "Vientiane Prefecture" (LA-VT, the capital) keeps its word, so it does
//     not collide with "Vientiane Province" (LA-VI, now "Vientiane",
//     vientiane-laos), and is a prefecture by that word.
//   - Districts: the 148 level-6 relations of the country (none has an ISO
//     code), every name:en ending in " District"; 111 local names start with
//     ເມືອງ (muang, "district"), which goes. 142 have name:lo, the six others
//     a Lao name; Sisattanak's name is Latin (Sisattanāk) and its name:lo is
//     used. One name has an accent: "Pak Sé District" → pak-se.
//   - English: name:en, then int_name, name:lo-Latn and name, as the scan
//     read them. Not taken as English: names in Lao, or in the scripts of the
//     neighbours whose villages the extract holds beyond the border (Thai,
//     Han, Khmer, Myanmar); such a place has no English name and is skipped.
//   - Places: the default six kinds; 457 hamlets (93 named in English) stay
//     out as they do in Thailand.
//
// LandPlots: docs/countries/laos.md, scanned 2026-11-02 from the
// laos-latest.osm.pbf downloaded 2026-11-01.
var importLA = Import{
	Extract: "laos",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "LA",
		Count:       [2]int{18, 18},
		Kind:        "province",
		KindByAffix: map[string]string{"Prefecture": "prefecture"},
		Strip:       Affixes{EnSuffix: []string{" Province"}},
	},
	District: Level{
		AdminLevels: []int{6},
		Kind:        "district",
		Strip:       Affixes{EnSuffix: []string{" District"}, LocalPrefix: []string{"ເມືອງ"}},
	},
	English:      []string{"name:en", "int_name", "name:lo-Latn", "name"},
	EnglishNot:   []string{"Lao", "Thai", "Han", "Khmer", "Myanmar"},
	Local:        []string{"name:lo", "name"},
	LocalScripts: []string{"Lao"},
	LocalLang:    "lo",
	SlugSuffix:   true,
	LandPlots:    275_740,
}
