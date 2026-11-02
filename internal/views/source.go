package views

import (
	"context"
	"errors"
	"net/http"
)

// ErrNotFound is returned by Source methods when the surveyor, plot, photo or
// region does not exist, or when a page number is past the last page.
// Handlers render PageError with status 404 for it.
var ErrNotFound = errors.New("views: not found")

// ErrGone is returned by Source.Photo for a photo an admin has hidden.
// Handlers render PageError with status 410 for it.
var ErrGone = errors.New("views: gone")

// Source is the read side behind the server-rendered pages. *db.DB
// implements it; viewstest.Fake is an in-memory version for templates and
// handler tests.
//
// Conventions for every method:
//   - page < 1 is treated as 1; a page past the last one (other than 1) is ErrNotFound.
//   - Lookups by name are case-insensitive; the result carries the canonical
//     spelling (Surveyor.Profile.Name) so the handler can redirect.
//   - Page.Title, Canonical, Description, Feed, NoIndex and OG are filled;
//     Viewer, CSRF and Now are left for the handler (Now may be filled too).
type Source interface {
	// Home returns the front page.
	Home(ctx context.Context) (*Home, error)
	// Latest returns one page of /latest.
	Latest(ctx context.Context, page int) (*Latest, error)
	// Surveyors returns one page of /surveyors.
	Surveyors(ctx context.Context, page int) (*SurveyorList, error)
	// Regions returns /regions.
	Regions(ctx context.Context) (*RegionList, error)
	// Surveyor returns one page of a profile, or ErrNotFound (also for an
	// unknown q.Region or a page past the last one).
	Surveyor(ctx context.Context, name string, q SurveyorQuery) (*Surveyor, error)
	// SurveyorPlots returns the plots with a photo of the Surveyor's, for the
	// GPX and GeoJSON exports, or ErrNotFound.
	SurveyorPlots(ctx context.Context, name string) (*SurveyorPlotList, error)
	// Plot returns a plot page. A malformed cell, a cell that is not res 8 or
	// one outside every region is ErrNotFound; a blank cell in a region is a
	// page with Surveyed false.
	Plot(ctx context.Context, cell string) (*Plot, error)
	// PlotCell resolves a plot's running number in a region (by slug, any
	// case) to its cell, for /plot/{slug}/{number}; ErrNotFound if unknown.
	PlotCell(ctx context.Context, slug string, number int) (string, error)
	// Photo returns a photo page, ErrGone for a hidden photo, or ErrNotFound
	// (also for a malformed id).
	Photo(ctx context.Context, id string) (*Photo, error)
	// Region returns one page of a region, or ErrNotFound.
	Region(ctx context.Context, slug string, page int) (*Region, error)
	// RegionRefs returns every region by name, for the home-region select.
	RegionRefs(ctx context.Context) ([]RegionRef, error)

	// LatestFeed returns /latest.rss.
	LatestFeed(ctx context.Context) (*Feed, error)
	// RegionFeed returns /region/{slug}.rss, or ErrNotFound.
	RegionFeed(ctx context.Context, slug string) (*Feed, error)
	// SurveyorFeed returns /surveyor/{name}.rss, or ErrNotFound.
	SurveyorFeed(ctx context.Context, name string) (*Feed, error)
}

// Page names for RenderFunc. Each is one template in internal/web.
const (
	PageHome        = "home"         // *Home
	PageLatest      = "latest"       // *Latest
	PageSurveyors   = "surveyors"    // *SurveyorList
	PageRegions     = "regions"      // *RegionList
	PageSurveyor    = "surveyor"     // *Surveyor
	PagePlot        = "plot"         // *Plot
	PagePhoto       = "photo"        // *Photo
	PageRegion      = "region"       // *Region
	PageFeed        = "feed"         // *Feed, rendered as RSS 2.0 (application/rss+xml)
	PageSignIn      = "signin"       // *SignIn
	PageCheckEmail  = "check-email"  // *CheckEmail
	PageChooseName  = "choose-name"  // *ChooseName
	PageEditProfile = "edit-profile" // *EditProfile
	PageError       = "error"        // *Error
)

// RenderFunc writes page (one of the Page* names) with data and status. data
// is the matching page struct; when it implements Based, the renderer sets
// Viewer and CSRF from the request context (ViewerFrom, CSRFFrom).
// internal/web provides the implementation, so handlers in other packages
// render without importing it.
type RenderFunc func(w http.ResponseWriter, r *http.Request, status int, page string, data any)

type viewerKey struct{}

// WithViewer returns ctx carrying v (nil for anonymous visitors).
func WithViewer(ctx context.Context, v *Viewer) context.Context {
	return context.WithValue(ctx, viewerKey{}, v)
}

// ViewerFrom returns the viewer stored by WithViewer, or nil.
func ViewerFrom(ctx context.Context) *Viewer {
	v, _ := ctx.Value(viewerKey{}).(*Viewer)
	return v
}

// PageConfirmSignIn is the magic-link confirmation page (*ConfirmSignIn);
// task 2 addition.
const PageConfirmSignIn = "confirm-signin"

type csrfKey struct{}

// CSRFField is the name of the hidden form field that carries Page.CSRF in
// every POST form (task 2 addition).
const CSRFField = "csrf"

// WithCSRF returns ctx carrying the CSRF token for the visitor's forms; set
// by internal/auth's middleware (task 2 addition).
func WithCSRF(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfKey{}, token)
}

// CSRFFrom returns the token stored by WithCSRF, or "" when the visitor has
// neither a session nor a sign-in form open (no POST form needs one then).
func CSRFFrom(ctx context.Context) string {
	t, _ := ctx.Value(csrfKey{}).(string)
	return t
}
