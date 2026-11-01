// Package views holds the data shapes of the server-rendered pages: one struct
// per page, the items they share, the Source interface the page handlers read
// from, and the Viewer in the request context.
//
// The package only uses the standard library, so internal/db (which fills the
// structs), internal/auth and internal/web (which render them) can all import
// it without importing each other.
//
// Images are carried as storage object keys, never as URLs; templates build
// URLs from IMG_BASE_URL + key. Cell ids are carried for links and map data
// only; user-facing text always uses the place and region names.
package views

import (
	"strconv"
	"time"
)

// Sizes of the lists on the pages. Source implementations use them so that
// handlers, templates and fixtures agree.
const (
	HomeLatest       = 24 // photos in "Latest surveys" on the front page
	HomeNewSurveyors = 8  // surveyors in "New Surveyors"
	PerPage          = 48 // photos per page on paginated lists
	SurveyorsPerPage = 60 // surveyors per page on /surveyors
	FeedItems        = 50 // items in an RSS feed
	TopSurveyors     = 3  // most active surveyors per region on the front page
	RegionTop        = 10 // most active surveyors per list on a region page
	ActiveDays       = 30 // window for "most active" lists
)

// Account rules shared by the forms (pattern attributes) and the handlers.
const (
	NameMinLen  = 3
	NameMaxLen  = 30
	NamePattern = `^[A-Za-z0-9-]{3,30}$` // as in the surveyors.name check
	BioMaxLen   = 500                    // characters, not bytes
)

// Page is the common base of every page. Source implementations fill Title,
// Canonical, Description, Feed and OG; the handler fills Viewer, CSRF and Now
// before rendering (see Base).
type Page struct {
	Viewer      *Viewer   // signed-in visitor, nil for anonymous visitors
	CSRF        string    // token for POST forms (sign-out in the masthead, account forms); set by task 2's middleware
	Now         time.Time // reference time for relative times ("3 hours ago")
	Title       string    // <title> without the site name, e.g. "Pattaya, Chon Buri"
	Canonical   string    // canonical path with query, e.g. "/latest?page=2"; templates prefix the site URL
	Description string    // meta description and og:description; plain text
	Feed        string    // path of the page's RSS feed for <link rel="alternate">; empty if none
	NoIndex     bool      // ask robots not to index (blank plots, account pages, errors)
	OG          OpenGraph // extra Open Graph fields; zero value means a plain website card
	// Type is the typographic direction on <html data-type>, set by the
	// renderer (design round 1 compares directions; see docs/DESIGN_BRIEF.md).
	Type string
}

// Base returns the page base so a render helper can set Viewer, CSRF and Now
// on any page struct that embeds Page.
func (p *Page) Base() *Page { return p }

// Based is implemented by every page struct through its embedded Page.
type Based interface{ Base() *Page }

// OpenGraph holds the Open Graph fields beyond title, description and URL.
type OpenGraph struct {
	Type        string // "article" for photo pages; empty means "website"
	ImageKey    string // object key of the share image (medium, else large); empty for none
	ImageWidth  int    // pixels of the ImageKey variant; 0 if unknown
	ImageHeight int
	ImageAlt    string
}

// Viewer is the signed-in visitor. A visitor who followed a magic link but has
// not chosen a Surveyor name yet has an Email and an empty SurveyorID/Name.
type Viewer struct {
	SurveyorID string // surveyors.id (UUID string); empty until a name is chosen
	Name       string // Surveyor name; empty until chosen
	Email      string
}

// HasName reports whether the viewer has chosen a Surveyor name.
func (v *Viewer) HasName() bool { return v != nil && v.Name != "" }

// ImageKeys are the object keys of a photo's stored variants.
type ImageKeys struct {
	Thumb  string // longest side 480 px
	Medium string // longest side 1200 px; empty for prototype photos (use Large)
	Large  string // longest side 2000 px
}

// Best returns Medium if present, else Large: the key for photo pages and
// share images.
func (k ImageKeys) Best() string {
	if k.Medium != "" {
		return k.Medium
	}
	return k.Large
}

// Licence is the licence a photo is published under.
type Licence struct {
	Name string // short name, e.g. "CC BY-SA 4.0"
	URL  string // deed URL
}

// CCBYSA4 is the licence of every photo (see docs/decisions.md, Home).
var CCBYSA4 = Licence{Name: "CC BY-SA 4.0", URL: "https://creativecommons.org/licenses/by-sa/4.0/"}

// RegionRef identifies a region (a Thai province for now) for links and labels.
type RegionRef struct {
	Slug   string // URL slug from name:en, e.g. "chon-buri"
	NameEn string // e.g. "Chon Buri"
	NameTh string // e.g. "ชลบุรี"; may be empty
}

// PlotRef identifies a plot for links and labels.
type PlotRef struct {
	Cell    string    // H3 res-8 index; for links (/plot/{cell}), maps and the plot page's details line only
	Place   string    // display name: nearest OSM place within 5 km, else the region's name
	PlaceTh string    // Thai name of the place; may be empty
	Region  RegionRef // region containing the plot's centre; zero value outside every region
	Number  int       // running number within the region (/plot/{slug}/{number}); 0 if none yet
	// District is the district (khet/amphoe) containing the centre, without
	// the word "district"; empty where none is imported.
	District string
}

// Name is the plot's user-facing name, "{place}, {district} · {province}
// {number}", e.g. "Hua Mak, Bang Kapi · Bangkok 1000". Without a nearby
// place (the place is the province itself) it is "Bang Kapi · Bangkok 1000";
// a place named like its district is not repeated, and parts that are
// unknown are left out.
func (p PlotRef) Name() string {
	province := p.Region.NameEn
	if p.Number > 0 && province != "" {
		province += " " + strconv.Itoa(p.Number)
	}
	place := p.Place
	if place == p.Region.NameEn {
		place = ""
	}
	local := place
	switch {
	case place == "":
		local = p.District
	case p.District != "" && p.District != place:
		local = place + ", " + p.District
	}
	switch {
	case local == "" && province == "":
		return "A plot"
	case local == "":
		return province
	case province == "":
		return local
	}
	return local + " · " + province
}

// Path is the plot's friendly path, /plot/{region-slug}/{number}, or the
// canonical /plot/{cell} when it has no number yet.
func (p PlotRef) Path() string {
	if p.Number > 0 && p.Region.Slug != "" {
		return "/plot/" + p.Region.Slug + "/" + strconv.Itoa(p.Number)
	}
	return "/plot/" + p.Cell
}

// SurveyorRef identifies a Surveyor for links and labels (/surveyor/{name}).
type SurveyorRef struct {
	Name string
}

// PhotoItem is one photo in a grid, a list or a feed.
type PhotoItem struct {
	ID          string // photos.id (UUID string); links to /photo/{id}
	Keys        ImageKeys
	Width       int // pixels of the large variant; 0 if unknown (prototype photos)
	Height      int
	Plot        PlotRef
	Surveyor    SurveyorRef
	TakenAt     *time.Time // from EXIF; nil if unknown
	UploadedAt  time.Time
	FirstSurvey bool // this photo claimed its plot
}

// Counters are the plain numbers on the front page and region pages.
type Counters struct {
	Plots     int // surveyed plots
	Photos    int
	Surveyors int // front page: all Surveyor accounts; region page: Surveyors with a photo there
	Regions   int // regions with at least one surveyed plot; 0 on region pages
}

// Coverage is the share of a region's land plots that are surveyed.
type Coverage struct {
	Claimed int // surveyed plots in the region
	Land    int // res-8 cells whose centre lies on land in the region
}

// Percent returns Claimed/Land in percent (0–100), 0 when Land is 0.
func (c Coverage) Percent() float64 {
	if c.Land <= 0 {
		return 0
	}
	return 100 * float64(c.Claimed) / float64(c.Land)
}

// Pagination describes one page of a paginated list. Links are built by the
// template from the page's Canonical path without its query and "?page=N".
type Pagination struct {
	Page    int // 1-based
	PerPage int
	Total   int // items across all pages
}

// Pages returns the number of pages (at least 1).
func (p Pagination) Pages() int {
	if p.PerPage <= 0 || p.Total <= p.PerPage {
		return 1
	}
	return (p.Total + p.PerPage - 1) / p.PerPage
}

// HasPrev reports whether there is a previous page.
func (p Pagination) HasPrev() bool { return p.Page > 1 }

// HasNext reports whether there is a next page.
func (p Pagination) HasNext() bool { return p.Page < p.Pages() }

// Prev returns the previous page number.
func (p Pagination) Prev() int { return p.Page - 1 }

// Next returns the next page number.
func (p Pagination) Next() int { return p.Page + 1 }

// SurveyorActivity is a Surveyor with their counts in some window.
type SurveyorActivity struct {
	Surveyor SurveyorRef
	Photos   int // photos uploaded in the window
	Plots    int // plots claimed in the window
}

// Bounds is a lat/lng bounding box for small maps.
type Bounds struct {
	MinLat, MinLng, MaxLat, MaxLng float64
}
