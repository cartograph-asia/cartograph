package countries

// importVN opens Vietnam (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H, third
// in the order). Read from the tags of vietnam-latest.osm.pbf (2026-11-01)
// and its cut, after the 2025 reform (provinces merged to 34, districts
// abolished, communes and wards directly under the province):
//
//   - Provinces: the 34 level-4 relations with ISO 3166-2 VN-xx, every one
//     with name:en and a Vietnamese name ("Tỉnh Cao Bằng", "Thành phố Hà
//     Nội"). name:en ends in " Province" (24) or " City" (2: Bắc Ninh,
//     Quang Ninh; "Ho Chi Minh City" is the third). " Province" goes, " City"
//     stays: it is part of the common name (Ho Chi Minh City, slug
//     ho-chi-minh-city-vietnam), and it stays in every province that
//     carries it. The local prefixes Tỉnh (province) and Thành phố (city) go.
//     The kind word follows the name as OSM has it: "Thành phố" or " City"
//     makes a city (nine: Hà Nội, Hồ Chí Minh, Hải Phòng, Đà Nẵng, Cần Thơ,
//     Huế, and Bắc Ninh, Quảng Ninh, Đồng Nai, which OSM names "Thành phố"
//     too).
//   - Districts: the level-6 relations of the country, communes (Xã),
//     wards (Phường) and special zones (Đặc khu, the island districts), 3,319
//     of them with Khánh Hòa whole. Each has a Vietnamese name with its
//     prefix; about half have name:en ("A Dơi Commune", "Nam Triệu Ward",
//     "Vân Đồn Special Administrative Region", four of them "… Special
//     Administration Region"), the others take the Latin name as English.
//     Prefix and suffix go, so English and local are mostly the same name
//     and the local one is stored as NULL. Cồn Cỏ is the one name without a
//     prefix; its name:en makes it a special zone.
//   - Names: English from name:en, int_name, then name (Vietnamese is
//     written in Latin letters, so name is English-capable); never a name
//     in the scripts of the neighbours whose villages the extract holds
//     beyond the border (Han, Lao, Khmer). Local from name before name:vi:
//     the two are equal for every Vietnamese relation except where name:vi
//     lacks the prefix (Hà Nội, Khánh Hòa, Hà Tĩnh), and the prefix decides
//     the kind word. Slugs fold the accents (Cần Thơ → can-tho-vietnam).
//   - Places: the default six kinds; 3,295 hamlets stay out as they do in
//     Thailand and Laos.
//   - Excluded: r6753263, China's "Nansha District" (level 6, in the
//     extract for its islands), which an earlier run imported as a Khánh Hòa
//     commune. Vietnam's own Trường Sa special zone (r3479413) is not
//     excluded.
//   - Neighbours: Khánh Hòa (r1887959) takes 16 territorial-sea ways
//     around the Spratly features from the Philippines' cut
//     (docs/countries/incomplete-relations.md).
//
// LandPlots: docs/countries/vietnam.md, scanned 2026-11-02 with the
// philippines cut from the vietnam-latest.osm.pbf downloaded 2026-11-01.
var importVN = Import{
	Extract: "vietnam",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "VN",
		Count:       [2]int{34, 34},
		Kind:        "province",
		KindByAffix: map[string]string{"Thành phố": "city", "City": "city"},
		Strip: Affixes{
			EnSuffix:    []string{" Province"},
			LocalPrefix: []string{"Tỉnh ", "Thành phố "},
		},
	},
	District: Level{
		AdminLevels: []int{6},
		Exclude:     []int64{6753263},
		Kind:        "commune",
		KindByAffix: map[string]string{
			"Xã": "commune", "Commune": "commune",
			"Phường": "ward", "Ward": "ward",
			"Đặc khu": "special zone", "Special Administrative Region": "special zone", "Special Administration Region": "special zone",
		},
		Strip: Affixes{
			EnSuffix:    []string{" Commune", " Ward", " Special Administrative Region", " Special Administration Region"},
			EnPrefix:    []string{"Xã ", "Phường ", "Đặc khu "},
			LocalPrefix: []string{"Xã ", "Phường ", "Đặc khu "},
		},
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Han", "Lao", "Khmer"},
	Local:        []string{"name", "name:vi"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "vi",
	SlugSuffix:   true,
	Neighbours:   []string{"philippines"},
	LandPlots:    400_417,
}
