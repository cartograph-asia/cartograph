package countries

import "strings"

// Import is how one country is read from OSM (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md,
// stage 2): which relations are its provinces and districts, which tags give
// its names, what is stripped from them. It is plain data; the rules that use
// it live in internal/geo/osmfile and internal/importer. Each country has its
// own file, import_<code>.go, so a country's PR edits only that file.
type Import struct {
	Extract      string // Geofabrik name: map/data/<Extract>-latest.osm.pbf ("malaysia-singapore-brunei", "east-timor")
	Province     Level
	District     Level    // zero value: no district level (Brunei, Philippines)
	English      []string // tags tried in order for name_en
	EnglishNot   []string // Unicode scripts a name_en candidate must not contain ({"Thai"})
	Local        []string // tags tried in order for name_local; plain "name" counts only if written in LocalScripts
	LocalScripts []string // {"Thai"}, {"Latin"}, {"Han", "Hiragana", "Katakana"}, …
	LocalLang    string   // BCP 47 for lang=""; empty: countries.Country.Lang
	SlugSuffix   bool     // province slug + "-" + country slug; false only for th
	PlaceKinds   []string // place=* values read; empty: today's six (Taiwan adds "hamlet")
	LandPlots    int      // the scan's figure, with report and extract date in a comment
	Partial      []int64  // province relations that may be built from their closed rings only (WP F)
	Hold         string   // non-empty: importers refuse with this reason
}

// Level is the rule for one administrative level (provinces or districts).
type Level struct {
	AdminLevels []int             // {4}; Taiwan districts {7, 8}
	ISOPrefix   string            // required ISO3166-2 prefix ("TH" matches "TH-20"); "" = none required
	Parent      int64             // without ISO: relation whose geometry must hold the centre (Singapore r536780)
	Include     []int64           // relations without the ISO code that count anyway (PH Special Geographic Area)
	Count       [2]int            // min, max relations; a dry-run check for provinces, reported for districts
	Kind        string            // display word: "province", "prefecture", "state", "region", "district", "commune"
	KindByAffix map[string]string // word by affix of the English or local name, matched before stripping ("市": "city", "縣": "county", "State": "state")
	Strip       Affixes
}

// Affixes are the words stripped from names. Every entry is tried once, in
// order, suffixes first, then prefixes, trimming spaces between (as
// DistrictName does today).
type Affixes struct{ EnPrefix, EnSuffix, LocalPrefix, LocalSuffix []string }

// Imports holds the settings of every country in All, by code.
var Imports = map[string]Import{
	"bn": importBN,
	"kh": importKH,
	"cn": importCN,
	"id": importID,
	"jp": importJP,
	"la": importLA,
	"my": importMY,
	"mn": importMN,
	"mm": importMM,
	"kp": importKP,
	"ph": importPH,
	"sg": importSG,
	"kr": importKR,
	"tw": importTW,
	"th": importTH,
	"tl": importTL,
	"vn": importVN,
}

// ImportOf returns the import settings of the country with the given ISO
// code, in any case.
func ImportOf(code string) (Import, bool) {
	i, ok := Imports[strings.ToLower(code)]
	return i, ok
}

// holdNotOpened is the Hold of a country whose settings are not written yet:
// its own PR (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H) adds the rules and
// removes the hold.
const holdNotOpened = "not opened yet"
