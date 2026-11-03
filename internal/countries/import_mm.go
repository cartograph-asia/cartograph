package countries

// importMM holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/myanmar.md, scanned 2026-11-02 from the
// myanmar-latest.osm.pbf downloaded 2026-11-01.
var importMM = Import{
	Extract:   "myanmar",
	LandPlots: 804_966,
	Hold:      holdNotOpened,
}
