package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/admin"
	"github.com/cartograph-asia/cartograph/internal/auth"
	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
	"github.com/cartograph-asia/cartograph/internal/web"
)

// A country's page lives at /{slug}, so no route of the site may start with
// a country's slug (docs/COUNTRY_PAGES_BRIEF.md). This mounts the same route
// groups as routes() on one router and walks it.
func TestNoRouteStartsWithACountrySlug(t *testing.T) {
	rd, err := web.NewRenderer(web.Config{BaseURL: "https://cartograph.asia", ImgBaseURL: "https://img.cartograph.asia"})
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	a := auth.New(auth.Config{}, nil, nil, rd.RenderFunc(), nil)
	a.Routes(r)
	admin.New(nil, a, nil, rd.RenderFunc()).Routes(r)
	web.NewHandlers(viewstest.NewFake(), rd).Routes(r)
	seen := 0
	err = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		first, _, _ := strings.Cut(strings.TrimPrefix(route, "/"), "/")
		if first == "{country}" {
			return nil // the country pages themselves
		}
		seen++
		if c, ok := countries.BySlug(first); ok {
			t.Errorf("route %s %s starts with the slug of %s", method, route, c.NameEn)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen < 20 {
		t.Errorf("walked only %d routes; the mounts changed", seen)
	}
}
