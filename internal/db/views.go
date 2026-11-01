package db

import (
	"context"
	"errors"
	"fmt"
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
	coalesce(pn.number, 0)`

const photoFrom = ` from photos ph
	join plots pl on pl.cell = ph.cell
	left join plot_numbers pn on pn.cell = ph.cell
	left join regions r on r.id = pl.region_id
	left join surveyors s on s.id = ph.surveyor_id `

// surveyorKey groups photos by account, else by the free-text name.
const surveyorKey = `coalesce(s.name, ph.surveyor)`

func scanPhoto(row pgx.CollectableRow) (views.PhotoItem, error) {
	var p views.PhotoItem
	var id uuid.UUID
	err := row.Scan(&id, &p.Keys.Thumb, &p.Keys.Medium, &p.Keys.Large, &p.Width, &p.Height, &p.Plot.Cell,
		&p.Plot.Place, &p.Plot.PlaceTh, &p.Plot.Region.Slug, &p.Plot.Region.NameEn, &p.Plot.Region.NameTh,
		&p.Surveyor.Name, &p.TakenAt, &p.UploadedAt, &p.FirstSurvey, &p.Plot.Number)
	p.ID = id.String()
	p.UploadedAt = p.UploadedAt.UTC()
	if p.TakenAt != nil {
		t := p.TakenAt.UTC()
		p.TakenAt = &t
	}
	return p, err
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
	h := &views.Home{Page: v.base("", "/", "A photo for every plot of land in Asia, surveyed by its people.", "/latest.rss")}
	err := v.d.Pool.QueryRow(ctx, `select
		(select count(*) from plots where photo_count > 0),
		(select count(*) from photos),
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
	if h.Regions, err = v.regionSummaries(ctx, true); err != nil {
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
		select s.name, s.joined_at, f.id from surveyors s
		join lateral (select id from photos where surveyor_id = s.id order by uploaded_at, id limit 1) f on true
		order by s.joined_at desc, s.id limit $1`, views.HomeNewSurveyors)
	if err != nil {
		return nil, fmt.Errorf("db: new surveyors: %w", err)
	}
	type row struct {
		name   string
		joined time.Time
		photo  uuid.UUID
	}
	rs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		return x, r.Scan(&x.name, &x.joined, &x.photo)
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
		out = append(out, views.NewSurveyor{Surveyor: views.SurveyorRef{Name: r.name}, JoinedAt: r.joined.UTC(), FirstPhoto: byID[r.photo.String()]})
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
	cov  views.Coverage
	last *time.Time
	b    views.Bounds
}

const regionCols = `id, slug, name_en, coalesce(name_th, ''), claimed_cells, land_cells, last_surveyed_at,
	min_lat, min_lng, max_lat, max_lng`

func scanRegion(r pgx.CollectableRow) (regionRow, error) {
	var x regionRow
	err := r.Scan(&x.id, &x.ref.Slug, &x.ref.NameEn, &x.ref.NameTh, &x.cov.Claimed, &x.cov.Land, &x.last,
		&x.b.MinLat, &x.b.MinLng, &x.b.MaxLat, &x.b.MaxLng)
	if x.last != nil {
		t := x.last.UTC()
		x.last = &t
	}
	return x, err
}

// regionSummaries returns provinces (only those with a surveyed plot if
// activeOnly) by most recent activity, then name, with their top surveyors.
func (v *Views) regionSummaries(ctx context.Context, activeOnly bool) ([]views.RegionSummary, error) {
	where := "where kind = 'province'"
	if activeOnly {
		where += " and claimed_cells > 0"
	}
	rows, err := v.d.Pool.Query(ctx, `select `+regionCols+` from regions `+where+`
		order by last_surveyed_at desc nulls last, name_en`)
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
		select region_id, name, photos, plots from (
		  select pl.region_id, `+surveyorKey+` as name, count(*) as photos,
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
		if err := rows.Scan(&id, &a.Surveyor.Name, &a.Photos, &a.Plots); err != nil {
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
	total, err := v.count(ctx, `select count(*) from photos`)
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
		select s.name, s.joined_at,
		  (select count(*) from plots where first_surveyor_id = s.id),
		  (select count(*) from photos where surveyor_id = s.id),
		  (select id from photos where surveyor_id = s.id order by uploaded_at desc, id desc limit 1)
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
		err := r.Scan(&x.item.Surveyor.Name, &x.item.JoinedAt, &x.item.Plots, &x.item.Photos, &x.last)
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
	rs, err := v.regionSummaries(ctx, false)
	if err != nil {
		return nil, err
	}
	return &views.RegionList{Page: v.base("Regions", "/regions", "Survey coverage by province.", ""), Regions: rs}, nil
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
		select s.id, s.name, s.joined_at, s.bio, r.slug, r.name_en, r.name_th
		from surveyors s left join regions r on r.id = s.home_region_id
		where lower(s.name) = lower($1)`, name,
	).Scan(&s.id, &s.profile.Name, &s.profile.JoinedAt, &s.profile.Bio, &slug, &en, &th)
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

// Surveyor implements views.Source.
func (v *Views) Surveyor(ctx context.Context, name string, page int) (*views.Surveyor, error) {
	s, err := v.surveyor(ctx, name)
	if err != nil {
		return nil, err
	}
	out := &views.Surveyor{Profile: s.profile}
	var minLat, minLng, maxLat, maxLng *float64
	err = v.d.Pool.QueryRow(ctx, `
		select
		  (select count(*) from plots where first_surveyor_id = $1),
		  count(*), count(distinct pl.region_id),
		  min(ph.lat), min(ph.lng), max(ph.lat), max(ph.lng)
		from photos ph join plots pl on pl.cell = ph.cell where ph.surveyor_id = $1`, s.id,
	).Scan(&out.Stats.Plots, &out.Stats.Photos, &out.Stats.Regions, &minLat, &minLng, &maxLat, &maxLng)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor stats: %w", err)
	}
	if minLat != nil {
		out.Bounds = &views.Bounds{MinLat: *minLat, MinLng: *minLng, MaxLat: *maxLat, MaxLng: *maxLng}
	}
	pg, err := pagination(page, views.PerPage, out.Stats.Photos)
	if err != nil {
		return nil, err
	}
	out.Pagination = pg
	if out.Photos, err = v.photos(ctx, "where ph.surveyor_id = $1", newestFirst+" limit $2 offset $3", s.id, pg.PerPage, offset(pg)); err != nil {
		return nil, err
	}
	rows, err := v.d.Pool.Query(ctx, `select distinct cell from photos where surveyor_id = $1 order by cell`, s.id)
	if err != nil {
		return nil, fmt.Errorf("db: surveyor cells: %w", err)
	}
	if out.PlotCells, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
		return nil, fmt.Errorf("db: surveyor cells: %w", err)
	}
	desc := s.profile.Bio
	if desc == "" {
		desc = fmt.Sprintf("Photos by %s on Cartograph.", s.profile.Name)
	}
	path := surveyorPath(s.profile.Name)
	out.Page = v.base(s.profile.Name, withPage(path, pg.Page), desc, path+".rss")
	return out, nil
}

// Plot implements views.Source.
func (v *Views) Plot(ctx context.Context, cell string) (*views.Plot, error) {
	cell, res, err := h3.Parse(cell)
	if err != nil || res != h3.PlotRes {
		return nil, views.ErrNotFound
	}
	out := &views.Plot{Plot: views.PlotRef{Cell: cell}}
	var place, placeTh, firstName, slug, en, th *string
	var claimedAt *time.Time
	err = v.d.Pool.QueryRow(ctx, `
		select pl.photo_count, pl.claimed_at, pl.place_name, pl.place_name_th,
		       coalesce(s.name, pl.first_surveyor), r.slug, r.name_en, r.name_th
		from plots pl
		left join surveyors s on s.id = pl.first_surveyor_id
		left join regions r on r.id = pl.region_id
		where pl.cell = $1 and pl.photo_count > 0`, cell,
	).Scan(&out.PhotoCount, &claimedAt, &place, &placeTh, &firstName, &slug, &en, &th)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("db: plot page: %w", err)
	default:
		out.Surveyed = true
		t := claimedAt.UTC()
		out.ClaimedAt = &t
		if firstName != nil {
			out.FirstSurveyor = &views.SurveyorRef{Name: *firstName}
		}
		if slug != nil {
			out.Plot.Region = views.RegionRef{Slug: *slug, NameEn: *en, NameTh: deref(th)}
		}
		out.Plot.Place, out.Plot.PlaceTh = deref(place), deref(placeTh)
	}
	if out.Plot.Place == "" || out.Plot.Region.Slug == "" {
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
	rows, err := v.d.Pool.Query(ctx, `select cell from plots where cell = any($1) and photo_count > 0`, ring)
	if err != nil {
		return nil, fmt.Errorf("db: plot neighbours: %w", err)
	}
	surveyed, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("db: plot neighbours: %w", err)
	}
	set := map[string]bool{}
	for _, c := range surveyed {
		set[c] = true
	}
	for _, c := range ring {
		out.Neighbours = append(out.Neighbours, views.CellStatus{Cell: c, Surveyed: set[c]})
	}

	name := out.Plot.Name()
	out.Page = v.base(name, "/plot/"+cell, "", "")
	if out.Surveyed {
		who := ""
		if out.FirstSurveyor != nil {
			who = " by " + out.FirstSurveyor.Name
		}
		out.Description = fmt.Sprintf("%s, first surveyed%s. %s.", name, who, plural(out.PhotoCount, "photo"))
	} else {
		out.NoIndex = true
		out.Description = fmt.Sprintf("%s: nobody has surveyed this plot yet.", name)
	}
	return out, nil
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
		return nil, views.ErrNotFound
	}
	ph := ps[0]
	var licence string
	if err := v.d.Pool.QueryRow(ctx, `select licence from photos where id = $1`, uid).Scan(&licence); err != nil {
		return nil, fmt.Errorf("db: photo licence: %w", err)
	}
	out := &views.Photo{Photo: ph, Licence: views.Licence{Name: licence}}
	if licence == views.CCBYSA4.Name {
		out.Licence = views.CCBYSA4
	}
	rows, err := v.d.Pool.Query(ctx, `select id from photos where cell = $1 order by uploaded_at, id`, ph.Plot.Cell)
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
	out.Page = v.base(fmt.Sprintf("%s by %s", ph.Plot.Name(), ph.Surveyor.Name), "/photo/"+ph.ID,
		fmt.Sprintf("A photo near %s by %s. %s.", place, ph.Surveyor.Name, out.Licence.Name), "")
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

func (v *Views) region(ctx context.Context, slug string) (*regionRow, error) {
	rows, err := v.d.Pool.Query(ctx, `select `+regionCols+` from regions where slug = lower($1)`, slug)
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
	out := &views.Region{Region: r.ref, Coverage: r.cov, Bounds: r.b, LastSurveyedAt: r.last}
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
	return v.feed(ctx, "Cartograph: latest surveys", "/latest", "The newest photos on Cartograph.", "")
}

// RegionFeed implements views.Source.
func (v *Views) RegionFeed(ctx context.Context, slug string) (*views.Feed, error) {
	r, err := v.region(ctx, slug)
	if err != nil {
		return nil, err
	}
	return v.feed(ctx, "Cartograph: "+r.ref.NameEn, "/region/"+r.ref.Slug, "The newest photos in "+r.ref.NameEn+".",
		"where pl.region_id = $1", r.id)
}

// SurveyorFeed implements views.Source.
func (v *Views) SurveyorFeed(ctx context.Context, name string) (*views.Feed, error) {
	s, err := v.surveyor(ctx, name)
	if err != nil {
		return nil, err
	}
	return v.feed(ctx, "Cartograph: "+s.profile.Name, surveyorPath(s.profile.Name), "The newest photos by "+s.profile.Name+".",
		"where ph.surveyor_id = $1", s.id)
}
