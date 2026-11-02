package countries

// importSG opens Singapore (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H,
// decision 6), from malaysia-singapore-brunei-latest.osm.pbf (2026-11-01),
// imported after Malaysia and Brunei. Read from the tags of its cut:
//
//   - Provinces: the five level-5 relations of the URA's regions (Central,
//     East, North, North-east and West Region). They carry no ISO code, so
//     they are recognised by lying inside the country relation r536780
//     (Parent); the extract's other 32 level-5 relations (Sarawak's and
//     Sabah's divisions, Seberang Perai, George Town, Indonesian kabupaten)
//     lie outside it. The word is "region". " Region" is not stripped: the
//     rest would be a compass point ("North-east"), not a name; the URA
//     and the English Wikipedia call them "Central Region" and so on, so
//     the slug is central-region-singapore.
//   - Districts: the 55 level-6 relations, the URA's planning areas (Ang Mo
//     Kio, Bukit Merah, Tampines, …); the word is "planning area". Their
//     name is English and Latin, none has name:en; 47 carry name:zh and
//     name:ta, which are not read.
//   - English: name:en, int_name, name; never a name in Han, Tamil, Arabic
//     or Thai script. The local language is English (countries.Country.Lang
//     "en"): the local name is name:en, else name, so it always equals the
//     English name and is stored as NULL; the Chinese, Malay and Tamil names
//     (name:zh, name:ms, name:ta) are left to a later decision.
//   - Places: the default six kinds; Singapore has 410 neighbourhoods, 77
//     suburbs, 41 quarters, 4 villages and one city, all named in Latin.
//
// LandPlots: docs/countries/singapore.md, scanned 2026-11-02 from the
// malaysia-singapore-brunei-latest.osm.pbf downloaded 2026-11-01.
var importSG = Import{
	Extract: "malaysia-singapore-brunei",
	Province: Level{
		AdminLevels: []int{5},
		Parent:      536780, // Singapore
		Count:       [2]int{5, 5},
		Kind:        "region",
	},
	District: Level{
		AdminLevels: []int{6},
		Kind:        "planning area",
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Han", "Tamil", "Arabic", "Thai"},
	Local:        []string{"name:en", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "en",
	SlugSuffix:   true,
	LandPlots:    941,
}
