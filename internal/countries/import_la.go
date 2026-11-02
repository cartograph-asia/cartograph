package countries

// importLA holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/laos.md, scanned 2026-11-02 from the
// laos-latest.osm.pbf downloaded 2026-11-01.
var importLA = Import{
	Extract:   "laos",
	LandPlots: 275_740,
	Hold:      holdNotOpened,
}
