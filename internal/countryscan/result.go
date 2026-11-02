package countryscan

import "github.com/cartograph-asia/cartograph/internal/countries"

// Result is everything a scan found; Report renders it.
type Result struct {
	Country countries.Country
	Extract string // base name of the file read
	// Neighbours are the base names of the neighbours' cuts read for
	// boundaries (Options.Neighbours); the report names them only when set.
	Neighbours []string
	Res        int
	Workers    int
	Water      bool // water polygons were used

	Shape        Shape
	CountryCells int // cells whose centre is in the outline, sea included
	LandCells    int // of those, not in water

	Levels   []*Level
	Province Proposal
	District Proposal
	// Names holds the name findings of the proposed levels, by role.
	Names  []*NameReport
	Places PlaceReport
	Naming NamingProposal
	Size   Size
}

// Shape is the outline of the country.
type Shape struct {
	Source     string
	RelationID int64 // the admin_level 2 relation, 0 when the outline is a union
	AreaKm2    float64
	Polygons   int
}

// TagStats counts what the relations of a level carry.
type TagStats struct {
	N         int            // relations counted
	ISO       int            // with an ISO3166-2 tag
	ISOHere   int            // with this country's prefix
	Prefixes  map[string]int // ISO3166-2 prefixes (the part before the hyphen)
	NameEn    int            // with name:en
	IntName   int            // with int_name
	HasName   int            // with name
	LatinName int            // whose name is written in Latin letters
	English   int            // with any English-capable name (see englishName)
	Scripts   map[string]int // script of name
	NameKeys  map[string]int // name:* tags, by number of relations
	Examples  []string       // ten names spread over the alphabet
}

// Level is one admin_level in the extract.
type Level struct {
	Level   int
	Extract int    // relations of this level in the file
	Scope   string // "country" (geometry built, centre in the country) or "extract"
	TagStats

	Geometry    bool
	Skipped     int      // in scope, rings not closed in the extract
	SkippedList []string // the first of them, "r123 name"
	SkippedAny  int      // all relations of the level that could not be assembled
	OtherISO    int      // left out: ISO3166-2 of another country
	Outside     int      // left out: centre outside the country
	AreaKm2     float64
	Covered     int // land plots inside at least one relation
	Gaps        int // land plots inside none
	Overlaps    int // land plots inside two or more
	NoLand      int // relations without a land plot
	LandMin     int
	LandMedian  int
	LandMax     int
	FillErrors  int
	CoverPct    float64
	GapPct      float64
	OverlapPct  float64
	AreaPct     float64 // sum of areas, % of the country relation's area
	CapExceeded bool
}

// Proposal is the level proposed for provinces or districts.
type Proposal struct {
	Role   string // "province" or "district"
	Level  int    // 0: none found
	Reason string
	// Candidates are all geometry levels that tile the land well enough.
	Candidates []int
	// Children is, for districts, the number per province (min, median, max)
	// and the districts whose centre lies in no province.
	ChildMin, ChildMedian, ChildMax, Orphans int
}

// NameReport is what the names of one proposed level show.
type NameReport struct {
	Role        string
	Level       int
	N           int
	EnglishKeys map[string]int // tag the English name came from
	NoEnglish   int            // relations without any English-capable name
	LocalKeys   map[string]int
	NoLocal     int
	SameAsLocal int // English name equal to the local name
	Prefixes    []Affix
	Suffixes    []Affix
	LocalPre    []Affix
	LocalSuf    []Affix
	// Slugs
	AccentLoss  int      // names whose plain slug differs from the folded slug
	AccentEx    []string // examples
	Duplicates  []string // slugs (with the parent's for districts) used more than once
	EmptySlug   int
	EmptyAfterF int
}

// KindStat counts place nodes of one place=* value.
type KindStat struct {
	Kind  string
	Total int
	Named int // with a name
	Latin int // with an English-capable name
}

// CityStat is one of the largest cities and the places around it.
type CityStat struct {
	Name       string
	Population int
	Kinds      []KindStat // place nodes within CityRadiusKm, by value
}

// PlaceReport is the place=* findings.
type PlaceReport struct {
	Nodes  int
	Kinds  []KindStat
	Cities []CityStat
}

// CityRadiusKm is how far around a large city place nodes are counted.
const CityRadiusKm = 8

// NamingProposal is the proposed place_naming row.
type NamingProposal struct {
	UrbanKinds    []string
	UrbanRadiusKm int
	Kinds         string
	RadiusKm      int
	Basis         string
}

// Size is what stage 3 will have to hold.
type Size struct {
	LandPlots  int // at the plot resolution (estimated when Res is lower)
	Estimated  bool
	Seconds    int
	MiB        int
	NumberRows int
}
