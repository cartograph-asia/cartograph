package countries

// importTW opens Taiwan (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H, ninth in
// the order; decision 4). Read from the tags of taiwan-latest.osm.pbf
// (2026-11-01) and its cut:
//
//   - Provinces: the 22 level-4 relations with ISO 3166-2 TW-xx, the six
//     special municipalities, the three provincial cities and the thirteen
//     counties, every one with name:en and a Han name:zh equal to name. The
//     extract's 23rd level-4 relation is Fujian (CN-FJ), which the prefix
//     leaves out. The local name ends in 縣 (county, 13) or 市 (city, 9),
//     which is the kind word; the English names carry it unevenly: eleven
//     counties end in " County", Kinmen and Penghu do not, one city is
//     "Taoyuan City", the eight others are bare ("Keelung", "Tainan").
//     " County", " City", 縣 and 市 are stripped, so all 22 read alike;
//     Chiayi and Hsinchu are both a county and a city, collide after
//     stripping and keep their words by decision 7 ("Chiayi County" 嘉義縣,
//     chiayi-county-taiwan; "Chiayi" 嘉義市, chiayi-taiwan).
//   - Districts: levels 7 and 8 together (decision 4): the 170 districts
//     (區) of the six special municipalities and the three cities at level 7,
//     the 199 townships (鄉), urban townships (鎮) and county-administered
//     cities (市) of the counties at level 8, 369 in all with Kaohsiung whole.
//     The 7,776 villages (里, 村) at level 9 are not districts. The local
//     suffix is the kind word: 區 district, 鄉 township, 鎮 town, 市 city.
//     It is stripped from both names, as the English words " District"
//     (162 of 170 at level 7), " Township" (81 of 199), " City" (13) and
//     " Town" (4) are; 99 level-8 English names carry no word ("Jiuru",
//     "Hengchun"), and stripping makes the rest agree with them. The local
//     suffixes go 鎮 and 鄉 before 市 and 區, each once, so 前鎮區 (Qianzhen),
//     平鎮區 and 左鎮區 lose their 區 only and 新市區 (Xinshi) keeps its 市.
//     The two level-8 relations of Xiamen's subdistricts in the extract lie
//     outside every province and are dropped; 大坵嶼 (an islet of Beigan,
//     level 8) has no English name and is dropped.
//   - Names: English from name:en, int_name, then a name in Latin letters;
//     never a name holding Han, kana or Bopomofo (villages of Fujian in the
//     extract, and Taiwanese names written only in Chinese). Not from
//     name:zh-Latn-pinyin: every relation has name:en, and of the place nodes
//     of the six kinds it would add two, both in tone-marked or run-together
//     pinyin ("Chēchéng", "citongxiang") unlike every other English name.
//     Local from name:zh-Hant, then name:zh, then a Han name: name holds
//     the indigenous name too for some places ("霧社 Alang Paran",
//     "蘭嶼鄉 Ponso no Tao") where name:zh is the Chinese one alone; the
//     three agree otherwise. LocalLang zh-Hant, as countries.Country.
//   - Places: the default six kinds. Hamlets (14,034 nodes, 12,544 with an
//     English name) are Taiwan's most frequent place kind and stay out
//     until Adrian decides (places_kind_check is unchanged).
//   - Neighbours: Kaohsiung (r2127079) takes w284788564, the territorial
//     sea around Pratas (Dongsha), from China's cut and w654482084, the
//     restricted waters around Taiping Island (Itu Aba), from the
//     Philippines' cut (docs/countries/incomplete-relations.md); with either
//     cut alone Kaohsiung does not assemble, so both are needed.
//
// LandPlots: docs/countries/taiwan.md, scanned 2026-11-03 with the china
// and philippines cuts from the taiwan-latest.osm.pbf downloaded 2026-11-01.
var importTW = Import{
	Extract: "taiwan",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "TW",
		Count:       [2]int{22, 22},
		Kind:        "county",
		KindByAffix: map[string]string{"縣": "county", "市": "city"},
		Strip: Affixes{
			EnSuffix:    []string{" County", " City"},
			LocalSuffix: []string{"縣", "市"},
		},
	},
	District: Level{
		AdminLevels: []int{7, 8},
		Kind:        "district",
		KindByAffix: map[string]string{"區": "district", "鄉": "township", "鎮": "town", "市": "city"},
		Strip: Affixes{
			EnSuffix:    []string{" District", " Township", " City", " Town"},
			LocalSuffix: []string{"鎮", "鄉", "市", "區"}, // 區 last: 前鎮區 loses its 區 only
		},
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Han", "Hiragana", "Katakana", "Bopomofo"},
	Local:        []string{"name:zh-Hant", "name:zh", "name"},
	LocalScripts: []string{"Han"},
	LocalLang:    "zh-Hant",
	SlugSuffix:   true,
	Neighbours:   []string{"china", "philippines"},
	LandPlots:    55_564,
}
