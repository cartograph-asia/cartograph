package viewstest

import (
	"context"
	"net/http"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// The constructors below return one filled page each, for rendering
// templates directly. They panic on error, which cannot happen with the
// fixed fixture data. Viewer is nil (anonymous) unless stated.

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

var ctx = context.Background()

// Home returns the front page.
func Home() *views.Home { return must(NewFake().Home(ctx)) }

// Latest returns page 1 of /latest with perPage photos per page (views.PerPage if 0).
func Latest(perPage int) *views.Latest {
	return must((&Fake{PerPage: perPage, d: build()}).Latest(ctx, 1))
}

// Surveyors returns /surveyors.
func Surveyors() *views.SurveyorList { return must(NewFake().Surveyors(ctx, 1)) }

// Regions returns /regions.
func Regions() *views.RegionList { return must(NewFake().Regions(ctx)) }

// Surveyor returns the profile of adrian, the most active fixture surveyor.
func Surveyor() *views.Surveyor {
	return must(NewFake().Surveyor(ctx, "adrian", views.SurveyorQuery{Page: 1}))
}

// PlaceholderSurveyor returns the profile of a prototype placeholder account
// (no bio, no home region).
func PlaceholderSurveyor() *views.Surveyor {
	return must(NewFake().Surveyor(ctx, "beach-walker", views.SurveyorQuery{Page: 1}))
}

// Plot returns Pattaya Beach (88658bd11dfffff), the plot with the most photos.
func Plot() *views.Plot { return must(NewFake().Plot(ctx, "88658bd11dfffff")) }

// BlankPlot returns a plot nobody has surveyed yet.
func BlankPlot() *views.Plot { return must(NewFake().Plot(ctx, "88658bd157fffff")) }

// Photo returns the featured photo (has Prev and Next).
func Photo() *views.Photo { return must(NewFake().Photo(ctx, FeaturedPhotoID)) }

// PrototypePhoto returns the oldest photo: a prototype upload without medium
// variant or size, the first survey of its plot, no Prev.
func PrototypePhoto() *views.Photo { return must(NewFake().Photo(ctx, photoID(0))) }

// Region returns Chon Buri.
func Region() *views.Region { return must(NewFake().Region(ctx, "chon-buri", 1)) }

// Feed returns /latest.rss.
func Feed() *views.Feed { return must(NewFake().LatestFeed(ctx)) }

func authPage(title, canonical string) views.Page {
	return views.Page{Now: Now, Title: title, Canonical: canonical, NoIndex: true, CSRF: "fixture-csrf-token"}
}

// SignIn returns the sign-in form after a failed attempt.
func SignIn() *views.SignIn {
	return &views.SignIn{Page: authPage("Sign in", "/signin"), Email: "adrian@example", Next: "/map",
		Error: "That does not look like an e-mail address."}
}

// CheckEmail returns the "check your e-mail" page.
func CheckEmail() *views.CheckEmail {
	return &views.CheckEmail{Page: authPage("Check your e-mail", "/signin"), Email: "adrian@example.org"}
}

// ChooseName returns the name form after a taken name, for a viewer without a name.
func ChooseName() *views.ChooseName {
	p := &views.ChooseName{Page: authPage("Choose your Surveyor name", "/welcome"), Name: "Adrian", Next: "/map",
		Error: "That name is taken. Names are unique regardless of case."}
	p.Viewer = NewViewer()
	return p
}

// EditProfile returns the profile form for adrian, just saved.
func EditProfile() *views.EditProfile {
	s := surveyors[0]
	p := &views.EditProfile{Page: authPage("Edit profile", "/settings"), Name: s.name, Bio: s.bio,
		HomeRegion: s.home.Slug, Regions: must(NewFake().RegionRefs(ctx)), Saved: true}
	p.Viewer = Viewer()
	return p
}

// NotFound returns the 404 page.
func NotFound() *views.Error {
	return &views.Error{Page: views.Page{Now: Now, Title: "Not found", NoIndex: true},
		Status: http.StatusNotFound, Message: "There is nothing at this address."}
}
