package views

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Home is the front page (/).
type Home struct {
	Page
	Counters     Counters
	Latest       []PhotoItem     // the HomeLatest newest photos, newest first
	Featured     *Featured       // plot of the week; nil only when there is no photo at all
	Regions      []RegionSummary // regions with at least one surveyed plot, most recent activity first
	NewSurveyors []NewSurveyor   // the HomeNewSurveyors most recently joined Surveyors with a photo
}

// Featured is the "Plot of the week" block.
type Featured struct {
	Photo     PhotoItem
	Text      string // editorial text, plain; empty for the fallback
	Editorial bool   // set by hand in the featured table; false = fallback to the latest first survey
}

// RegionSummary is one row of the regions block on the front page and of /regions.
type RegionSummary struct {
	Region         RegionRef
	Coverage       Coverage
	LastSurveyedAt *time.Time         // newest photo in the region; nil if none
	TopSurveyors   []SurveyorActivity // up to TopSurveyors, most photos in the last ActiveDays first
}

// NewSurveyor is one entry of "New Surveyors".
type NewSurveyor struct {
	Surveyor   SurveyorRef
	JoinedAt   time.Time
	FirstPhoto PhotoItem // their oldest photo
}

// Latest is /latest: all photos, newest first, paginated.
type Latest struct {
	Page
	Photos     []PhotoItem
	Pagination Pagination
}

// SurveyorList is /surveyors: all Surveyors, most recently joined first,
// paginated. The front page's Surveyors counter links here.
type SurveyorList struct {
	Page
	Surveyors  []SurveyorListItem
	Pagination Pagination
}

// SurveyorListItem is one Surveyor on /surveyors.
type SurveyorListItem struct {
	Surveyor  SurveyorRef
	JoinedAt  time.Time
	Surveyed  int // plots with at least one of their photos
	Plots     int // plots claimed (first survey); at most Surveyed
	Photos    int
	LastPhoto *PhotoItem // newest photo; nil if none
}

// RegionList is /regions: every region, most recent activity first, then by
// name. The front page's regions counter links here.
type RegionList struct {
	Page
	Regions []RegionSummary
}

// Surveyor is the profile page (/surveyor/{name}).
type Surveyor struct {
	Page
	Profile      SurveyorProfile
	Stats        SurveyorStats
	HomeCoverage *Coverage         // their surveyed plots in the home region of Profile; nil without a home region
	Regions      []SurveyorRegion  // regions with photos of theirs, most plots first
	Years        []SurveyorYear    // years their photos were taken, newest first
	Query        SurveyorQuery     // the view of Photos, as parsed and validated
	FilterRegion *RegionRef        // the region of Query.Region; nil without that filter
	PlotCells    []string          // res-8 cells of every plot with a photo by them, for the lazy map (data, not text)
	PlotPaths    map[string]string // friendly path per cell in PlotCells; cells without a number are missing
	PlotPhotos   map[string]int    // photo count per cell in PlotCells, everyone's, for the small map's three greens
	Bounds       *Bounds           // box around PlotCells; nil if they have no photos
	Photos       []PhotoItem       // in the order of Query, paginated
	Pagination   Pagination
}

// The orders of a profile's photos (SurveyorQuery.Order); the empty string is
// newest uploaded.
const (
	OrderNewest = ""       // newest uploaded first
	OrderOldest = "oldest" // oldest uploaded first
	OrderTaken  = "taken"  // newest taken first, photos without a date last
)

// SurveyorQuery is how a profile's photos are listed: ?order=&region=&year=&one=&page=.
type SurveyorQuery struct {
	Page       int    // 1-based; below 1 is treated as 1
	Order      string // OrderNewest, OrderOldest or OrderTaken
	Region     string // slug of a region (or district) to filter by; empty for all
	Year       int    // year taken to filter by; 0 for all
	OnePerPlot bool   // only the newest photo of theirs in each plot
}

// Filtered reports whether q shows anything but all photos, newest uploaded first.
func (q SurveyorQuery) Filtered() bool {
	return q.Order != OrderNewest || q.Region != "" || q.Year != 0 || q.OnePerPlot
}

// Path is the canonical address of the profile of the Surveyor called name
// seen through q: parameters in key order, defaults and page 1 left out.
func (q SurveyorQuery) Path(name string) string {
	vs := url.Values{}
	if q.Order != OrderNewest {
		vs.Set("order", q.Order)
	}
	if q.Region != "" {
		vs.Set("region", q.Region)
	}
	if q.Year != 0 {
		vs.Set("year", strconv.Itoa(q.Year))
	}
	if q.OnePerPlot {
		vs.Set("one", "plot")
	}
	if q.Page > 1 {
		vs.Set("page", strconv.Itoa(q.Page))
	}
	path := "/surveyor/" + url.PathEscape(name)
	if e := vs.Encode(); e != "" {
		return path + "?" + e
	}
	return path
}

// WithOrder, WithRegion, WithYear and WithOne return q with one thing changed,
// back on page 1, for the links of a profile's controls.
func (q SurveyorQuery) WithOrder(o string) SurveyorQuery    { q.Page, q.Order = 0, o; return q }
func (q SurveyorQuery) WithRegion(s string) SurveyorQuery   { q.Page, q.Region = 0, s; return q }
func (q SurveyorQuery) WithYear(y int) SurveyorQuery        { q.Page, q.Year = 0, y; return q }
func (q SurveyorQuery) WithOnePerPlot(b bool) SurveyorQuery { q.Page, q.OnePerPlot = 0, b; return q }

// SurveyorRegion is one line of a profile's "Where" list.
type SurveyorRegion struct {
	Region RegionRef
	Plots  int // plots in the region with a photo of theirs
	Photos int
	Land   int // land plots in the region
}

// SurveyorYear is one line of a profile's "When" list.
type SurveyorYear struct {
	Year   int
	Photos int
}

// SurveyorPlotList is the data of a profile's GPX and GeoJSON exports.
type SurveyorPlotList struct {
	Surveyor SurveyorRef // canonical spelling
	Plots    []SurveyorPlot
}

// SurveyorPlot is one plot with a photo of the Surveyor's.
type SurveyorPlot struct {
	Plot     PlotRef
	Lat, Lng float64 // centre of the plot
	Photos   int     // their photos in the plot
	First    bool    // they claimed the plot
}

// SurveyorProfile is what a Surveyor says about themselves.
type SurveyorProfile struct {
	Name       string // canonical spelling; handlers redirect other spellings to it
	Display    string // display name; empty for none
	JoinedAt   time.Time
	Bio        string     // plain text, at most BioMaxLen characters; may be empty
	HomeRegion *RegionRef // self-chosen; nil if not set
}

// Label is the name to show: the display name, else the Surveyor name.
func (p SurveyorProfile) Label() string { return label(p.Name, p.Display) }

// SurveyorStats are the counters on a profile.
type SurveyorStats struct {
	Surveyed int // plots with at least one of their photos
	Plots    int // plots they claimed (first survey); at most Surveyed
	Photos   int
	Regions  int // regions with at least one of their photos
	Days     int // distinct dates (UTC) on which their photos were taken
}

// Plot is the plot page (/plot/{cell}). A valid res-8 cell inside a region
// that has no photo yet renders as a blank plot (Surveyed false, NoIndex).
type Plot struct {
	Page
	Plot          PlotRef
	Surveyed      bool
	FirstSurveyor *SurveyorRef // nil for a blank plot
	ClaimedAt     *time.Time   // nil for a blank plot
	PhotoCount    int
	Photos        []PhotoItem  // all photos, oldest first (chronological story of the plot)
	Lat, Lng      float64      // centre of the cell, for the small map
	Neighbours    []CellStatus // the ring of cells around the plot, for the small map
	// Paths is the friendly path per cell on the small map (the plot and its
	// neighbours); cells without a number are missing.
	Paths map[string]string
}

// CellStatus is a cell on a small map, whether it is surveyed and how many
// photos it has (the small map colours it in three steps like /map).
type CellStatus struct {
	Cell     string
	Surveyed bool
	Photos   int
}

// Photo is the photo page (/photo/{id}). Its Page.OG carries the share image.
type Photo struct {
	Page
	Photo    PhotoItem
	Licence  Licence
	Prev     *PhotoItem // older photo of the same plot; nil for the oldest
	Next     *PhotoItem // newer photo of the same plot; nil for the newest
	Position int        // 1-based position in the plot, oldest = 1
	Count    int        // photos in the plot
}

// Region is the region page (/region/{slug}).
type Region struct {
	Page
	Region         RegionRef
	Kind           string // the region's kind from the regions table: "province" or "country"
	Coverage       Coverage
	Counters       Counters // Plots, Photos and Surveyors within the region; Regions is 0
	Bounds         Bounds   // for the coverage map; the map script fetches /api/coverage
	Latest         []PhotoItem
	Pagination     Pagination         // of Latest
	ActiveRecent   []SurveyorActivity // most photos in the last ActiveDays, up to RegionTop
	ActiveAllTime  []SurveyorActivity // most photos ever, up to RegionTop
	LastSurveyedAt *time.Time
}

// KindName is the region's kind as the page's kicker ("Province"), so that
// later countries can show their own word; "Region" if the kind is unknown.
func (r Region) KindName() string {
	if r.Kind == "" {
		return "Region"
	}
	return strings.ToUpper(r.Kind[:1]) + r.Kind[1:]
}

// Feed is an RSS feed: /latest.rss, /region/{slug}.rss, /surveyor/{name}.rss.
// Page.Title is the channel title, Page.Canonical the HTML page it mirrors.
type Feed struct {
	Page
	Items   []PhotoItem // up to FeedItems, newest first
	Updated time.Time   // newest item's UploadedAt; Page.Now if empty
}

// SignIn is the sign-in form (/signin): one e-mail field.
type SignIn struct {
	Page
	InviteOnly bool   // SIGNUP=invite: the page says that sign-up needs an invitation
	Email      string // prefilled after an error
	Next       string // local path to return to after sign-in, e.g. "/map"
	Error      string // plain-text message shown above the form; empty if none
}

// CheckEmail is shown after the sign-in form was sent.
type CheckEmail struct {
	Page
	Email string
}

// ConfirmSignIn is the page behind the magic link (GET /signin/link?token=…).
// It only shows a "Sign in" button: a form that POSTs Token (field "token")
// and the CSRF field to /signin/link, so that mail scanners that prefetch the
// link do not use up the single-use token (task 2 addition).
type ConfirmSignIn struct {
	Page
	Token string
}

// ChooseName is the one-time Surveyor name form shown after the first sign-in.
type ChooseName struct {
	Page
	Display string // the optional display name, prefilled after an error
	Name    string // prefilled after an error
	// ErrorField is the field Error is about: "name" (default) or "display".
	ErrorField string
	Next       string
	Error      string
}

// EditProfile is the profile form (/settings or similar) for display name,
// bio and home region.
type EditProfile struct {
	Page
	Name       string      // the viewer's Surveyor name, shown read-only
	Display    string      // current or submitted display name
	ErrorField string      // the field Error is about: "display", "bio" or "home_region"
	Bio        string      // current or submitted value
	HomeRegion string      // slug of the chosen region; empty for none
	Regions    []RegionRef // options for the home-region select, by name
	Error      string
	Saved      bool // show a short "Saved." note
}

// Error is the error and not-found page.
type Error struct {
	Page
	Status  int    // HTTP status, e.g. 404
	Message string // plain text for the visitor, e.g. "This plot does not exist."
}
