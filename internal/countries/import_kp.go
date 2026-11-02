package countries

// importKP holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/north-korea.md, scanned 2026-11-02 from the
// north-korea-latest.osm.pbf downloaded 2026-11-01.
var importKP = Import{
	Extract:   "north-korea",
	LandPlots: 214_569,
	Hold:      holdNotOpened,
}
