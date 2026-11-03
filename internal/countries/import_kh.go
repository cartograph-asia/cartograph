package countries

// importKH holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/cambodia.md, scanned 2026-11-02 from the
// cambodia-latest.osm.pbf downloaded 2026-11-01.
var importKH = Import{
	Extract:   "cambodia",
	LandPlots: 213_169,
	Hold:      holdNotOpened,
}
