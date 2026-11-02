package countries

// importTL holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/timor-leste.md, scanned 2026-11-02 from the
// east-timor-latest.osm.pbf downloaded 2026-11-01.
var importTL = Import{
	Extract:   "east-timor",
	LandPlots: 19_546,
	Hold:      holdNotOpened,
}
