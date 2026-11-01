// Package viewstest provides deterministic fixture data for the page structs
// in internal/views and Fake, an in-memory views.Source built from it.
// The data is a small, realistic slice of Chon Buri around Pattaya with real
// H3 res-8 cells, so templates and maps can be developed without a database.
package viewstest

import (
	"fmt"
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Now is the reference time of the fixtures (16:00 in Bangkok).
var Now = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

// Regions of the fixture.
var (
	ChonBuri = views.RegionRef{Slug: "chon-buri", NameEn: "Chon Buri", NameTh: "ชลบุรี"}
	Rayong   = views.RegionRef{Slug: "rayong", NameEn: "Rayong", NameTh: "ระยอง"}
	Bangkok  = views.RegionRef{Slug: "bangkok", NameEn: "Bangkok", NameTh: "กรุงเทพมหานคร"}
)

// region is a fixture region with its land cell count and bounds.
type region struct {
	ref    views.RegionRef
	land   int
	bounds views.Bounds
}

var regions = []region{
	{ChonBuri, 5921, views.Bounds{MinLat: 12.50, MinLng: 100.68, MaxLat: 13.59, MaxLng: 101.73}},
	{Rayong, 4818, views.Bounds{MinLat: 12.53, MinLng: 101.00, MaxLat: 13.08, MaxLng: 101.84}},
	{Bangkok, 2129, views.Bounds{MinLat: 13.49, MinLng: 100.33, MaxLat: 13.96, MaxLng: 100.94}},
}

// plot is a fixture plot with its centre.
type plot struct {
	ref      views.PlotRef
	lat, lng float64
}

func p(cell, place, placeTh, district string, r views.RegionRef, number int, lat, lng float64) plot {
	return plot{views.PlotRef{Cell: cell, Place: place, PlaceTh: placeTh, District: district, Region: r, Number: number}, lat, lng}
}

// Plots of the fixture. The first seven are Pattaya Beach and its ring;
// 88658bd157fffff has no photo and serves as the blank plot.
var plots = []plot{
	p("88658bd11dfffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2417, 12.93190, 100.88670),
	p("88658bd10bfffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2418, 12.93720, 100.88220),
	p("88658bd103fffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2421, 12.94110, 100.88890),
	p("88658bd115fffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2416, 12.93580, 100.89340),
	p("88658bd111fffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2413, 12.92660, 100.89120),
	p("88658bd119fffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2412, 12.92270, 100.88450),
	p("88658bd157fffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2410, 12.92800, 100.88000),
	p("88658bd11bfffff", "Pattaya", "พัทยา", "Bang Lamung", ChonBuri, 2405, 12.92535, 100.87212),
	p("88658bd00dfffff", "Jomtien", "จอมเทียน", "Bang Lamung", ChonBuri, 2280, 12.88587, 100.87317),
	p("88658bd135fffff", "Na Kluea", "นาเกลือ", "Bang Lamung", ChonBuri, 2533, 12.96314, 100.88956),
	p("88658bd86bfffff", "Bang Lamung", "บางละมุง", "Bang Lamung", ChonBuri, 3021, 13.04610, 100.92978),
	p("88658a36a9fffff", "Si Racha", "ศรีราชา", "Si Racha", ChonBuri, 3804, 13.17188, 100.93200),
	p("8865886183fffff", "Sattahip", "สัตหีบ", "Sattahip", ChonBuri, 1187, 12.66285, 100.89934),
	p("8864a4b161fffff", "Bangkok", "กรุงเทพมหานคร", "Pathum Wan", Bangkok, 905, 13.75059, 100.54134),
}

// surveyor is a fixture Surveyor.
type surveyor struct {
	name   string
	joined time.Time
	bio    string
	home   *views.RegionRef
	email  string
}

var surveyors = []surveyor{
	{name: "adrian", joined: Now.Add(-40 * 24 * time.Hour), email: "adrian@example.org", home: &ChonBuri,
		bio: "Walking Pattaya one plot at a time. Mostly early mornings, before the heat."},
	{name: "somchai-k", joined: Now.Add(-12 * 24 * time.Hour), email: "somchai@example.org",
		bio: "Motorbike, old Fujifilm, Sukhumvit Road from Bang Lamung to Sattahip."},
	{name: "Mali", joined: Now.Add(-5 * 24 * time.Hour), email: "mali@example.org", home: &Bangkok},
	{name: "ploy88", joined: Now.Add(-26 * time.Hour), email: "ploy@example.org"},
	{name: "beach-walker", joined: Now.Add(-90 * 24 * time.Hour)}, // placeholder from the prototype, no e-mail, no bio
}

// shot is one fixture photo before expansion: plot index, surveyor index,
// hours before Now.
type shot struct {
	plot, surveyor int
	hoursAgo       float64
}

// shots are listed oldest first; the first shot of each plot claims it.
var shots = []shot{
	{0, 4, 24 * 60}, {7, 4, 24 * 59}, {8, 4, 24 * 58}, // prototype era
	{0, 0, 24 * 38}, {1, 0, 24 * 37}, {2, 0, 24 * 36}, {9, 0, 24 * 30},
	{3, 0, 24 * 21}, {0, 0, 24 * 20}, {4, 0, 24 * 19}, {8, 0, 24 * 15},
	{10, 1, 24 * 11}, {11, 1, 24 * 10}, {12, 1, 24 * 9}, {5, 1, 24 * 8},
	{13, 2, 24 * 4.5}, {13, 2, 24 * 4}, {0, 2, 24 * 3},
	{3, 0, 50}, {7, 0, 49}, {1, 1, 30}, {12, 1, 29},
	{0, 3, 25}, {9, 3, 24.5}, {8, 3, 20}, {11, 1, 8},
	{2, 0, 5}, {10, 1, 3.5}, {0, 0, 2}, {5, 3, 0.75},
}

// photoID returns the deterministic UUID of fixture photo i (0-based, oldest first).
func photoID(i int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1) }

// SurveyorID returns the deterministic UUID of fixture surveyor i.
func SurveyorID(i int) string { return fmt.Sprintf("00000000-0000-4000-9000-%012d", i+1) }

// data is the expanded fixture.
type data struct {
	photos     []views.PhotoItem // oldest first
	byID       map[string]int    // photo id -> index in photos
	bySurveyor map[int][]int     // surveyor index -> photo indexes, oldest first
	byPlot     map[int][]int     // plot index -> photo indexes, oldest first
}

func build() *data {
	d := &data{byID: map[string]int{}, bySurveyor: map[int][]int{}, byPlot: map[int][]int{}}
	claimed := map[int]bool{}
	for i, s := range shots {
		id := photoID(i)
		at := Now.Add(-time.Duration(s.hoursAgo * float64(time.Hour)))
		dir := at.Format("photos/2006/01/")
		keys := views.ImageKeys{Thumb: dir + id + "_thumb.jpg", Medium: dir + id + "_medium.jpg", Large: dir + id + "_large.jpg"}
		w, h := 2000, 1500
		if i < 3 { // prototype photos: no medium variant, no stored size
			keys.Medium, w, h = "", 0, 0
		} else if i%5 == 0 {
			w, h = 1500, 2000
		}
		var taken *time.Time
		if i%4 != 3 {
			t := at.Add(-time.Duration(20+i) * time.Minute)
			taken = &t
		}
		item := views.PhotoItem{
			ID: id, Keys: keys, Width: w, Height: h,
			Plot:     plots[s.plot].ref,
			Surveyor: views.SurveyorRef{Name: surveyors[s.surveyor].name},
			TakenAt:  taken, UploadedAt: at,
			FirstSurvey: !claimed[s.plot],
		}
		claimed[s.plot] = true
		d.byID[id] = len(d.photos)
		d.bySurveyor[s.surveyor] = append(d.bySurveyor[s.surveyor], len(d.photos))
		d.byPlot[s.plot] = append(d.byPlot[s.plot], len(d.photos))
		d.photos = append(d.photos, item)
	}
	return d
}

// FeaturedText is the editorial text of the fixture's plot of the week.
const FeaturedText = "Second Road at six in the morning: the songthaews are still parked and the beach belongs to the joggers. adrian has surveyed this plot three times since September."

// FeaturedPhotoID is the photo set as plot of the week.
var FeaturedPhotoID = photoID(8)

// Viewer returns a signed-in viewer (adrian).
func Viewer() *views.Viewer {
	return &views.Viewer{SurveyorID: SurveyorID(0), Name: "adrian", Email: "adrian@example.org"}
}

// NewViewer returns a viewer who signed in but has not chosen a name yet.
func NewViewer() *views.Viewer {
	return &views.Viewer{Email: "new@example.org"}
}
