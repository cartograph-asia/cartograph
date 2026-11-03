package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// OpenCountries returns the countries that have provinces, in the order of
// countries.All (codes not in the list are left out), for texts such as the
// upload error ("… covers so far (Laos and Thailand)"). It reads the set
// cached by ReloadRegions, loading it first if nothing is loaded yet, so it
// follows an import as the region index does.
func (d *DB) OpenCountries(ctx context.Context) ([]countries.Country, error) {
	d.regions.mu.Lock()
	open, loaded := d.regions.open, !d.regions.loadedAt.IsZero()
	d.regions.mu.Unlock()
	if !loaded {
		if err := d.ReloadRegions(ctx); err != nil {
			return nil, err
		}
		d.regions.mu.Lock()
		open = d.regions.open
		d.regions.mu.Unlock()
	}
	var out []countries.Country
	for _, c := range countries.All {
		if open[c.Code] {
			out = append(out, c)
		}
	}
	return out, nil
}

// loadOpenCountries returns the set of country codes that have provinces.
func (d *DB) loadOpenCountries(ctx context.Context) (map[string]bool, error) {
	rows, err := d.Pool.Query(ctx, `select distinct country from regions where kind = 'province'`)
	if err != nil {
		return nil, fmt.Errorf("db: open countries: %w", err)
	}
	codes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("db: open countries: %w", err)
	}
	open := make(map[string]bool, len(codes))
	for _, c := range codes {
		open[c] = true
	}
	return open, nil
}

func countryRef(c countries.Country) views.CountryRef {
	return views.CountryRef{Slug: c.Slug, NameEn: c.NameEn, Native: c.Native, Lang: c.Lang}
}

// countryTotal is a country's figures: sums over its provinces, which is
// exact because every plot belongs to exactly one province.
type countryTotal struct {
	cov    views.Coverage
	last   *time.Time
	b      views.Bounds
	active int // provinces with a surveyed plot
}

// countryTotals returns the totals of every country that has provinces
// (code "" for all, else only that one), keyed by code.
func (v *Views) countryTotals(ctx context.Context, code string) (map[string]countryTotal, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select country, sum(claimed_cells), sum(land_cells), max(last_surveyed_at),
		       min(min_lat), min(min_lng), max(max_lat), max(max_lng), count(*) filter (where claimed_cells > 0)
		from regions where kind = 'province' and ($1 = '' or country = $1) group by country`, code)
	if err != nil {
		return nil, fmt.Errorf("db: country totals: %w", err)
	}
	defer rows.Close()
	out := map[string]countryTotal{}
	for rows.Next() {
		var c string
		var t countryTotal
		if err := rows.Scan(&c, &t.cov.Claimed, &t.cov.Land, &t.last, &t.b.MinLat, &t.b.MinLng, &t.b.MaxLat, &t.b.MaxLng, &t.active); err != nil {
			return nil, fmt.Errorf("db: country totals: %w", err)
		}
		if t.last != nil {
			l := t.last.UTC()
			t.last = &l
		}
		out[c] = t
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: country totals: %w", err)
	}
	return out, nil
}

// countryActivity returns, per country code, the surveyors with the most
// photos since since (zero = all time), up to limit each, ties by name; code
// "" for every country.
func (v *Views) countryActivity(ctx context.Context, code string, since time.Time, limit int) (map[string][]views.SurveyorActivity, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select country, name, display, photos, plots from (
		  select r.country, `+surveyorKey+` as name, coalesce(max(s.display_name), '') as display, count(*) as photos,
		         count(*) filter (where pl.first_photo = ph.id) as plots,
		         row_number() over (partition by r.country order by count(*) desc, `+surveyorKey+`) as rn
		  `+photoFrom+`
		  where r.kind = 'province' and ph.uploaded_at >= $1 and ($3 = '' or r.country = $3)
		  group by r.country, `+surveyorKey+`
		) x where rn <= $2 order by country, rn`, since, limit, code)
	if err != nil {
		return nil, fmt.Errorf("db: country activity: %w", err)
	}
	defer rows.Close()
	out := map[string][]views.SurveyorActivity{}
	for rows.Next() {
		var c string
		var a views.SurveyorActivity
		if err := rows.Scan(&c, &a.Surveyor.Name, &a.Surveyor.Display, &a.Photos, &a.Plots); err != nil {
			return nil, fmt.Errorf("db: country activity: %w", err)
		}
		out[c] = append(out[c], a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: country activity: %w", err)
	}
	return out, nil
}

// countrySummaries returns a row for every country in the list.
func (v *Views) countrySummaries(ctx context.Context) ([]views.CountrySummary, error) {
	tot, err := v.countryTotals(ctx, "")
	if err != nil {
		return nil, err
	}
	top, err := v.countryActivity(ctx, "", v.since(), views.TopSurveyors)
	if err != nil {
		return nil, err
	}
	out := make([]views.CountrySummary, 0, len(countries.All))
	for _, c := range countries.All {
		s := views.CountrySummary{Country: countryRef(c)}
		if t, ok := tot[c.Code]; ok {
			s.Open, s.Coverage, s.LastSurveyedAt, s.TopSurveyors = true, t.cov, t.last, top[c.Code]
		}
		out = append(out, s)
	}
	views.SortCountries(out)
	return out, nil
}

// regionCountry returns the country of region id for the province page's
// link back; the zero value if the country is not in the list.
func (v *Views) regionCountry(ctx context.Context, id int) (views.CountryRef, error) {
	var code string
	if err := v.d.Pool.QueryRow(ctx, `select country from regions where id = $1`, id).Scan(&code); err != nil {
		return views.CountryRef{}, fmt.Errorf("db: region country: %w", err)
	}
	c, ok := countries.ByCode(code)
	if !ok {
		return views.CountryRef{}, nil
	}
	return countryRef(c), nil
}

// Country implements views.Source.
func (v *Views) Country(ctx context.Context, slug string, page int) (*views.Country, error) {
	c, ok := countries.BySlug(slug)
	if !ok {
		return nil, views.ErrNotFound
	}
	ref := countryRef(c)
	path := "/" + c.Slug
	tot, err := v.countryTotals(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	t, open := tot[c.Code]
	if !open {
		if page > 1 {
			return nil, views.ErrNotFound
		}
		out := &views.Country{Page: v.base(c.NameEn, path, "Cartograph does not cover "+c.NameEn+" yet.", ""), Country: ref}
		out.NoIndex = true
		return out, nil
	}
	out := &views.Country{Country: ref, Open: true, Coverage: t.cov, Bounds: t.b, LastSurveyedAt: t.last}
	out.Counters.Plots, out.Counters.Regions = t.cov.Claimed, t.active
	err = v.d.Pool.QueryRow(ctx, `select count(*), count(distinct `+surveyorKey+`)`+photoFrom+`where r.country = $1`, c.Code).
		Scan(&out.Counters.Photos, &out.Counters.Surveyors)
	if err != nil {
		return nil, fmt.Errorf("db: country counters: %w", err)
	}
	pg, err := pagination(page, views.PerPage, out.Counters.Photos)
	if err != nil {
		return nil, err
	}
	out.Pagination = pg
	if out.Latest, err = v.photos(ctx, "where r.country = $1", newestFirst+" limit $2 offset $3", c.Code, pg.PerPage, offset(pg)); err != nil {
		return nil, err
	}
	if out.Provinces, err = v.regionSummaries(ctx, false, c.Code); err != nil {
		return nil, err
	}
	recent, err := v.countryActivity(ctx, c.Code, v.since(), views.RegionTop)
	if err != nil {
		return nil, err
	}
	all, err := v.countryActivity(ctx, c.Code, time.Time{}, views.RegionTop)
	if err != nil {
		return nil, err
	}
	out.ActiveRecent, out.ActiveAllTime = recent[c.Code], all[c.Code]
	out.Page = v.base(c.NameEn, withPage(path, pg.Page),
		fmt.Sprintf("Survey coverage of %s: %.2f %% of %d plots.", c.NameEn, t.cov.Percent(), t.cov.Land), path+".rss")
	return out, nil
}

// CountryFeed implements views.Source.
func (v *Views) CountryFeed(ctx context.Context, slug string) (*views.Feed, error) {
	c, ok := countries.BySlug(slug)
	if !ok {
		return nil, views.ErrNotFound
	}
	tot, err := v.countryTotals(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	if _, open := tot[c.Code]; !open {
		return nil, views.ErrNotFound
	}
	return v.feed(ctx, views.Wordmark+": "+c.NameEn, "/"+c.Slug, "The newest photos in "+c.NameEn+".", "where r.country = $1", c.Code)
}
