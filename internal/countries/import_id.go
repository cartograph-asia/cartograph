package countries

// importID holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/indonesia.md, scanned 2026-11-02 from the
// indonesia-latest.osm.pbf downloaded 2026-11-01.
var importID = Import{
	Extract:   "indonesia",
	LandPlots: 2_316_624,
	Hold:      holdNotOpened,
}
