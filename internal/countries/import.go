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

	// Neighbours are Geofabrik extract names whose cuts
	// (map/data/scan/<name>-latest.osm.pbf.cut.osm.pbf) complete this
	// country's boundaries: a province or district relation of the country's
	// own cut whose member ways lie partly in a neighbour's extract (a border
	// line, a territorial-sea line; docs/countries/incomplete-relations.md)
	// takes those ways and their nodes from the neighbours' cuts. The
	// relations themselves come from the own cut only. The neighbours' cuts
	// are read for boundaries only, so that relations cut by the country's
	// own extract can be assembled; never for places, and they never join
	// the base map.
	Neighbours []string

	// PlaceReachKm is how far (kilometres) outside the run's provinces a
	// place node may lie and still be stored for the country, then without
	// a province (region_id NULL, a "weak" place: a beach village beyond the
	// simplified coastline, a hamlet on a river that is the border with a
	// country not imported yet). A place strictly inside one of the
	// country's provinces is stored with it; one strictly inside another
	// country's stored province, or farther than this from every province
	// of the run, is not stored. 0 means 2 km. A weak place passes to the
	// country into whose province it falls strictly when that country is
	// imported; a weak place of another country is never taken by a weak
	// claim (docs/decisions.md, WP L).
	PlaceReachKm float64

	// LegacyPlaces keeps the place rule Thailand was imported with before
	// WP L: every place node of the extract is stored, with the province
	// containing it or, failing that, the nearest one within
	// geo.NearTolerance (about 1 km), else without a province. Set for
	// Thailand alone, so that its stored places never change; PlaceReachKm
	// is then not read.
	LegacyPlaces bool

	// LegacyDuplicateSlug keeps the duplicate rule Thailand's districts were
	// imported with before the settings existed: of two districts of one run
	// whose slugs collide, the first by OSM id keeps the slug and the next
	// gets "-<osm id>" appended, their names stay stripped and no affix is
	// restored (instead of decision 7 of docs/COUNTRY_IMPORT_STAGE2_BRIEF.md).
	// A stored slug never changes, so this only decides what a fresh import
	// gives; it is set for Thailand alone, so that a fresh import (dev
	// stack, fixture, a restored or rebuilt production) gives the slugs
	// production stores (phra-nakhon-si-ayutthaya-bang-sai and
	// phra-nakhon-si-ayutthaya-bang-sai-11024893). Provinces keep decision 7.
	LegacyDuplicateSlug bool
}

// Level is the rule for one administrative level (provinces or districts).
type Level struct {
	AdminLevels []int   // {4}; Taiwan districts {7, 8}
	ISOPrefix   string  // required ISO3166-2 prefix ("TH" matches "TH-20"); "" = none required
	Parent      int64   // without ISO: relation whose geometry must hold the centre (Singapore r536780)
	Include     []int64 // relations without the ISO code that count anyway (PH Special Geographic Area)
	// Exclude lists relations that match the rule (AdminLevels, ISOPrefix)
	// but are never imported at this level, whatever their geometry: a
	// relation of another country that the extract holds and that would be
	// placed in one of this country's provinces (Vietnam: China's Nansha
	// District r6753263, open sea around its centroid, its first vertex in
	// a Khánh Hòa ring). The dry run counts them. A relation is never in
	// both Include and Exclude (the settings test checks it).
	Exclude     []int64
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
