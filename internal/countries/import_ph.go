package countries

// importPH holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/philippines.md, scanned 2026-11-02 from the
// philippines-latest.osm.pbf downloaded 2026-11-01.
var importPH = Import{
	Extract:   "philippines",
	LandPlots: 367_912,
	Hold:      holdNotOpened,
}
