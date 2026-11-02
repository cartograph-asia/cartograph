package importer

import (
	"context"
	"strings"
	"testing"
)

func TestRegionsRefusesACountryWithoutAPage(t *testing.T) {
	// The check comes before anything reads the file or the database.
	_, err := Regions(context.Background(), nil, RegionOptions{PBF: "unused.osm.pbf", Country: "zz"})
	if err == nil || !strings.Contains(err.Error(), "internal/countries") {
		t.Fatalf("err = %v, want a refusal naming internal/countries", err)
	}
}
