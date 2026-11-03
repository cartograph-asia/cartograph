package countries

// importBN holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/brunei.md, scanned 2026-11-02 from the
// malaysia-singapore-brunei-latest.osm.pbf downloaded 2026-11-01.
var importBN = Import{
	Extract:   "malaysia-singapore-brunei",
	LandPlots: 7_864,
	Hold:      holdNotOpened,
}
