package countries

// importSG holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/singapore.md, scanned 2026-11-02 from the
// malaysia-singapore-brunei-latest.osm.pbf downloaded 2026-11-01.
var importSG = Import{
	Extract:   "malaysia-singapore-brunei",
	LandPlots: 941,
	Hold:      holdNotOpened,
}
