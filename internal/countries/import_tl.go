package countries

// importTL opens Timor-Leste (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H).
// Read from the tags of east-timor-latest.osm.pbf (2026-11-01) and its cut:
//
//   - Provinces: the 14 level-4 relations, the municipalities (munisípiu; the
//     OSM wikipedia tags name them so). Thirteen carry ISO 3166-2 TL-xx (the
//     twelve municipalities and the special administrative region of
//     Oe-Cusse Ambeno); the 14th, Atauro (r12363798, a municipality since
//     2022), has no ISO code and comes in through Include. The extract also
//     holds two level-4 relations of Indonesia (ID-NT, ID-MA), which the ISO
//     prefix leaves out. All 14 carry name:en and name:tet; no name carries a
//     word to strip ("Municipality of …" does not occur), so Strip is empty.
//     The word is "municipality" for all fourteen. Oe-Cusse Ambeno is a
//     special administrative region (ISO 3166-2 lists it beside the twelve
//     municipalities), but OSM maps it at the same level and the name carries
//     no affix that tells it apart, so it keeps the common word; whether it
//     deserves "special administrative region" is Adrian's call.
//   - Districts: the 70 level-5 relations, the administrative posts (postu
//     administrativu, formerly sub-districts; none has an ISO code). The word
//     is "administrative post", the English name the government uses. Five of
//     the 70 lie in Atauro, which has no posts of its own: Vila Maumeta, its
//     main town, and four sucos (Beloi, Bikeli, Makadade, Makili) that OSM
//     tags at level 5 as well. The four posts of Oe-Cusse Ambeno assemble;
//     the four level-5 relations that drop out as not assembled are West
//     Timor's kabupaten (Kupang, Timor Tengah Utara, Belu, Malaka), cut by
//     the extract and not Timor-Leste's.
//   - English: name:en, then int_name and name, all Latin. 34 of the posts
//     have name:en; the others are written in Latin in name.
//   - Local: name:tet, then a Latin name. Tetum, not Portuguese, is the
//     language of the names: name:tet is on all 14 municipalities and all 70
//     posts, name:pt on 13 of 14 and on none of the posts, and name carries
//     Tetum spellings (Baukau, Bobonaru, Vikeke, Kova Lima) beside a few
//     Portuguese ones (Lautém, Liquiçá). So LocalLang is "tet" (ISO 639-3,
//     Tetum has no two-letter code) and not the Portuguese of
//     countries.Country.Lang; a name equal to the English one (42 of the
//     posts) is stored as NULL.
//   - Places: the default six kinds; 2,624 hamlets stay out as in Thailand.
//
// LandPlots: docs/countries/timor-leste.md, scanned 2026-11-02 from the
// east-timor-latest.osm.pbf downloaded 2026-11-01.
var importTL = Import{
	Extract: "east-timor",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "TL",
		Include:     []int64{12363798}, // Atauro
		Count:       [2]int{14, 14},
		Kind:        "municipality",
	},
	District: Level{
		AdminLevels: []int{5},
		Kind:        "administrative post",
	},
	English:      []string{"name:en", "int_name", "name"},
	Local:        []string{"name:tet", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "tet",
	SlugSuffix:   true,
	LandPlots:    19_546,
}
