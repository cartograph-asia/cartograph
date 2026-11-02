package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// Views implements views.Source on the database. It is a separate type
// because *DB already has a Plot method for the JSON API.
type Views struct {
	d   *DB
	now func() time.Time
}

var _ views.Source = (*Views)(nil)

// Views returns the page read side of d.
func (d *DB) Views() *Views { return &Views{d: d, now: time.Now} }

// WithNow returns a copy of v that uses now as the current time (tests).
func (v *Views) WithNow(now func() time.Time) *Views { return &Views{d: v.d, now: now} }

// mediumSide is the longest side of the medium variant (internal/photos).
const mediumSide = 1200

// photoCols selects one views.PhotoItem; use with photoFrom and scanPhoto.
// A photo without an account (prototype, before the placeholder migration)
// shows its free-text surveyor name.
const photoCols = `ph.id, ph.key_thumb, coalesce(ph.key_medium, ''), ph.key_large,
	coalesce(ph.width, 0), coalesce(ph.height, 0), ph.cell,
	coalesce(pl.place_name, r.name_en, ''), coalesce(pl.place_name_th, ''),
	coalesce(r.slug, ''), coalesce(r.name_en, ''), coalesce(r.name_th, ''),
	coalesce(s.name, ph.surveyor), ph.taken_at, ph.uploaded_at, pl.first_photo is not distinct from ph.id,
	coalesce(pn.number, 0), coalesce(dr.name_en, ''), coalesce(s.display_name, '')`

const photoFrom = ` from visible_photos ph
	join plots pl on pl.cell = ph.cell
	left join plot_numbers pn on pn.cell = ph.cell
	left join regions r on r.id = pl.region_id
	left join regions dr on dr.id = pl.district_id
	left join surveyors s on s.id = ph.surveyor_id `

// surveyorKey groups photos by account, else by the free-text name.
const surveyorKey = `coalesce(s.name, ph.surveyor)`

func scanPhoto(row pgx.CollectableRow) (views.PhotoItem, error) {
	var p views.PhotoItem
	var id uuid.UUID
	err := row.Scan(photoDest(&p, &id)...)
	finishPhoto(&p, id)
	return p, err
}

// photoDest returns the scan destinations of photoCols.
func photoDest(p *views.PhotoItem, id *uuid.UUID) []any {
	return []any{id, &p.Keys.Thumb, &p.Keys.Medium, &p.Keys.Large, &p.Width, &p.Height, &p.Plot.Cell,
		&p.Plot.Place, &p.Plot.PlaceTh, &p.Plot.Region.Slug, &p.Plot.Region.NameEn, &p.Plot.Region.NameTh,
		&p.Surveyor.Name, &p.TakenAt, &p.UploadedAt, &p.FirstSurvey, &p.Plot.Number, &p.Plot.District, &p.Surveyor.Display}
}

func finishPhoto(p *views.PhotoItem, id uuid.UUID) {
	p.ID = id.String()
	p.UploadedAt = p.UploadedAt.UTC()
	if p.TakenAt != nil {
		t := p.TakenAt.UTC()
		p.TakenAt = &t
	}
}

func (v *Views) photos(ctx context.Context, where, order string, args ...any) ([]views.PhotoItem, error) {
	rows, err := v.d.Pool.Query(ctx, "select "+photoCols+photoFrom+where+" "+order, args...)
	if err != nil {
		return nil, fmt.Errorf("db: photos: %w", err)
	}
	out, err := pgx.CollectRows(rows, scanPhoto)
	if err != nil {
		return nil, fmt.Errorf("db: photos: %w", err)
	}
	return out, nil
}

const newestFirst = `order by ph.uploaded_at desc, ph.id desc`
const oldestFirst = `order by ph.uploaded_at, ph.id`

func (v *Views) count(ctx context.Context, sql string, args ...any) (int, error) {
	var n int
	if err := v.d.Pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("db: count: %w", err)
	}
	return n, nil
}

// pagination validates page against total and returns limit/offset.
func pagination(page, perPage, total int) (views.Pagination, error) {
	if page < 1 {
		page = 1
	}
	pg := views.Pagination{Page: page, PerPage: perPage, Total: total}
	if page > pg.Pages() {
		return pg, views.ErrNotFound
	}
	return pg, nil
}

func offset(pg views.Pagination) int { return (pg.Page - 1) * pg.PerPage }

func withPage(path string, page int) string {
	if page <= 1 {
		return path
	}
	return fmt.Sprintf("%s?page=%d", path, page)
}

func (v *Views) base(title, canonical, desc, feed string) views.Page {
	return views.Page{Now: v.now().UTC(), Title: title, Canonical: canonical, Description: desc, Feed: feed}
}

// Home implements views.Source.
func (v *Views) Home(ctx context.Context) (*views.Home, error) {
	h := &views.Home{Page: v.base("", "/", "A photo for every plot of land in Southeast and East Asia, surveyed by its people.", "/latest.rss")}
	err := v.d.Pool.QueryRow(ctx, `select
		(select count(*) from plots where photo_count > 0),
		(select count(*) from visible_photos),
		(select count(*) from surveyors),
		(select count(*) from regions where kind = 'province' and claimed_cells > 0)`,
	).Scan(&h.Counters.Plots, &h.Counters.Photos, &h.Counters.Surveyors, &h.Counters.Regions)
	if err != nil {
		return nil, fmt.Errorf("db: home counters: %w", err)
	}
	if h.Latest, err = v.photos(ctx, "", newestFirst+" limit $1", views.HomeLatest); err != nil {
		return nil, err
	}
	if h.Featured, err = v.featured(ctx); err != nil {
		return nil, err
	}
	if h.Regions, err = v.regionSummaries(ctx, true, ""); err != nil {
		return nil, err
	}
	if h.NewSurveyors, err = v.newSurveyors(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

// featured returns the current featured entry, else the latest first survey.
func (v *Views) featured(ctx context.Context) (*views.Featured, error) {
	var id uuid.UUID
	var body string
	err := v.d.Pool.QueryRow(ctx, `
		select photo_id, body from featured
		where starts_at <= $1 and (ends_at is null or ends_at > $1)
		order by starts_at desc, id desc limit 1`, v.now()).Scan(&id, &body)
	if err == nil {
		ps, err := v.photos(ctx, "where ph.id = $1", "", id)
		if err != nil {
			return nil, err
		}
		if len(ps) == 1 {
			return &views.Featured{Photo: ps[0], Text: body, Editorial: true}, nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("db: featured: %w", err)
	}
	ps, err := v.photos(ctx, "where pl.first_photo = ph.id", newestFirst+" limit 1")
	if err != nil || len(ps) == 0 {
		return nil, err
	}
	return &views.Featured{Photo: ps[0]}, nil
}

func (v *Views) newSurveyors(ctx context.Context) ([]views.NewSurveyor, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select s.name, coalesce(s.display_name, ''), s.joined_at, f.id from surveyors s
		join lateral (select id from visible_photos where surveyor_id = s.id order by uploaded_at, id limit 1) f on true
		order by s.joined_at desc, s.id limit $1`, views.HomeNewSurveyors)
	if err != nil {
		return nil, fmt.Errorf("db: new surveyors: %w", err)
	}
	type row struct {
		name, display string
		joined        time.Time
		photo         uuid.UUID
	}
	rs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		return x, r.Scan(&x.name, &x.display, &x.joined, &x.photo)
	})
	if err != nil {
		return nil, fmt.Errorf("db: new surveyors: %w", err)
	}
	ids := make([]uuid.UUID, len(rs))
	for i, r := range rs {
		ids[i] = r.photo
	}
	byID, err := v.photosByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]views.NewSurveyor, 0, len(rs))
	for _, r := range rs {
		out = append(out, views.NewSurveyor{Surveyor: views.SurveyorRef{Name: r.name, Display: r.display}, JoinedAt: r.joined.UTC(), FirstPhoto: byID[r.photo.String()]})
	}
	return out, nil
}

func (v *Views) photosByID(ctx context.Context, ids []uuid.UUID) (map[string]views.PhotoItem, error) {
	out := map[string]views.PhotoItem{}
	if len(ids) == 0 {
		return out, nil
	}
	ps, err := v.photos(ctx, "where ph.id = any($1)", "", ids)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		out[p.ID] = p
	}
	return out, nil
}

// regionRow is a region with its stored counters.
type regionRow struct {
	id   int
	ref  views.RegionRef
	kind string
	cov  views.Coverage
	last *time.Time
	b    views.Bounds
}

const regionCols = `id, kind, slug, name_en, coalesce(name_th, ''), claimed_cells, land_cells, last_surveyed_at,
	min_lat, min_lng, max_lat, max_lng`

func scanRegion(r pgx.CollectableRow) (regionRow, error) {
	var x regionRow
	err := r.Scan(&x.id, &x.kind, &x.ref.Slug, &x.ref.NameEn, &x.ref.NameTh, &x.cov.Claimed, &x.cov.Land, &x.last,
		&x.b.MinLat, &x.b.MinLng, &x.b.MaxLat, &x.b.MaxLng)
	if x.last != nil {
		t := x.last.UTC()
		x.last = &t
	}
	return x, err
}

// regionSummaries returns provinces (of one country, "" for all; only those
// with a surveyed plot if activeOnly) by most recent activity, then name, with their top surveyors.
func (v *Views) regionSummaries(ctx context.Context, activeOnly bool, country string) ([]views.RegionSummary, error) {
	where := "where kind = 'province' and ($1 = '' or country = $1)"
	if activeOnly {
		where += " and claimed_cells > 0"
	}
	rows, err := v.d.Pool.Query(ctx, `select `+regionCols+` from regions `+where+`
		order by last_surveyed_at desc nulls last, name_en`, country)
	if err != nil {
		return nil, fmt.Errorf("db: regions: %w", err)
	}
	rs, err := pgx.CollectRows(rows, scanRegion)
	if err != nil {
		return nil, fmt.Errorf("db: regions: %w", err)
	}
	top, err := v.activity(ctx, "", v.since(), views.TopSurveyors)
	if err != nil {
		return nil, err
	}
	out := make([]views.RegionSummary, 0, len(rs))
	for _, r := range rs {
		out = append(out, views.RegionSummary{Region: r.ref, Coverage: r.cov, LastSurveyedAt: r.last, TopSurveyors: top[r.id]})
	}
	return out, nil
}

func (v *Views) since() time.Time { return v.now().Add(-views.ActiveDays * 24 * time.Hour) }

// activity returns, per region id, the surveyors with the most photos since
// since (zero = all time), up to limit each, ties by name. regionSlug limits
// it to one region.
func (v *Views) activity(ctx context.Context, regionSlug string, since time.Time, limit int) (map[int][]views.SurveyorActivity, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select region_id, name, display, photos, plots from (
		  select pl.region_id, `+surveyorKey+` as name, coalesce(max(s.display_name), '') as display, count(*) as photos,
		         count(*) filter (where pl.first_photo = ph.id) as plots,
		         row_number() over (partition by pl.region_id order by count(*) desc, `+surveyorKey+`) as rn
		  `+photoFrom+`
		  where pl.region_id is not null and ph.uploaded_at >= $1 and ($3 = '' or r.slug = $3)
		  group by pl.region_id, `+surveyorKey+`
		) x where rn <= $2 order by region_id, rn`, since, limit, regionSlug)
	if err != nil {
		return nil, fmt.Errorf("db: activity: %w", err)
	}
	defer rows.Close()
	out := map[int][]views.SurveyorActivity{}
	for rows.Next() {
		var id int
		var a views.SurveyorActivity
		if err := rows.Scan(&id, &a.Surveyor.Name, &a.Surveyor.Display, &a.Photos, &a.Plots); err != nil {
			return nil, fmt.Errorf("db: activity: %w", err)
		}
		out[id] = append(out[id], a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: activity: %w", err)
	}
	return out, nil
}

// Latest implements views.Source.
func (v *Views) Latest(ctx context.Context, page int) (*views.Latest, error) {
	total, err := v.count(ctx, `select count(*) from visible_photos`)
	if err != nil {
		return nil, err
	}
	pg, err := pagination(page, views.PerPage, total)
	if err != nil {
		return nil, err
	}
	ps, err := v.photos(ctx, "", newestFirst+" limit $1 offset $2", pg.PerPage, offset(pg))
	if err != nil {
		return nil, err
	}
	return &views.Latest{
		Page:   v.base("Latest surveys", withPage("/latest", pg.Page), "The newest photos on Cartograph.", "/latest.rss"),
		Photos: ps, Pagination: pg,
	}, nil
}

// Surveyors implements views.Source.
func (v *Views) Surveyors(ctx context.Context, page int) (*views.SurveyorList, error) {
	total, err := v.count(ctx, `select count(*) from surveyors`)
	if err != nil {
		return nil, err
	}
	pg, err := pagination(page, views.SurveyorsPerPage, total)
	if err != nil {
		return nil, err
	}
	rows, err := v.d.Pool.Query(ctx, `
		select s.name, coalesce(s.display_name, ''), s.joined_at,
		  (select count(distinct cell) from visible_photos where surveyor_id = s.id),
		  (select count(*) from plots where first_surveyor_id = s.id),
		  (select count(*) from visible_photos where surveyor_id = s.id),
		  (select id from visible_photos where surveyor_id = s.id order by uploaded_at desc, id desc limit 1)
		from surveyors s order by s.joined_at desc, s.id limit $1 offset $2`, pg.PerPage, offset(pg))
	if err != nil {
		return nil, fmt.Errorf("db: surveyors: %w", err)
	}
	type row struct {
		item views.SurveyorListItem
		last *uuid.UUID
	}
	rs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.item.Surveyor.Name, &x.item.Surveyor.Display, &x.item.JoinedAt, &x.item.Surveyed, &x.item.Plots, &x.item.Photos, &x.last)
		x.item.JoinedAt = x.item.JoinedAt.UTC()
		return x, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: surveyors: %w", err)
	}
	var ids []uuid.UUID
	for _, r := range rs {
		if r.last != nil {
			ids = append(ids, *r.last)
		}
	}
	byID, err := v.photosByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &views.SurveyorList{
		Page:       v.base("Surveyors", withPage("/surveyors", pg.Page), "Everyone who surveys for Cartograph.", ""),
		Pagination: pg,
		Surveyors:  []views.SurveyorListItem{},
	}
	for _, r := range rs {
		if r.last != nil {
			p := byID[r.last.String()]
			r.item.LastPhoto = &p
		}
		out.Surveyors = append(out.Surveyors, r.item)
	}
	return out, nil
}

// Regions implements views.Source.
func (v *Views) Regions(ctx context.Context) (*views.RegionList, error) {
	rs, err := v.regionSummaries(ctx, false, "")
	if err != nil {
		return nil, err
	}
	cs, err := v.countrySummaries(ctx)
	if err != nil {
		return nil, err
	}
	return &views.RegionList{Page: v.base("Regions", "/regions", "Survey coverage by country and province.", ""), Countries: cs, Regions: rs}, nil
}

// surveyorRow is a surveyor looked up by name.
type surveyorRow struct {
	id      uuid.UUID
	profile views.SurveyorProfile
}

func (v *Views) surveyor(ctx context.Context, name string) (*surveyorRow, error) {
	var s surveyorRow
	var slug, en, th *string
	err := v.d.Pool.QueryRow(ctx, `
		select s.id, s.name, coalesce(s.display_name, ''), s.joined_at, s.bio, r.slug, r.name_en, r.name_th
		from surveyors s left join regions r on r.id = s.home_region_id
		where lower(s.name) = lower($1)`, name,
	).Scan(&s.id, &s.profile.Name, &s.profile.Display, &s.profile.JoinedAt, &s.profile.Bio, &slug, &en, &th)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: surveyor: %w", err)
	}
	s.profile.JoinedAt = s.profile.JoinedAt.UTC()
	if slug != nil {
		s.profile.HomeRegion = &views.RegionRef{Slug: *slug, NameEn: *en, NameTh: deref(th)}
	}
	return &s, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func surveyorPath(name string) string { return "/surveyor/" + url.PathEscape(name) }

// takenOn is the date a photo with alias ph was taken on the clock where it
// was taken (taken_at_local), else the UTC date of taken_at; null without either.
func takenOn(ph string) string {
	return "coalesce(" + ph + ".taken_at_local, " + ph + ".taken_at at time zone 'UTC')"
}

var takenLocal = takenOn("ph")

// uploadedOn is the date a photo with alias ph was uploaded, on Bangkok's
// clock like every date the pages show (internal/web's zone).
func uploadedOn(ph string) string { return ph + ".uploaded_at at time zone 'Asia/Bangkok'" }

// surveyorOrder maps a views.SurveyorQuery order to its ORDER BY clause.
var surveyorOrder = map[string]string{
	views.OrderNewest: newestFirst,
	views.OrderOldest: oldestFirst,
	views.OrderTaken:  `order by ` + takenLocal + ` desc nulls last, ph.uploaded_at desc, ph.id desc`,
}

// surveyorWhere returns the conditions of q on the photo alias ph and the plot
// alias pl, joined as in photoFrom, and the arguments they bind, numbered from
// next. The region filter matches a plot's province and its district.
func surveyorWhere(q views.SurveyorQuery, regionID *int, next int, ph, pl string) (string, []any) {
	var sql string
	var args []any
	if regionID != nil {
		sql += fmt.Sprintf(" and (%s.region_id = $%d or %s.district_id = $%d)", pl, next, pl, next)
		args = append(args, *regionID)
		next++
	}
	if q.Year != 0 {
		sql += fmt.Sprintf(" and extract(year from %s) = $%d", takenOn(ph), next)
		args = append(args, q.Year)
		next++
	}
	if q.Uploaded != 0 {
		sql += fmt.Sprintf(" and extract(year from %s) = $%d", uploadedOn(ph), next)
		args = append(args, q.Uploaded)
	}
	return sql, args
}

// Surveyor implements views.Source.
func (v *Views) Surveyor(ctx context.Context, name string, q views.SurveyorQuery) (*views.Surveyor, error) {
	s, err := v.surveyor(ctx, name)
	if err != nil {
		return nil, err
	}
	order, ok := surveyorOrder[q.Order]
	if !ok {
		return nil, views.ErrNotFound
	}
	out := &views.Surveyor{Profile: s.profile}
	var regionID *int
	if q.Region != "" {
		var id int
		var ref views.RegionRef
		err := v.d.Pool.QueryRow(ctx, `select id, slug, name_en, coalesce(name_th, '') from regions where slug = lower($1)`, q.Region).
			Scan(&id, &ref.Slug, &ref.NameEn, &ref.NameTh)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, views.ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("db: surveyor region filter: %w", err)
		}
		regionID, q.Region, out.FilterRegion = &id, ref.Slug, &ref
	}
	err = v.d.Pool.QueryRow(ctx, `
		select
		  count(distinct ph.cell), (select count(*) from plots where first_surveyor_id = $1),
		  count(*), count(distinct pl.region_id), count(distinct `+takenLocal+`::date)
		from visible_photos ph join plots pl on pl.cell = ph.cell where ph.surveyor_id = $1`, s.id,
	).Scan(&out.Stats.Surveyed, &out.Stats.Plots, &out.Stats.Photos, &out.Stats.Regions, &out.Stats.Days)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor stats: %w", err)
	}

	// The photos of the view: filtered, optionally one per plot, ordered.
	cond, fargs := surveyorWhere(q, regionID, 2, "ph", "pl")
	where := "where ph.surveyor_id = $1" + cond
	args := append([]any{s.id}, fargs...)
	if q.OnePerPlot {
		sub, _ := surveyorWhere(q, regionID, 2, "ph2", "pl2")
		where += ` and ph.id in (select distinct on (ph2.cell) ph2.id from visible_photos ph2 join plots pl2 on pl2.cell = ph2.cell
			where ph2.surveyor_id = $1` + sub + ` order by ph2.cell, ph2.uploaded_at desc, ph2.id desc)`
	}
	total, err := v.count(ctx, `select count(*) from visible_photos ph join plots pl on pl.cell = ph.cell `+where, args...)
	if err != nil {
		return nil, err
	}
	pg, err := pagination(q.Page, views.PerPage, total)
	if err != nil {
		return nil, err
	}
	q.Page = pg.Page
	out.Query, out.Pagination = q, pg
	n := len(args)
	if out.Photos, err = v.photos(ctx, where, fmt.Sprintf("%s limit $%d offset $%d", order, n+1, n+2), append(args, pg.PerPage, offset(pg))...); err != nil {
		return nil, err
	}

	if out.Regions, err = v.surveyorRegions(ctx, s.id); err != nil {
		return nil, err
	}
	if out.Years, err = v.surveyorYears(ctx, s.id, takenLocal); err != nil {
		return nil, err
	}
	if out.UploadYears, err = v.surveyorYears(ctx, s.id, uploadedOn("ph")); err != nil {
		return nil, err
	}
	if h := s.profile.HomeRegion; h != nil {
		var c views.Coverage
		err := v.d.Pool.QueryRow(ctx, `
			select (select count(distinct ph.cell) from visible_photos ph join plots pl on pl.cell = ph.cell
			        where ph.surveyor_id = $1 and (pl.region_id = r.id or pl.district_id = r.id)), r.land_cells
			from regions r where r.slug = $2`, s.id, h.Slug).Scan(&c.Claimed, &c.Land)
		if err != nil {
			return nil, fmt.Errorf("db: surveyor home coverage: %w", err)
		}
		out.HomeCoverage = &c
	}

	rows, err := v.d.Pool.Query(ctx, `select pl.cell, pl.photo_count from plots pl
		where pl.cell in (select cell from visible_photos where surveyor_id = $1) order by pl.cell`, s.id)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor cells: %w", err)
	}
	out.PlotPhotos = map[string]int{}
	var cell string
	var np int
	if _, err = pgx.ForEachRow(rows, []any{&cell, &np}, func() error {
		out.PlotCells = append(out.PlotCells, cell)
		out.PlotPhotos[cell] = np
		return nil
	}); err != nil {
		return nil, fmt.Errorf("db: surveyor cells: %w", err)
	}
	if out.PlotPaths, err = v.plotPaths(ctx, out.PlotCells); err != nil {
		return nil, err
	}
	if out.Bounds, err = cellBounds(out.PlotCells); err != nil {
		return nil, err
	}
	desc := s.profile.Bio
	if desc == "" {
		desc = fmt.Sprintf("Photos by %s on Cartograph.", s.profile.Label())
	}
	out.Page = v.base(s.profile.Label(), q.Path(s.profile.Name), desc, surveyorPath(s.profile.Name)+".rss")
	out.NoIndex = q.Filtered()
	return out, nil
}

// surveyorRegions is the "Where" list: provinces with photos of the
// Surveyor's, most plots first.
func (v *Views) surveyorRegions(ctx context.Context, id uuid.UUID) ([]views.SurveyorRegion, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select r.slug, r.name_en, coalesce(r.name_th, ''), count(distinct ph.cell), count(*), r.land_cells
		from visible_photos ph join plots pl on pl.cell = ph.cell join regions r on r.id = pl.region_id
		where ph.surveyor_id = $1 group by r.id order by 4 desc, r.name_en, r.id`, id)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor regions: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (x views.SurveyorRegion, err error) {
		err = r.Scan(&x.Region.Slug, &x.Region.NameEn, &x.Region.NameTh, &x.Plots, &x.Photos, &x.Land)
		return
	})
	if err != nil {
		return nil, fmt.Errorf("db: surveyor regions: %w", err)
	}
	return out, nil
}

// surveyorYears is a "When" list: the years of date (takenLocal or
// uploadedOn) over their photos, newest first; photos without the date are
// left out.
func (v *Views) surveyorYears(ctx context.Context, id uuid.UUID, date string) ([]views.SurveyorYear, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select extract(year from `+date+`)::int as y, count(*) from visible_photos ph
		where ph.surveyor_id = $1 and `+date+` is not null group by y order by y desc`, id)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor years: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (x views.SurveyorYear, err error) {
		err = r.Scan(&x.Year, &x.Photos)
		return
	})
	if err != nil {
		return nil, fmt.Errorf("db: surveyor years: %w", err)
	}
	return out, nil
}

// SurveyorPlots implements views.Source.
func (v *Views) SurveyorPlots(ctx context.Context, name string) (*views.SurveyorPlotList, error) {
	s, err := v.surveyor(ctx, name)
	if err != nil {
		return nil, err
	}
	rows, err := v.d.Pool.Query(ctx, `
		select ph.cell, coalesce(pl.place_name, r.name_en, ''), coalesce(pl.place_name_th, ''),
		       coalesce(r.slug, ''), coalesce(r.name_en, ''), coalesce(r.name_th, ''),
		       coalesce(pn.number, 0), coalesce(dr.name_en, ''), count(*), pl.first_surveyor_id is not distinct from $1
		from visible_photos ph
		join plots pl on pl.cell = ph.cell
		left join plot_numbers pn on pn.cell = ph.cell
		left join regions r on r.id = pl.region_id
		left join regions dr on dr.id = pl.district_id
		where ph.surveyor_id = $1
		group by ph.cell, pl.place_name, pl.place_name_th, pl.first_surveyor_id, r.id, pn.number, dr.id
		order by ph.cell`, s.id)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor plots: %w", err)
	}
	out := &views.SurveyorPlotList{Surveyor: views.SurveyorRef{Name: s.profile.Name, Display: s.profile.Display}}
	out.Plots, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (x views.SurveyorPlot, err error) {
		p := &x.Plot
		err = r.Scan(&p.Cell, &p.Place, &p.PlaceTh, &p.Region.Slug, &p.Region.NameEn, &p.Region.NameTh, &p.Number, &p.District, &x.Photos, &x.First)
		return
	})
	if err != nil {
		return nil, fmt.Errorf("db: surveyor plots: %w", err)
	}
	for i := range out.Plots {
		x := &out.Plots[i]
		if x.Lat, x.Lng, err = h3.Centre(x.Plot.Cell); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Plot implements views.Source.
func (v *Views) Plot(ctx context.Context, cell string) (*views.Plot, error) {
	cell, res, err := h3.Parse(cell)
	if err != nil || res != h3.PlotRes {
		return nil, views.ErrNotFound
	}
	out := &views.Plot{Plot: views.PlotRef{Cell: cell}}
	var place, placeTh, firstName, slug, en, th, district *string
	var firstDisplay string
	var claimedAt *time.Time
	err = v.d.Pool.QueryRow(ctx, `
		select pl.photo_count, pl.claimed_at, pl.place_name, pl.place_name_th,
		       coalesce(s.name, pl.first_surveyor), coalesce(s.display_name, ''), r.slug, r.name_en, r.name_th, dr.name_en
		from plots pl
		left join surveyors s on s.id = pl.first_surveyor_id
		left join regions r on r.id = pl.region_id
		left join regions dr on dr.id = pl.district_id
		where pl.cell = $1 and pl.photo_count > 0`, cell,
	).Scan(&out.PhotoCount, &claimedAt, &place, &placeTh, &firstName, &firstDisplay, &slug, &en, &th, &district)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("db: plot page: %w", err)
	default:
		out.Surveyed = true
		t := claimedAt.UTC()
		out.ClaimedAt = &t
		if firstName != nil {
			out.FirstSurveyor = &views.SurveyorRef{Name: *firstName, Display: firstDisplay}
		}
		if slug != nil {
			out.Plot.Region = views.RegionRef{Slug: *slug, NameEn: *en, NameTh: deref(th)}
		}
		out.Plot.Place, out.Plot.PlaceTh, out.Plot.District = deref(place), deref(placeTh), deref(district)
	}
	if out.Plot.Place == "" || out.Plot.Region.Slug == "" || out.Plot.District == "" {
		// Blank plot, or a surveyed plot not yet backfilled: locate it now.
		loc, err := v.d.LocatePlot(ctx, cell)
		if errors.Is(err, ErrOutsideRegions) && !out.Surveyed {
			return nil, views.ErrNotFound
		}
		if err != nil && !errors.Is(err, ErrOutsideRegions) {
			return nil, err
		}
		if loc != nil {
			if out.Plot.Region.Slug == "" {
				out.Plot.Region, err = v.regionRef(ctx, loc.RegionID)
				if err != nil {
					return nil, err
				}
			}
			if out.Plot.Place == "" {
				out.Plot.Place, out.Plot.PlaceTh = loc.PlaceEn, deref(loc.PlaceTh)
			}
			if out.Plot.District == "" {
				out.Plot.District = loc.District
			}
		}
	}
	err = v.d.Pool.QueryRow(ctx, `select number from plot_numbers where cell = $1`, cell).Scan(&out.Plot.Number)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("db: plot number: %w", err)
	}
	if out.Lat, out.Lng, err = h3.Centre(cell); err != nil {
		return nil, err
	}
	if out.Surveyed {
		if out.Photos, err = v.photos(ctx, "where ph.cell = $1", oldestFirst, cell); err != nil {
			return nil, err
		}
	}
	ring, err := h3.Ring(cell, 1)
	if err != nil {
		return nil, err
	}
	rows, err := v.d.Pool.Query(ctx, `select cell, photo_count from plots where cell = any($1) and photo_count > 0`, ring)
	if err != nil {
		return nil, fmt.Errorf("db: plot neighbours: %w", err)
	}
	photos := map[string]int{}
	var nc string
	var nn int
	if _, err = pgx.ForEachRow(rows, []any{&nc, &nn}, func() error {
		photos[nc] = nn
		return nil
	}); err != nil {
		return nil, fmt.Errorf("db: plot neighbours: %w", err)
	}
	for _, c := range ring {
		out.Neighbours = append(out.Neighbours, views.CellStatus{Cell: c, Surveyed: photos[c] > 0, Photos: photos[c]})
	}
	if out.Paths, err = v.plotPaths(ctx, append([]string{cell}, ring...)); err != nil {
		return nil, err
	}

	name := out.Plot.Name()
	out.Page = v.base(name, "/plot/"+cell, "", "")
	if out.Surveyed {
		who := ""
		if out.FirstSurveyor != nil {
			who = " by " + out.FirstSurveyor.Label()
		}
		out.Description = fmt.Sprintf("%s, first surveyed%s. %s.", name, who, plural(out.PhotoCount, "photo"))
	} else {
		out.NoIndex = true
		out.Description = fmt.Sprintf("%s: nobody has surveyed this plot yet.", name)
	}
	return out, nil
}

// plotPaths returns the friendly path of every numbered cell in cells.
func (v *Views) plotPaths(ctx context.Context, cells []string) (map[string]string, error) {
	rows, err := v.d.Pool.Query(ctx, `
		select n.cell, r.slug, n.number from plot_numbers n join regions r on r.id = n.region_id
		where n.cell = any($1)`, cells)
	if err != nil {
		return nil, fmt.Errorf("db: plot paths: %w", err)
	}
	out := map[string]string{}
	for rows.Next() {
		var p views.PlotRef
		if err := rows.Scan(&p.Cell, &p.Region.Slug, &p.Number); err != nil {
			return nil, fmt.Errorf("db: plot paths: %w", err)
		}
		out[p.Cell] = p.Path()
	}
	return out, rows.Err()
}

// cellBounds is the box around the boundaries of cells, or nil for none.
// Public bounds are built from plots only: exact photo coordinates never
// leave the database (docs/decisions.md, 2026-11-01).
func cellBounds(cells []string) (*views.Bounds, error) {
	if len(cells) == 0 {
		return nil, nil
	}
	b := views.Bounds{MinLat: math.Inf(1), MinLng: math.Inf(1), MaxLat: math.Inf(-1), MaxLng: math.Inf(-1)}
	for _, c := range cells {
		ring, err := h3.Boundary(c)
		if err != nil {
			return nil, fmt.Errorf("db: cell bounds: %w", err)
		}
		for _, p := range ring {
			b.MinLat, b.MaxLat = math.Min(b.MinLat, p.Lat), math.Max(b.MaxLat, p.Lat)
			b.MinLng, b.MaxLng = math.Min(b.MinLng, p.Lng), math.Max(b.MaxLng, p.Lng)
		}
	}
	return &b, nil
}

// PlotCell implements views.Source.
func (v *Views) PlotCell(ctx context.Context, slug string, number int) (string, error) {
	var cell string
	err := v.d.Pool.QueryRow(ctx, `
		select n.cell from plot_numbers n join regions r on r.id = n.region_id
		where lower(r.slug) = lower($1) and n.number = $2`, slug, number).Scan(&cell)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", views.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("db: plot by number: %w", err)
	}
	return cell, nil
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func (v *Views) regionRef(ctx context.Context, id int) (views.RegionRef, error) {
	var r views.RegionRef
	err := v.d.Pool.QueryRow(ctx, `select slug, name_en, coalesce(name_th, '') from regions where id = $1`, id).
		Scan(&r.Slug, &r.NameEn, &r.NameTh)
	if err != nil {
		return r, fmt.Errorf("db: region ref: %w", err)
	}
	return r, nil
}

// Photo implements views.Source.
func (v *Views) Photo(ctx context.Context, id string) (*views.Photo, error) {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, views.ErrNotFound
	}
	ps, err := v.photos(ctx, "where ph.id = $1", "", uid)
	if err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		var hidden bool
		err := v.d.Pool.QueryRow(ctx, `select hidden_at is not null from photos where id = $1`, uid).Scan(&hidden)
		if err == nil && hidden {
			return nil, views.ErrGone
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("db: photo: %w", err)
		}
		return nil, views.ErrNotFound
	}
	ph := ps[0]
	var licence string
	if err := v.d.Pool.QueryRow(ctx, `select licence from visible_photos where id = $1`, uid).Scan(&licence); err != nil {
		return nil, fmt.Errorf("db: photo licence: %w", err)
	}
	out := &views.Photo{Photo: ph, Licence: views.Licence{Name: licence}}
	if licence == views.CCBYSA4.Name {
		out.Licence = views.CCBYSA4
	}
	if out.Camera, err = v.camera(ctx, uid); err != nil {
		return nil, err
	}
	rows, err := v.d.Pool.Query(ctx, `select id from visible_photos where cell = $1 order by uploaded_at, id`, ph.Plot.Cell)
	if err != nil {
		return nil, fmt.Errorf("db: photo siblings: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("db: photo siblings: %w", err)
	}
	out.Count = len(ids)
	var near []uuid.UUID
	for i, x := range ids {
		if x != uid {
			continue
		}
		out.Position = i + 1
		if i > 0 {
			near = append(near, ids[i-1])
		}
		if i < len(ids)-1 {
			near = append(near, ids[i+1])
		}
		byID, err := v.photosByID(ctx, near)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			p := byID[ids[i-1].String()]
			out.Prev = &p
		}
		if i < len(ids)-1 {
			p := byID[ids[i+1].String()]
			out.Next = &p
		}
	}
	place := ph.Plot.Place
	if place == "" {
		place = "an unnamed plot"
	}
	out.Page = v.base(fmt.Sprintf("%s by %s", ph.Plot.Name(), ph.Surveyor.Label()), "/photo/"+ph.ID,
		fmt.Sprintf("A photo near %s by %s. %s.", place, ph.Surveyor.Label(), out.Licence.Name), "")
	out.OG = views.OpenGraph{Type: "article", ImageKey: ph.Keys.Best(), ImageAlt: "Photo near " + place}
	out.OG.ImageWidth, out.OG.ImageHeight = ph.Width, ph.Height
	if ph.Keys.Medium != "" && ph.Width > 0 && ph.Height > 0 {
		out.OG.ImageWidth, out.OG.ImageHeight = scaleTo(ph.Width, ph.Height, mediumSide)
	}
	return out, nil
}

// scaleTo returns w×h fitted into side×side without upscaling, rounded as
// imaging.Fit does.
func scaleTo(w, h, side int) (int, int) {
	if w <= side && h <= side {
		return w, h
	}
	if w >= h {
		return side, max(1, int(float64(h)*float64(side)/float64(w)+0.5))
	}
	return max(1, int(float64(w)*float64(side)/float64(h)+0.5)), side
}

// region looks up a region page by slug. Districts have no counts of their
// own yet, so they have no page.
func (v *Views) region(ctx context.Context, slug string) (*regionRow, error) {
	rows, err := v.d.Pool.Query(ctx, `select `+regionCols+` from regions where slug = lower($1) and kind <> 'district'`, slug)
	if err != nil {
		return nil, fmt.Errorf("db: region: %w", err)
	}
	r, err := pgx.CollectExactlyOneRow(rows, scanRegion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: region: %w", err)
	}
	return &r, nil
}

// Region implements views.Source.
func (v *Views) Region(ctx context.Context, slug string, page int) (*views.Region, error) {
	r, err := v.region(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := &views.Region{Region: r.ref, Kind: r.kind, Coverage: r.cov, Bounds: r.b, LastSurveyedAt: r.last}
	if out.Country, err = v.regionCountry(ctx, r.id); err != nil {
		return nil, err
	}
	out.Counters.Plots = r.cov.Claimed
	err = v.d.Pool.QueryRow(ctx, `select count(*), count(distinct `+surveyorKey+`)`+photoFrom+`where pl.region_id = $1`, r.id).
		Scan(&out.Counters.Photos, &out.Counters.Surveyors)
	if err != nil {
		return nil, fmt.Errorf("db: region counters: %w", err)
	}
	pg, err := pagination(page, views.PerPage, out.Counters.Photos)
	if err != nil {
		return nil, err
	}
	out.Pagination = pg
	if out.Latest, err = v.photos(ctx, "where pl.region_id = $1", newestFirst+" limit $2 offset $3", r.id, pg.PerPage, offset(pg)); err != nil {
		return nil, err
	}
	recent, err := v.activity(ctx, r.ref.Slug, v.since(), views.RegionTop)
	if err != nil {
		return nil, err
	}
	all, err := v.activity(ctx, r.ref.Slug, time.Time{}, views.RegionTop)
	if err != nil {
		return nil, err
	}
	out.ActiveRecent, out.ActiveAllTime = recent[r.id], all[r.id]
	path := "/region/" + r.ref.Slug
	out.Page = v.base(r.ref.NameEn, withPage(path, pg.Page),
		fmt.Sprintf("Survey coverage of %s: %.2f %% of %d plots.", r.ref.NameEn, r.cov.Percent(), r.cov.Land), path+".rss")
	return out, nil
}

// RegionRefs implements views.Source.
func (v *Views) RegionRefs(ctx context.Context) ([]views.RegionRef, error) {
	rows, err := v.d.Pool.Query(ctx, `select slug, name_en, coalesce(name_th, '') from regions where kind = 'province' order by name_en`)
	if err != nil {
		return nil, fmt.Errorf("db: region refs: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (views.RegionRef, error) {
		var x views.RegionRef
		return x, r.Scan(&x.Slug, &x.NameEn, &x.NameTh)
	})
	if err != nil {
		return nil, fmt.Errorf("db: region refs: %w", err)
	}
	return out, nil
}

func (v *Views) feed(ctx context.Context, title, canonical, desc, where string, args ...any) (*views.Feed, error) {
	args = append(args, views.FeedItems)
	items, err := v.photos(ctx, where, fmt.Sprintf("%s limit $%d", newestFirst, len(args)), args...)
	if err != nil {
		return nil, err
	}
	f := &views.Feed{Page: v.base(title, canonical, desc, ""), Items: items, Updated: v.now().UTC()}
	if len(items) > 0 {
		f.Updated = items[0].UploadedAt
	}
	return f, nil
}

// LatestFeed implements views.Source.
func (v *Views) LatestFeed(ctx context.Context) (*views.Feed, error) {
	return v.feed(ctx, views.Wordmark+": latest surveys", "/latest", "The newest photos on Cartograph.", "")
}

// RegionFeed implements views.Source.
func (v *Views) RegionFeed(ctx context.Context, slug string) (*views.Feed, error) {
	r, err := v.region(ctx, slug)
	if err != nil {
		return nil, err
	}
	return v.feed(ctx, views.Wordmark+": "+r.ref.NameEn, "/region/"+r.ref.Slug, "The newest photos in "+r.ref.NameEn+".",
		"where pl.region_id = $1", r.id)
}

// SurveyorFeed implements views.Source.
func (v *Views) SurveyorFeed(ctx context.Context, name string) (*views.Feed, error) {
	s, err := v.surveyor(ctx, name)
	if err != nil {
		return nil, err
	}
	return v.feed(ctx, views.Wordmark+": "+s.profile.Label(), surveyorPath(s.profile.Name), "The newest photos by "+s.profile.Label()+".",
		"where ph.surveyor_id = $1", s.id)
}

// camera returns the camera data of a photo, nil if it has none.
func (v *Views) camera(ctx context.Context, id uuid.UUID) (*views.Camera, error) {
	var c views.Camera
	err := v.d.Pool.QueryRow(ctx, `
		select camera, coalesce(lens, ''), coalesce(focal_length_mm, 0), coalesce(focal_length_35mm, 0),
		       coalesce(f_number, 0), coalesce(exposure_time_s, 0), coalesce(iso, 0)
		from photo_camera where photo_id = $1`, id,
	).Scan(&c.Label, &c.Lens, &c.FocalMM, &c.Focal35MM, &c.FNumber, &c.ExposureS, &c.ISO)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: photo camera: %w", err)
	}
	return &c, nil
}
