package countries

// importJP holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/japan.md, scanned 2026-11-02 from the
// japan-latest.osm.pbf downloaded 2026-11-01.
var importJP = Import{
	Extract:   "japan",
	LandPlots: 572_035,
	Hold:      holdNotOpened,
}
