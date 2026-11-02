package countries

// importKR holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/south-korea.md, scanned 2026-11-02 from the
// south-korea-latest.osm.pbf downloaded 2026-11-01.
var importKR = Import{
	Extract:   "south-korea",
	LandPlots: 191_458,
	Hold:      holdNotOpened,
}
