package countries

// importKR opens South Korea (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H,
// seventh in the order). Read from the tags of south-korea-latest.osm.pbf
// (2026-11-01) and its cut:
//
//   - Provinces: 16 level-4 relations. Fifteen carry ISO 3166-2 KR-xx (Seoul,
//     Busan, Daegu, Incheon, Daejeon, Ulsan, Sejong and the provinces); the
//     16th, Jeonnam-Gwangju Special Metropolitan City (전남광주통합특별시,
//     r21039457, Wikidata Q138870299), has no ISO code and comes in through
//     Include. It is the merger of the old Gwangju (KR-29) and South Jeolla
//     (KR-46) in force since 2026: the extract holds neither of them any more,
//     so without it the south-west of the country (Gwangju and all of Jeonnam)
//     would be no province at all. The extract also holds North Korea's
//     level-4 relations (KP-xx), which the ISO prefix leaves out. All 16 have
//     name:en and name:ko.
//   - Province names. Local names carry the designation: 특별시 (special
//     city: Seoul), 광역시 (metropolitan city: Busan, Daegu, Incheon, Daejeon,
//     Ulsan), 특별자치시 (special self-governing city: Sejong), 특별자치도
//     (special self-governing province: Gangwon, Jeonbuk, Jeju), 통합특별시
//     (Jeonnam-Gwangju) and plain 도 (province: Gyeonggi and the four
//     Chungcheong and Gyeongsang provinces). Those designations are the kind
//     word (KindByAffix, from the local name; the English name carries them
//     only for Jeonnam-Gwangju) and are stripped from the local name as
//     Korean usage drops them (서울, 부산, 강원, 전북, 제주). The plain 도 stays:
//     it is part of the name (경기도, 충청북도; "경상북" is no name). English
//     names are mostly bare already ("Seoul", "North Gyeongsang");
//     " State" (Gangwon State), "-do" (Jeonbuk-do) and " Special Metropolitan
//     City" go, so the slugs are seoul-south-korea, gangwon-south-korea,
//     jeonbuk-south-korea, jeonnam-gwangju-south-korea. The word for
//     Jeonnam-Gwangju is OSM's English one, "special metropolitan city".
//   - Districts: the 229 level-6 relations of the country (no ISO code):
//     73 si (cities: "Andong-si"), 77 gun (counties, "Cheorwon-gun") and 69
//     gu (districts of the metropolitan cities, "Jongno-gu"), the last 10
//     with a bare English name ("Ansan", "Jeju", "Sinan") and one new Incheon
//     district, 검단구, without an English name, which is skipped. The suffixes
//     "-si", "-gun", "-gu" and the Korean 시, 군, 구 are stripped from the
//     names, so the 219 with and the ten without a suffix agree ("Andong",
//     "Gangnam"-style), and the type is the kind word: city, county or
//     district. The five names that are only a direction (Jung, Dong, Seo, Nam,
//     Buk, 23 districts) become one word; the settings keep no "-gu" for them.
//     Kind word: KindWord matches an affix at either end of the name, and
//     구리시 (Guri), 구미시, 군산시, 군포시 and 구례군 start with 구 or 군. So
//     the English suffixes are listed too: they are longer and tried first,
//     and every name that has an English suffix is decided by it; the Korean
//     one decides the ten bare names, none of which starts with 구, 군 or 시.
//   - Names: English from name:en, int_name, name:ko-Latn, name (never a name
//     in Hangul, Han, kana or Cyrillic: villages beyond the border in North
//     Korea, China, Japan and Russia); local from name:ko, else a Hangul name.
//     name:ko-Hani is Hanja, not the local name. Accents do not occur.
//   - Places: the default six kinds; 719 hamlets stay out as in Thailand.
//
// LandPlots: docs/countries/south-korea.md, scanned 2026-11-02 from the
// south-korea-latest.osm.pbf downloaded 2026-11-01.
var importKR = Import{
	Extract: "south-korea",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "KR",
		Include:     []int64{21039457}, // Jeonnam-Gwangju, no ISO code since the 2026 merger
		Count:       [2]int{16, 16},
		Kind:        "province",
		KindByAffix: map[string]string{
			"특별자치도":                     "special self-governing province",
			"특별자치시":                     "special self-governing city",
			"통합특별시":                     "special metropolitan city",
			"특별시":                       "special city",
			"광역시":                       "metropolitan city",
			"Special Metropolitan City": "special metropolitan city",
			"Metropolitan City":         "metropolitan city",
			"Special City":              "special city",
		},
		Strip: Affixes{
			EnSuffix:    []string{" Special Metropolitan City", " Metropolitan City", " Special City", " Province", " State", "-do"},
			LocalSuffix: []string{"통합특별시", "특별자치도", "특별자치시", "특별시", "광역시"},
		},
	},
	District: Level{
		AdminLevels: []int{6},
		Kind:        "district",
		KindByAffix: map[string]string{
			"-si": "city", "-gun": "county", "-gu": "district",
			"시": "city", "군": "county", "구": "district",
		},
		Strip: Affixes{
			EnSuffix:    []string{"-si", "-gun", "-gu"},
			LocalSuffix: []string{"시", "군", "구"},
		},
	},
	English:      []string{"name:en", "int_name", "name:ko-Latn", "name"},
	EnglishNot:   []string{"Hangul", "Han", "Hiragana", "Katakana", "Cyrillic"},
	Local:        []string{"name:ko", "name"},
	LocalScripts: []string{"Hangul"},
	LocalLang:    "ko",
	SlugSuffix:   true,
	LandPlots:    191_458,
}
