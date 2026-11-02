package countries

// importPH opens the Philippines (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H,
// decision 2). Read from the tags of philippines-latest.osm.pbf (2026-11-01)
// and its cut:
//
//   - Provinces: the level-4 relations, as Adrian decided: the 82 with ISO
//     3166-2 PH-xx and the Special Geographic Area of Bangsamoro (r21710245,
//     no ISO code; border_type=special_geographic_area, the 63 barangays of
//     Cotabato that voted to join the region in 2019), which comes in through
//     Include, so 83–83. The extract's six other level-4 relations belong to
//     Vietnam, Taiwan, China, Indonesia and Malaysia (Khánh Hòa, Kaohsiung,
//     Hainan, Gorontalo, North Sulawesi, Sabah); the prefix leaves them out.
//     The 18 level-3 regions (Calabarzon, Central Visayas, …) are whole but
//     not used.
//   - Metro Manila and the independent cities are not held at level 4. OSM
//     maps the National Capital Region at level 3 only (r147488, PH-00; its
//     16 cities and Pateros at level 6), and the other 17 highly urbanised
//     cities and two independent component cities at level 6 only
//     (border_type=highly_urbanized_city, independent_component_city): no
//     level-4 relation contains them, they are holes in their provinces
//     (Cebu City in Cebu, Davao City in Davao del Sur, Baguio in Benguet,
//     …). Measured on the cut with the importer's land rule: 12,588 land
//     plots, 3.4 % of the country, lie there: Davao City 2,805, Puerto
//     Princesa 2,692, Zamboanga City 1,832, Iligan 867, Butuan 790, General
//     Santos 634, Cagayan de Oro 509, Cebu City 367, Bacolod 201, Olongapo
//     141, Tacloban 134, Lucena 111, Naga 103, Cotabato City 103, Iloilo
//     City 89, Angeles 87, Baguio 77, Lapu-Lapu 75, Mandaue 38, Metro Manila
//     933. These plots get nothing: their cells lie outside every province,
//     so they are not land plots of this import (the scan's 367,912 does not
//     count them either, its outline is the union of the level-4
//     relations), they get no plot number, and an upload there is refused
//     as outside the covered countries. Ormoc and Dagupan (independent
//     component cities) and the component cities lie inside their
//     provinces' relations and get plots. Whether the cities come in as
//     provinces of their own is Adrian's to decide: Include of Metro Manila
//     as one relation (r147488 is whole; Pasig and Mandaluyong do not close
//     in the extract) and of the 19 level-6 cities, word "city" by
//     KindByAffix, 103 provinces. Adding them later only appends plots and
//     numbers.
//   - No district level for now (decision 2): the level-6 municipalities and
//     cities cover only 52 % of the land plots, level 5 holds a few
//     congressional districts and Metro Manila's four districts.
//   - Kind word: "province" for all but the Special Geographic Area, whose
//     whole name is its word ("special geographic area"). No name is
//     stripped: "Mountain Province" (PH-MOU) is the province's name, and no
//     English name at level 4 carries "City" or "Province of".
//   - English: name:en (79 of 83), else int_name, else name; all 83 names
//     are Latin. Not taken as English: a name in Han script, which the
//     extract holds for features in the Spratlys (南沙区, 渚碧岛) that are
//     mapped by China; such a place has no English name unless name:en.
//   - Local: the Filipino name, from name:fil (one province: South Cotabato
//     "Timog Kutabato"), else name:tl (78 provinces; OSM's Filipino names
//     are tagged as Tagalog, the language Filipino is standardised from),
//     else a Latin name. 10 of the 83 differ from the English one (Kabite,
//     Bulakan, Hilagang Samar, Lalawigang Bulubundukin, …); the others are
//     equal and stored as NULL. Places: 28 nodes whose name:en differs carry
//     the Filipino name in name (Banaba Kanluran → Banaba West), 20 a
//     name:tl (Maynila, Lungsod Quezon); all others NULL. LocalLang is
//     "fil", the country's language.
//   - Places: the default six kinds. place=town and city are the municipal
//     and city centres (admin_level=6 on 1,491 and 149 nodes), village,
//     quarter and suburb the barangays (admin_level=10 on 21,811, 5,742 and
//     513), neighbourhood mostly puroks, sitios and subdivisions; the 11,584
//     hamlets (sitios) stay out as in Thailand.
//
// LandPlots: docs/countries/philippines.md, scanned 2026-11-02 from the
// philippines-latest.osm.pbf downloaded 2026-11-01.
var importPH = Import{
	Extract: "philippines",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "PH",
		Include:     []int64{21710245}, // Special Geographic Area
		Count:       [2]int{83, 83},
		Kind:        "province",
		KindByAffix: map[string]string{"Special Geographic Area": "special geographic area"},
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Han"},
	Local:        []string{"name:fil", "name:tl", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "fil",
	SlugSuffix:   true,
	LandPlots:    367_912,
}
