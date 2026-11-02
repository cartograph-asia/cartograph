package countries

// importCN holds only what the scan found; the rules for provinces,
// districts and names come with the country's own PR.
//
// LandPlots: docs/countries/china.md, scanned 2026-11-02 from the
// china-latest.osm.pbf downloaded 2026-11-01.
//
// The figure is too low: the extract does not hold China's outline whole
// (Xizang, Fujian and Hainan do not assemble), so the China package
// replaces it with a corrected scan.
var importCN = Import{
	Extract:   "china",
	LandPlots: 11_976_594,
	Hold:      "China does not open in the alpha (decision 8, docs/COUNTRY_IMPORT_STAGE2_BRIEF.md)",
}
