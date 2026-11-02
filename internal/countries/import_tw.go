package countries

// importTW holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/taiwan.md, scanned 2026-11-02 from the
// taiwan-latest.osm.pbf downloaded 2026-11-01.
var importTW = Import{
	Extract:   "taiwan",
	LandPlots: 51_028,
	Hold:      holdNotOpened,
}
