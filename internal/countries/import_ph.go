package countries

// importPH opens the Philippines (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H,
// decision 2). Read from the tags of philippines-latest.osm.pbf (2026-11-01)
// and its cut:
//
//   - Provinces: the level-4 relations, as Adrian decided: the 82 with ISO
//     3166-2 PH-xx and the Special Geographic Area of Bangsamoro (r21710245,
//     no ISO code; border_type=special_geographic_area, the 63 barangays of
//     Cotabato that voted to join the region in 2019), which comes in through
//     Include, so 83. The extract's six other level-4 relations belong to
//     Vietnam, Taiwan, China, Indonesia and Malaysia (Khánh Hòa, Kaohsiung,
//     Hainan, Gorontalo, North Sulawesi, Sabah); the prefix leaves them out.
//     The 18 level-3 regions (Calabarzon, Central Visayas, …) are whole but
//     not used.
//   - Metro Manila and the independent cities are not held at level 4, so
//     they come in through Include as provinces of their own (decision of
//     2026-11-02, for Adrian to confirm or revert before a production
//     import). OSM maps the National Capital Region at level 3 only (r147488,
//     PH-00; its 16 cities and Pateros at level 6, of which Pasig and
//     Mandaluyong do not close in the extract), and the other 17 highly
//     urbanised cities and two independent component cities at level 6 only
//     (border_type=highly_urbanized_city, independent_component_city): no
//     level-4 relation contains them, they are holes in their provinces
//     (Cebu City in Cebu, Davao City in Davao del Sur, Baguio in Benguet,
//     …). Measured on the cut with the importer's land rule, 12,588 land
//     plots, 3.4 % of the country, lie there: Davao City 2,805, Puerto
//     Princesa 2,692, Zamboanga City 1,832, Iligan 867, Butuan 790, General
//     Santos 634, Cagayan de Oro 509, Cebu City 367, Bacolod 201, Olongapo
//     141, Tacloban 134, Lucena 111, Naga 103, Cotabato City 103, Iloilo
//     City 89, Angeles 87, Baguio 77, Lapu-Lapu 75, Mandaue 38, Metro Manila
//     933. Without them these cells had no plot number and an upload there
//     was refused. Metro Manila is one relation (r147488, whole; not its
//     cities); the 19 cities are the level-6 relations above, listed by id
//     under cities. Ormoc and Dagupan (independent component cities), the
//     other component cities and the municipalities lie inside their
//     provinces' relations and get plots as part of the province. The
//     province relations have holes where the 19 cities lie (measured: no
//     land plot lies in a city and in its province's polygon), so the
//     combined RegionIndex's order does not decide anything here.
//     Count: 83 + Metro Manila + 19 = 103.
//   - No district level for now (decision 2): the level-6 municipalities and
//     cities cover only 52 % of the land plots, level 5 holds a few
//     congressional districts and Metro Manila's four districts.
//   - Kind word: "province" for the 82 provinces, the whole name of the
//     Special Geographic Area ("special geographic area"), "region" for Metro
//     Manila (its name in OSM is "Metro Manila", the National Capital
//     Region) and "city" for the 19 cities: by affix where the name ends in
//     " City" (Davao City, Cebu City, Iloilo City, Zamboanga City, Cotabato
//     City, Naga City), else by the whole name (Iligan, Baguio, Butuan, …).
//     " City" stays in the name, as it is part of the common name (Davao
//     City, Cebu City; slug davao-city-philippines). No name is stripped:
//     "Mountain Province" (PH-MOU) is a province's name, and no level-4
//     English name carries "City" or "Province of", so the affix "City"
//     catches nothing else.
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
// LandPlots: see docs/countries/philippines.md. The scan's 367,912 (scanned
// 2026-11-02 from the philippines-latest.osm.pbf downloaded 2026-11-01) is the
// 83 provinces alone: its outline is the union of the level-4 relations and
// cannot take Include of other levels. The figure here is the importer's own
// count with the cities, from the dry run of 2026-11-02 (LANDFIGURE).
var importPH = Import{
	Extract: "philippines",
	Province: Level{
		AdminLevels: []int{4},
		ISOPrefix:   "PH",
		Include: []int64{
			21710245, // Special Geographic Area
			147488,   // Metro Manila (National Capital Region, level 3)
			// The highly urbanised and independent component cities that lie
			// outside every level-4 relation (level 6):
			2387482,  // Cagayan de Oro
			3084673,  // Naga City
			3499101,  // Iloilo City
			3617877,  // Zamboanga City
			3818838,  // Iligan
			3821368,  // Cotabato City
			3936841,  // Davao City
			5154028,  // Tacloban
			9386775,  // Angeles
			9481097,  // Puerto Princesa
			11124741, // Lucena
			11349321, // Bacolod
			12163916, // Lapu-Lapu
			12165003, // Mandaue
			12455830, // Cebu City
			13946201, // Baguio
			14128978, // Olongapo
			14141553, // Butuan
			14144757, // General Santos
		},
		Count: [2]int{103, 103},
		Kind:  "province",
		KindByAffix: map[string]string{
			"Special Geographic Area": "special geographic area",
			"Metro Manila":            "region",
			"City":                    "city",
			// The cities whose names do not end in "City", by their whole name.
			"Angeles":         "city",
			"Bacolod":         "city",
			"Baguio":          "city",
			"Butuan":          "city",
			"Cagayan de Oro":  "city",
			"General Santos":  "city",
			"Iligan":          "city",
			"Lapu-Lapu":       "city",
			"Lucena":          "city",
			"Mandaue":         "city",
			"Olongapo":        "city",
			"Puerto Princesa": "city",
			"Tacloban":        "city",
		},
	},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Han"},
	Local:        []string{"name:fil", "name:tl", "name"},
	LocalScripts: []string{"Latin"},
	LocalLang:    "fil",
	SlugSuffix:   true,
	LandPlots:    367_912,
}
