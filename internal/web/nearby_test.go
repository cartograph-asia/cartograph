package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/views"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
)

// The photo page and the plot page link to Wikidata's Nearby page for the
// plot's centre, which is also all the plot page says about where it is.
func TestNearbyOnWikidata(t *testing.T) {
	const cell = "88658bd11dfffff"
	lat, lng, err := h3.Centre(cell)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`href="https://www.wikidata.org/wiki/Special:Nearby#/coord/%.4f,%.4f">Nearby on Wikidata</a>`, lat, lng)
	for name, body := range map[string]string{
		"photo": render(t, views.PagePhoto, viewstest.Photo()),
		"plot":  render(t, views.PagePlot, viewstest.Plot()),
	} {
		must(t, body, want)
		if strings.Count(body, "Special:Nearby") != 1 {
			t.Errorf("%s: more than one Nearby link", name)
		}
	}
	// A plot without a position has no link, and an unparsable cell makes none.
	if got := nearbyURL("not-a-cell"); got != "" {
		t.Errorf("nearbyURL of nothing: %q", got)
	}
}
