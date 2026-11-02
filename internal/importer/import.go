package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/cartograph-asia/cartograph/internal/db"
)

// Summary is one country's line of cmd/import's table.
type Summary struct {
	Country       string
	Provinces     int
	Districts     int
	LandPlots     int
	NewNumbers    int
	BorderSkipped int
	Places        int
	Result        string
}

// Import runs one country's import in one process: the checks, then the
// regions, then the places (cmd/import). A dry run reads only and reports
// both; places are then located in the planned provinces. The summary is
// filled as far as the run got, also when it returns an error.
func Import(ctx context.Context, d *db.DB, o Options) (*Summary, error) {
	sum := &Summary{Country: strings.ToLower(o.Country)}
	if sum.Country == "" {
		sum.Country = "th"
	}
	fail := func(err error) (*Summary, error) {
		sum.Result = "failed: " + err.Error()
		if errors.Is(err, ErrChecks) {
			sum.Result = "refused: " + strings.TrimPrefix(err.Error(), "importer: "+sum.Country+": ")
		}
		return sum, err
	}
	pl, err := planRegions(ctx, d, o)
	if err != nil {
		return fail(err)
	}
	st, err := pl.run(ctx, d)
	if st != nil {
		sum.Provinces, sum.Districts, sum.LandPlots, sum.NewNumbers, sum.BorderSkipped = st.Regions, st.Districts, st.LandCells, st.NewNumbers, st.BorderSkipped
	}
	if err != nil && !(o.DryRun && errors.Is(err, ErrChecks) && !pl.hold) {
		return fail(err)
	}
	// A dry run reports the places also when a check failed.
	checksErr := err
	ps, err := places(ctx, d, o, pl)
	if ps != nil {
		sum.Places = ps.Places
	}
	if err != nil {
		return fail(err)
	}
	if checksErr != nil {
		return fail(checksErr)
	}
	sum.Result = "ok"
	if o.DryRun {
		sum.Result = "dry run ok"
	}
	return sum, nil
}

// WriteSummary prints the table of cmd/import.
func WriteSummary(w io.Writer, rows []*Summary) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "country\tprovinces\tdistricts\tland plots\tnew numbers\tborder cells skipped\tplaces\t\tresult")
	for _, s := range rows {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t\t%s\n", s.Country, s.Provinces, s.Districts, s.LandPlots, s.NewNumbers, s.BorderSkipped, s.Places, s.Result)
	}
	tw.Flush()
}
