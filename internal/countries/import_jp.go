package countries

// importJP opens Japan (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H, tenth in
// the order). Read from the tags of japan-latest.osm.pbf (2026-11-01) and its
// cut:
//
//   - Provinces: the 47 prefectures, level-4 relations with ISO 3166-2 JP-xx.
//     The extract also holds Busan, South Gyeongsang (KR-xx) and Sakhalin
//     (RU-SAK), which the ISO prefix leaves out. All 47 assemble from the own
//     cut, so no neighbours are read.
//   - Prefecture names. English (name:en) is "Aichi Prefecture" for 46 of
//     them, Hokkaido, Kyoto and Osaka included ("Hokkaido Prefecture",
//     "Osaka Prefecture"), and plain "Tokyo"; " Prefecture" is stripped, so
//     the slugs are aichi-japan, hokkaido-japan, tokyo-japan. The local name
//     is the official one with its designation, 都 (Tokyo), 道 (Hokkaido),
//     府 (Kyoto, Osaka) or 県 (the other 43), and keeps it: 北海道 without 道
//     is no name ("北海"), and Japanese writes a prefecture with its
//     designation (東京都, 京都府, 愛知県). The kind word is "prefecture" for
//     all 47, as the Japanese government and OSM's English names call them,
//     with one exception: 都 makes Tokyo a "metropolis" (Tokyo Metropolis,
//     the word Tokyo's own English uses). Hokkaido stays a prefecture rather
//     than a literal "circuit", and Kyoto and Osaka rather than "urban
//     prefecture": no English name in OSM uses those words.
//   - Local names come from name before name:ja, since name carries the
//     official form where they differ: Tokyo's name is 東京都 but its name:ja
//     東京, Okinawa City's name 沖縄市 but its name:ja 沖縄 (the prefecture is
//     沖縄県); among the place nodes, name:ja sometimes holds only the block
//     ("三丁目" for 桜堤3). name counts only when written in Han or kana;
//     Hiroshima has no name:ja at all.
//   - Districts: the municipalities at level 7 (1,742 relations in the
//     extract): 792 cities (市), 743 towns (町), 183 villages (村) and
//     Tokyo's 23 special wards (区), which are municipalities in their own
//     right, and 所属未定地 (below). The importer keeps 1,740: Sasebo
//     (r4010194) is dropped as "centre outside the provinces", since its
//     centroid lies outside its own shape (in the sea between its islands)
//     and the first vertex taken instead lies outside Nagasaki's simplified
//     outline, 736 land plots without a district (docs/decisions.md,
//     2026-11-03, Japan opened). The 20 designated
//     cities (政令指定都市: Sapporo, Yokohama, Osaka, …) are level-7
//     relations as a whole; their 172 wards (行政区) are level 8 and stay out:
//     a ward lies inside its city, so taking both would overlap, and a plot
//     named "…, Yokohama" says more than "…, Naka". Ward names repeat from
//     city to city (Minami Ward 12 times, Nishi 11, Kita 9, Naka 4), so
//     wards would collide within one prefecture too (Osaka Prefecture: the
//     Kita Wards of Osaka and Sakai). The municipalities hold 0 to 3,377
//     land plots, median 190 (the scan), so none is too large to name
//     plots by.
//   - District names. English names (name:en) carry no designation ("Furano",
//     "Shinjuku", "Kushiro"); one is "Ojiya City", whose " City" is
//     stripped. The local name keeps its 市, 町, 村 or 区, which is how
//     Japanese writes a municipality, and the only thing that tells apart
//     the twelve pairs whose English names are equal within one prefecture
//     (Kushiro: 釧路市 and 釧路町, Fuchu: 府中市 and 府中町, Toshima: Tokyo's
//     豊島区 and 利島村); stripped, the special wards would become single
//     characters (北, 港). Those pairs keep their equal English names and
//     get their OSM id in the slug (decision 7: there is no English affix
//     to restore), hokkaido-japan-kushiro-4069477 and -4071274.
//   - Kind words from the local suffix: 市 city, 町 town, 村 village,
//     区 special ward. KindWord matches an affix at either end of a name and
//     tries one-character affixes in byte order (区, 市, 村, 町), so four
//     towns that start with 市 or 村 would be cities or villages; they are
//     listed whole, which as longer affixes are tried first. 村山市
//     (Murayama), 村上市 (Murakami) and 町田市 (Machida) come out right as
//     they are, since 市 comes before 村 and 町.
//   - 所属未定地 (r2679919, "Shozokumiteichi", "land of undetermined
//     affiliation", in Chiba) is no municipality: about 3 km² of tidal flat
//     in Tokyo Bay off Ichikawa and Funabashi (Sanbanze), whose municipal
//     boundary is not settled, without a land plot. It is excluded, so the
//     dry run counts it instead of a district named "Shozokumiteichi".
//   - Names: English from name:en, int_name, name:ja-Latn, name:ja_rm, name,
//     never a name in Han, kana, Hangul or Cyrillic (the extract holds
//     Korean and Russian places); 27 district names carry macrons
//     (Kyōtango, Higashiōsaka, Bunkyō), which Slug folds (kyotango). No name
//     has an apostrophe.
//   - Places: the default six kinds. Japanese OSM puts the municipality seats
//     in city, town and village, the oaza and machi in quarter and the
//     chōme (numbered blocks) in neighbourhood; the 5,883 hamlets (1,009
//     with an English name) stay out as in Thailand.
//
// LandPlots: docs/countries/japan.md, scanned 2026-11-03 from the
// japan-latest.osm.pbf downloaded 2026-11-01 (unchanged from the scan of
// 2026-11-02).
var importJP = Import{
	Extract: "japan",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "JP",
		Count:       [2]int{47, 47},
		Kind:        "prefecture",
		KindByAffix: map[string]string{"都": "metropolis"},
		Strip:       Affixes{EnSuffix: []string{" Prefecture"}},
	},
	District: Level{
		AdminLevels: []int{7},
		Exclude:     []int64{2679919}, // 所属未定地, the Sanbanze tidal flat, no municipality
		Kind:        "municipality",
		KindByAffix: map[string]string{
			"市": "city", "町": "town", "村": "village", "区": "special ward",
			// towns that start with 市 or 村: matched whole, before the one-character affixes
			"市川町": "town", "市貝町": "town", "市川三郷町": "town", "村田町": "town",
		},
		Strip: Affixes{EnSuffix: []string{" City"}},
	},
	English:      []string{"name:en", "int_name", "name:ja-Latn", "name:ja_rm", "name"},
	EnglishNot:   []string{"Han", "Hiragana", "Katakana", "Hangul", "Cyrillic"},
	Local:        []string{"name", "name:ja"},
	LocalScripts: []string{"Han", "Hiragana", "Katakana"},
	LocalLang:    "ja",
	SlugSuffix:   true,
	LandPlots:    572_035,
}
