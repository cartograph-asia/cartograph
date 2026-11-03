package countries

// importID opens Indonesia (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H, row 12
// of the order of countries). Read from the tags of indonesia-latest.osm.pbf
// (2026-11-01) and its cut:
//
//   - Provinces: the 38 level-4 relations with ISO 3166-2 ID-xx, every one
//     with name:en and name:id. The extract's 22 other level-4 relations are
//     Malaysian, Timorese and Papua New Guinean and fall out by the prefix.
//     No English name carries " Province"; it is stripped anyway, with the
//     local "Provinsi ", should one appear. Two names carry their status:
//     "Special Region of Yogyakarta" (Daerah Istimewa Yogyakarta) and
//     "Special Capital Region of Jakarta" (Daerah Khusus Ibukota Jakarta);
//     the words go, so the slugs are yogyakarta-indonesia and
//     jakarta-indonesia, and KindByAffix keeps them as "special region" and
//     "special capital region". Aceh is a special region too, though its
//     name ("Aceh") says nothing, so it is named whole.
//   - Riau Islands (r3797244) and North Kalimantan (r5449460) assemble only
//     with one way each from the malaysia-singapore-brunei cut (a
//     territorial-sea line east of Singapore, 3.5 km of land border on
//     Borneo; docs/countries/incomplete-relations.md), so Neighbours is
//     "malaysia-singapore-brunei"; without it both provinces are lost.
//   - Districts: the level-5 relations, the kabupaten (regencies) and kota
//     (cities). None but one has an ISO code; those of the extract that are
//     not Indonesia's (Sarawak's divisions, Timor-Leste's municipalities)
//     drop out because their centre lies in no Indonesian province. Most
//     names are bare ("Agam", "Kendari"); OSM writes the word in name for
//     some ("Kabupaten Karo", "Kota Medan", "Kab Bogor", "Nias Regency") and
//     in name:en for others ("City of Medan", "Klungkung Regency", "Dumai
//     City"), and these words go in both languages. The word is the kind:
//     Kota, City → "city", else "regency". The affixes carry their space,
//     so "Lima Puluh Kota" (a regency) and "Kotabaru" are not cities. Most
//     cities have the word only in official_name ("Kota Kendari"), which
//     the name rules do not read, so they are "regency" too (reported in
//     docs/decisions.md; district kind words are not shown on any page yet).
//   - English: name:en, int_name, name:id-Latn, name; Indonesian is Latin,
//     so name is English-capable. Never a name in Arabic (Jawi) or Han
//     script. Local: name:id, else name; equal to the English name (most)
//     it is NULL. LocalLang "id".
//   - Places: the default six kinds; 27,294 hamlets (dusun) stay out as in
//     Thailand.
//
// LandPlots: docs/countries/indonesia.md, scanned 2026-11-03 with the
// malaysia-singapore-brunei cut as neighbour from the indonesia-latest.osm.pbf
// downloaded 2026-11-01.
var importID = Import{
	Extract: "indonesia",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "ID",
		Count:       [2]int{38, 38},
		Kind:        "province",
		KindByAffix: map[string]string{
			"Special Capital Region of": "special capital region",
			"Daerah Khusus Ibukota":     "special capital region",
			"Special Region of":         "special region",
			"Daerah Istimewa":           "special region",
			"Aceh":                      "special region",
		},
		Strip: Affixes{
			EnSuffix:    []string{" Province"},
			EnPrefix:    []string{"Special Capital Region of ", "Special Region of "},
			LocalPrefix: []string{"Provinsi ", "Daerah Khusus Ibukota ", "Daerah Istimewa "},
		},
	},
	District: Level{
		AdminLevels: []int{5},
		Kind:        "regency",
		KindByAffix: map[string]string{
			"Kota ":      "city",
			"City of ":   "city",
			" City":      "city",
			"Kabupaten ": "regency",
			"Kab ":       "regency",
			" Regency":   "regency",
		},
		Strip: Affixes{
			EnSuffix:    []string{" Regency", " City"},
			EnPrefix:    []string{"City of ", "Kabupaten ", "Kota ", "Kab "},
			LocalSuffix: []string{" Regency"},
			LocalPrefix: []string{"Kabupaten ", "Kota ", "Kab "},
		},
	},
	English:      []string{"name:en", "int_name", "name:id-Latn", "name"},
	EnglishNot:   []string{"Arabic", "Han"},
	Local:        []string{"name:id", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "id",
	SlugSuffix:   true,
	Neighbours:   []string{"malaysia-singapore-brunei"},
	LandPlots:    2_423_959,
}
