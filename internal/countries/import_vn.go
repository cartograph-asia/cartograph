package countries

// importVN holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/vietnam.md, scanned 2026-11-02 from the
// vietnam-latest.osm.pbf downloaded 2026-11-01.
var importVN = Import{
	Extract:   "vietnam",
	LandPlots: 390_593,
	Hold:      holdNotOpened,
}
