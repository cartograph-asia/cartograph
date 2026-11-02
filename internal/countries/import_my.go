package countries

// importMY opens Malaysia (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H), the
// first of the three countries of malaysia-singapore-brunei-latest.osm.pbf
// (2026-11-01); Brunei and Singapore follow from the same extract, in that
// order. Read from the tags of its cut:
//
//   - Provinces: the 16 level-4 relations with ISO 3166-2 MY-xx, the 13
//     states and the 3 federal territories (Kuala Lumpur, Putrajaya,
//     Labuan). The extract's other 16 level-4 relations are Thai,
//     Indonesian, Bruneian and Philippine and fall out by the prefix. No name
//     carries a word to strip: name:en is "Penang", "Malacca", the others
//     equal name and name:ms (Sabah, Selangor, Johor, Negeri Sembilan and
//     Labuan have no name:en; their name is Latin). The word is "state";
//     the three federal territories are told apart only by their
//     official_name ("Wilayah Persekutuan Kuala Lumpur"), which the name
//     rules do not read, so KindByAffix names them whole (an affix that is
//     the whole name matches, osmfile.KindWord).
//   - Sabah (r3879783) assembles only with two territorial-sea ways around
//     Swallow Reef (Layang-Layang) that lie in the Philippines extract
//     (docs/countries/incomplete-relations.md), so Neighbours is
//     "philippines"; nothing of Sabah's land depends on them, but without
//     them the relation does not close and the province is lost.
//   - Districts: the level-6 relations (daerah; jajahan in Kelantan; the
//     districts of Sabah and Sarawak, below Sarawak's level-5 divisions).
//     None has an ISO code; those of the extract that are not Malaysia's
//     (Brunei's mukim, Singapore's planning areas, southern Thailand's
//     amphoe, Riau and West Kalimantan's kecamatan, Palawan's municipalities)
//     drop out because their centre lies in no Malaysian province. English
//     names: name:en on 28 (" District" on six of them, which goes:
//     "Jasin District", "Kemaman District", "North Seberang Perai District"),
//     else name. Local names: name:ms, whose "Daerah " (Langkawi, Sandakan,
//     Mersing, Kota Tinggi, Manjung) and "Jajahan " (Kelantan's six) go,
//     else name; equal to the English name (most) they are NULL. Kelantan's
//     name is Jawi and Latin ("باچوق / Bachok"); its name:en is taken.
//   - English: name:en, int_name, name; never a name in Arabic (Jawi), Han,
//     Tamil or Thai script (the extract holds two Thai-named villages beyond
//     the border). Local: name:ms, else a Latin name; LocalLang "ms".
//   - Places: the default six kinds; 2,853 hamlets stay out as in Thailand.
//
// LandPlots: docs/countries/malaysia.md, scanned 2026-11-02 with the
// philippines cut as neighbour from the malaysia-singapore-brunei-latest.osm.pbf
// downloaded 2026-11-01.
var importMY = Import{
	Extract: "malaysia-singapore-brunei",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "MY",
		Count:       [2]int{16, 16},
		Kind:        "state",
		KindByAffix: map[string]string{
			"Kuala Lumpur": "federal territory",
			"Putrajaya":    "federal territory",
			"Labuan":       "federal territory",
		},
	},
	District: Level{
		AdminLevels: []int{6},
		Kind:        "district",
		Strip:       Affixes{EnSuffix: []string{" District"}, LocalPrefix: []string{"Daerah ", "Jajahan "}},
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Arabic", "Han", "Tamil", "Thai"},
	Local:        []string{"name:ms", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "ms",
	SlugSuffix:   true,
	Neighbours:   []string{"philippines"},
	LandPlots:    419_805,
}
