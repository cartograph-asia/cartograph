package countries

// importMN holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/mongolia.md, scanned 2026-11-02 from the
// mongolia-latest.osm.pbf downloaded 2026-11-01.
var importMN = Import{
	Extract:   "mongolia",
	LandPlots: 2_225_456,
	Hold:      "Mongolia does not open in the alpha (decision 8, docs/COUNTRY_IMPORT_STAGE2_BRIEF.md)",
}
