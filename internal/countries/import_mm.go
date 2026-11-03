package countries

// importMM opens Myanmar (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H, eleventh
// in the order; decision 5). Read from the tags of myanmar-latest.osm.pbf
// (2026-11-01) and its cut:
//
//   - Provinces: the 15 level-4 relations with ISO 3166-2 MM-xx, the seven
//     states (Chin, Kachin, Kayah, Kayin, Mon, Rakhine, Shan), the seven
//     regions (Ayeyarwady, Bago, Magway, Mandalay, Sagaing, Tanintharyi,
//     Yangon) and the Naypyitaw Union Territory. The extract holds two more
//     level-4 relations of the country without an ISO code, Northern Wa
//     State (r9399861) and Wa State Southern Region (r9399860): the areas the
//     United Wa State Army holds, which lie inside Shan State and made the
//     scan's 2.88 % overlap. The ISO prefix leaves them out (decision 5), so
//     their land is Shan State's. The other 19 level-4 relations of the
//     extract are the neighbours' (CN, TH, IN, BD, LA), cut by the extract,
//     and fall out by the prefix too.
//   - Province names. The English names carry the word or not, as OSM has
//     them: "Kachin State", "Bago Region", "Naypyitaw Union Territory", but
//     "Chin", "Rakhine", "Yangon", "Mandalay", "Magway", "Ayeyarwady" (Kayin
//     has no name:en; its int_name is "Kayin State"). The Burmese names all
//     carry it: ပြည်နယ် (state), တိုင်းဒေသကြီး (region, since 2008), the older
//     တိုင်း (division, now region: Mandalay, Tanintharyi and Ayeyarwady's
//     name:my) and ပြည်ထောင်စုနယ်မြေ (union territory). So the kind word
//     comes from the Burmese name where the English one lacks it, and every
//     word is stripped from both names: kachin-myanmar ("ကချင်"),
//     yangon-myanmar ("ရန်ကုန်"), naypyitaw-myanmar ("နေပြည်တော်").
//   - Districts: the townships at level 7 (358 relations of the country;
//     decision 5), "Falam Township" (ဖလမ်းမြို့နယ်); Yangon's and a few
//     other townships are named without the word ("Ahlone", "Mawlamyine",
//     "North Dagon"). " Township" and မြို့နယ် are stripped, so all agree and
//     the kind word is "township". Three slips in the Burmese names are
//     handled by the suffixes: two lack the final asat (လက်ပတန်းမြို့နယ,
//     ညောင်လေးပင်မြို့နယ), one has a zero-width non-joiner before the word
//     (ရှီးရှမ်း‌မြို့နယ်). Mong Yawng is the one township OSM names "Mong Yawng
//     Township (de facto)" (မိုင်းယောင်းမြို့နယ် (အမှန်တကယ်)); the relation is
//     the only Mong Yawng, so the qualifier goes with the word.
//   - Names: English from name:en, int_name, name:my-Latn, name (a Latin
//     name only: 342 places, many from an import of 2008); never a name in
//     Myanmar script (one int_name is) or in the scripts of the neighbours
//     whose villages the extract holds beyond the border (Han, Thai, Lao,
//     Bengali, one Cyrillic; Devanagari and Tibetan as a precaution). Local
//     from name:my, then a name in Myanmar script: name:my first, because
//     the Wa and Kokang townships' name joins Wa (Latin) or Burmese with
//     Chinese ("Veng' Na' Vi' 纳威区", "လောက်ကိုင်မြို့နယ် 老街区") while their
//     name:my is Burmese alone. The names are Unicode, not Zawgyi: of about
//     59,000 Burmese names of relations and place nodes, 7 place names are
//     Zawgyi-encoded and none of a province or township. Shan names in
//     Myanmar script (ဝၢၼ်ႈ…) are local names too, marked "my".
//   - Places: the default six kinds; 1,234 hamlets stay out as in Thailand.
//
// LandPlots: docs/countries/myanmar.md, scanned 2026-11-02 and again
// 2026-11-03 (the same report) from the myanmar-latest.osm.pbf downloaded
// 2026-11-01.
var importMM = Import{
	Extract: "myanmar",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "MM",
		Count:       [2]int{15, 15},
		Kind:        "region",
		KindByAffix: map[string]string{
			"State":           "state",
			"Region":          "region",
			"Union Territory": "union territory",
			"ပြည်နယ်":         "state",
			"တိုင်းဒေသကြီး":     "region",
			"တိုင်း":            "region",
			"ပြည်ထောင်စုနယ်မြေ": "union territory",
		},
		Strip: Affixes{
			EnSuffix:    []string{" State", " Region", " Union Territory"},
			LocalSuffix: []string{"တိုင်းဒေသကြီး", "တိုင်း", "ပြည်နယ်", "ပြည်ထောင်စုနယ်မြေ"},
		},
	},
	District: Level{
		AdminLevels: []int{7},
		Kind:        "township",
		Strip: Affixes{
			EnSuffix:    []string{" Township (de facto)", " Township"},
			LocalSuffix: []string{"မြို့နယ် (အမှန်တကယ်)", "‌မြို့နယ်", "မြို့နယ်", "မြို့နယ"},
		},
	},
	English:      []string{"name:en", "int_name", "name:my-Latn", "name"},
	EnglishNot:   []string{"Myanmar", "Han", "Thai", "Lao", "Bengali", "Devanagari", "Tibetan", "Cyrillic"},
	Local:        []string{"name:my", "name"},
	LocalScripts: []string{"Myanmar"},
	LocalLang:    "my",
	SlugSuffix:   true,
	LandPlots:    804_966,
}
