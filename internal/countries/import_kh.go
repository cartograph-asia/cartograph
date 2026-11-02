package countries

// importKH opens Cambodia, the second country after Thailand and Laos
// (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H). Read from the tags of
// cambodia-latest.osm.pbf (2026-11-01) and its cut:
//
//   - Provinces: the 25 level-4 relations with ISO 3166-2 KH-xx, every one
//     with name:en and a Khmer name. 24 are khaet (province) and carry ខេត្ត
//     before the Khmer name, which goes; Phnom Penh (KH-12, the capital
//     municipality) is called រាជធានីភ្នំពេញ, whose រាជធានី (reach theani,
//     "capital") goes too, and is a capital by that word. "Kampong" (Kampong
//     Cham, Kampong Speu, …) is part of the name, not an affix. One English
//     name carries OSM's transliteration of khaet: "Khaet Preah Sihanouk"
//     (KH-18), which becomes "Preah Sihanouk" (preah-sihanouk-cambodia).
//     The name:en of the others has no affix at all.
//   - Districts: the level-6 relations whose centre lies in a province, 200
//     (none has an ISO code, none a commune or sangkat; 102 more of the
//     extract's level-6 relations lie outside Cambodia, in the neighbours).
//     Their English names carry no affix, except " District" on three and
//     "Khan " on the 14 khan of Phnom Penh ("Khan Daun Penh"), whose Khmer
//     names start with ខណ្ឌ (13 of 14); both go. The Khmer prefixes ស្រុក (srok,
//     district) and ក្រុង (krong, municipality) go where a name has them (4
//     and 1).
//     The districts leave about 7 % of the land without a district: 15,432 of
//     the 213,166 plots, in Koh Kong (6,515 of its 12,726), Kampong Thom
//     (4,110), Pursat (2,710), Kampong Chhnang (1,716) and Battambang (350),
//     where no level-6 relation of the extract covers the forested hills. A
//     plot there is in its province and has no district: it is named after
//     the nearest place and its province, or after the province and number
//     alone when no place is near, and the district part is left out of the
//     name (views.PlotRef.Name).
//   - English: name:en, then int_name, name:km-Latn and name. Not taken as
//     English: names in Khmer, or in the scripts of the neighbours whose
//     villages the extract holds beyond the border (Thai, Lao, Han,
//     Myanmar); such a place has no English name and is skipped.
//   - Local: name:km, else a Khmer name.
//   - Places: the default six kinds; 564 hamlets stay out as they do in
//     Thailand and Laos.
//
// LandPlots: docs/countries/cambodia.md, scanned 2026-11-02 from the
// cambodia-latest.osm.pbf downloaded 2026-11-01.
var importKH = Import{
	Extract: "cambodia",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "KH",
		Count:       [2]int{25, 25},
		Kind:        "province",
		KindByAffix: map[string]string{"រាជធានី": "capital"},
		Strip: Affixes{
			EnPrefix:    []string{"Khaet "},
			LocalPrefix: []string{"ខេត្ត", "រាជធានី"},
		},
	},
	District: Level{
		AdminLevels: []int{6},
		Kind:        "district",
		Strip: Affixes{
			EnSuffix:    []string{" District"},
			EnPrefix:    []string{"Khan "},
			LocalPrefix: []string{"ខណ្ឌ", "ស្រុក", "ក្រុង"},
		},
	},
	English:      []string{"name:en", "int_name", "name:km-Latn", "name"},
	EnglishNot:   []string{"Khmer", "Thai", "Lao", "Han", "Myanmar"},
	Local:        []string{"name:km", "name"},
	LocalScripts: []string{"Khmer"},
	LocalLang:    "km",
	SlugSuffix:   true,
	LandPlots:    213_169,
}
