package viewstest

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Fake is an in-memory views.Source over the fixture data. It follows the
// Source conventions (pagination, case-insensitive names, ErrNotFound) so
// handler tests behave as against the database.
type Fake struct {
	PerPage int // photos per page; views.PerPage if 0 (set lower to exercise pagination)
	d       *data
}

var _ views.Source = (*Fake)(nil)

// NewFake returns a Fake over the fixture data.
func NewFake() *Fake { return &Fake{d: build()} }

func (f *Fake) perPage() int {
	if f.PerPage > 0 {
		return f.PerPage
	}
	return views.PerPage
}

func base(title, canonical, desc, feed string) views.Page {
	return views.Page{Now: Now, Title: title, Canonical: canonical, Description: desc, Feed: feed}
}

// newestFirst returns the photos at idx, newest first.
func (f *Fake) newestFirst(idx []int) []views.PhotoItem {
	out := make([]views.PhotoItem, 0, len(idx))
	for i := len(idx) - 1; i >= 0; i-- {
		out = append(out, f.d.photos[idx[i]])
	}
	return out
}

func (f *Fake) all() []int {
	idx := make([]int, len(f.d.photos))
	for i := range idx {
		idx[i] = i
	}
	return idx
}

// paginate cuts one page out of items.
func paginate[T any](items []T, page, per int) ([]T, views.Pagination, error) {
	if page < 1 {
		page = 1
	}
	pg := views.Pagination{Page: page, PerPage: per, Total: len(items)}
	if page > pg.Pages() {
		return nil, pg, views.ErrNotFound
	}
	lo := min((page-1)*per, len(items))
	hi := min(lo+per, len(items))
	return items[lo:hi], pg, nil
}

func withPage(path string, page int) string {
	if page <= 1 {
		return path
	}
	return fmt.Sprintf("%s?page=%d", path, page)
}

// activity counts photos per surveyor among idx since since (zero = all time).
func (f *Fake) activity(idx []int, since time.Time, limit int) []views.SurveyorActivity {
	counts := map[string]*views.SurveyorActivity{}
	for _, i := range idx {
		ph := f.d.photos[i]
		if ph.UploadedAt.Before(since) {
			continue
		}
		a := counts[ph.Surveyor.Name]
		if a == nil {
			a = &views.SurveyorActivity{Surveyor: ph.Surveyor}
			counts[ph.Surveyor.Name] = a
		}
		a.Photos++
		if ph.FirstSurvey {
			a.Plots++
		}
	}
	out := make([]views.SurveyorActivity, 0, len(counts))
	for _, a := range counts {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Photos != out[j].Photos {
			return out[i].Photos > out[j].Photos
		}
		return out[i].Surveyor.Name < out[j].Surveyor.Name
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (f *Fake) regionPhotos(slug string) []int {
	var idx []int
	for i, ph := range f.d.photos {
		if ph.Plot.Region.Slug == slug {
			idx = append(idx, i)
		}
	}
	return idx
}

func (f *Fake) regionSummary(r region) views.RegionSummary {
	idx := f.regionPhotos(r.ref.Slug)
	s := views.RegionSummary{
		Region:       r.ref,
		Coverage:     views.Coverage{Land: r.land},
		TopSurveyors: f.activity(idx, Now.Add(-views.ActiveDays*24*time.Hour), views.TopSurveyors),
	}
	for _, i := range idx {
		if f.d.photos[i].FirstSurvey {
			s.Coverage.Claimed++
		}
	}
	if len(idx) > 0 {
		t := f.d.photos[idx[len(idx)-1]].UploadedAt
		s.LastSurveyedAt = &t
	}
	return s
}

// regionSummaries returns every region, most recent activity first, then by name.
func (f *Fake) regionSummaries(activeOnly bool) []views.RegionSummary {
	var out []views.RegionSummary
	for _, r := range regions {
		s := f.regionSummary(r)
		if activeOnly && s.LastSurveyedAt == nil {
			continue
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].LastSurveyedAt, out[j].LastSurveyedAt
		switch {
		case a != nil && b != nil && !a.Equal(*b):
			return a.After(*b)
		case (a == nil) != (b == nil):
			return a != nil
		}
		return out[i].Region.NameEn < out[j].Region.NameEn
	})
	return out
}

func (f *Fake) counters() views.Counters {
	c := views.Counters{Photos: len(f.d.photos), Surveyors: len(surveyors)}
	for _, ph := range f.d.photos {
		if ph.FirstSurvey {
			c.Plots++
		}
	}
	c.Regions = len(f.regionSummaries(true))
	return c
}

// Home implements views.Source.
func (f *Fake) Home(ctx context.Context) (*views.Home, error) {
	h := &views.Home{
		Page:     base("", "/", "A photo for every plot of land in Asia, surveyed by its people.", "/latest.rss"),
		Counters: f.counters(),
		Regions:  f.regionSummaries(true),
	}
	h.Latest = f.newestFirst(f.all())
	if len(h.Latest) > views.HomeLatest {
		h.Latest = h.Latest[:views.HomeLatest]
	}
	h.Featured = &views.Featured{Photo: f.d.photos[f.d.byID[FeaturedPhotoID]], Text: FeaturedText, Editorial: true}
	// Most recently joined surveyors with at least one photo.
	order := make([]int, len(surveyors))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return surveyors[order[i]].joined.After(surveyors[order[j]].joined) })
	for _, si := range order {
		idx := f.d.bySurveyor[si]
		if len(idx) == 0 || len(h.NewSurveyors) == views.HomeNewSurveyors {
			continue
		}
		h.NewSurveyors = append(h.NewSurveyors, views.NewSurveyor{
			Surveyor: views.SurveyorRef{Name: surveyors[si].name}, JoinedAt: surveyors[si].joined,
			FirstPhoto: f.d.photos[idx[0]],
		})
	}
	return h, nil
}

// Latest implements views.Source.
func (f *Fake) Latest(ctx context.Context, page int) (*views.Latest, error) {
	items, pg, err := paginate(f.newestFirst(f.all()), page, f.perPage())
	if err != nil {
		return nil, err
	}
	return &views.Latest{
		Page:   base("Latest surveys", withPage("/latest", pg.Page), "The newest photos on Cartograph.", "/latest.rss"),
		Photos: items, Pagination: pg,
	}, nil
}

// Surveyors implements views.Source.
func (f *Fake) Surveyors(ctx context.Context, page int) (*views.SurveyorList, error) {
	var all []views.SurveyorListItem
	for si, s := range surveyors {
		it := views.SurveyorListItem{Surveyor: views.SurveyorRef{Name: s.name}, JoinedAt: s.joined}
		for _, i := range f.d.bySurveyor[si] {
			it.Photos++
			if f.d.photos[i].FirstSurvey {
				it.Plots++
			}
		}
		if idx := f.d.bySurveyor[si]; len(idx) > 0 {
			last := f.d.photos[idx[len(idx)-1]]
			it.LastPhoto = &last
		}
		all = append(all, it)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].JoinedAt.After(all[j].JoinedAt) })
	items, pg, err := paginate(all, page, views.SurveyorsPerPage)
	if err != nil {
		return nil, err
	}
	return &views.SurveyorList{
		Page:      base("Surveyors", withPage("/surveyors", pg.Page), "Everyone who surveys for Cartograph.", ""),
		Surveyors: items, Pagination: pg,
	}, nil
}

// Regions implements views.Source.
func (f *Fake) Regions(ctx context.Context) (*views.RegionList, error) {
	return &views.RegionList{
		Page:    base("Regions", "/regions", "Survey coverage by province.", ""),
		Regions: f.regionSummaries(false),
	}, nil
}

func findSurveyor(name string) (int, bool) {
	for i, s := range surveyors {
		if strings.EqualFold(s.name, name) {
			return i, true
		}
	}
	return 0, false
}

// Surveyor implements views.Source.
func (f *Fake) Surveyor(ctx context.Context, name string, page int) (*views.Surveyor, error) {
	si, ok := findSurveyor(name)
	if !ok {
		return nil, views.ErrNotFound
	}
	s := surveyors[si]
	idx := f.d.bySurveyor[si]
	items, pg, err := paginate(f.newestFirst(idx), page, f.perPage())
	if err != nil {
		return nil, err
	}
	path := "/surveyor/" + url.PathEscape(s.name)
	out := &views.Surveyor{
		Page:       base(s.name, withPage(path, pg.Page), s.bio, path+".rss"),
		Profile:    views.SurveyorProfile{Name: s.name, JoinedAt: s.joined, Bio: s.bio, HomeRegion: s.home},
		Photos:     items,
		Pagination: pg,
	}
	if out.Description == "" {
		out.Description = fmt.Sprintf("Photos by %s on Cartograph.", s.name)
	}
	seenCell, seenRegion := map[string]bool{}, map[string]bool{}
	for _, i := range idx {
		ph := f.d.photos[i]
		out.Stats.Photos++
		if ph.FirstSurvey {
			out.Stats.Plots++
		}
		if !seenRegion[ph.Plot.Region.Slug] {
			seenRegion[ph.Plot.Region.Slug] = true
			out.Stats.Regions++
		}
		if !seenCell[ph.Plot.Cell] {
			seenCell[ph.Plot.Cell] = true
			out.PlotCells = append(out.PlotCells, ph.Plot.Cell)
		}
	}
	for _, c := range out.PlotCells {
		pl := plots[plotIndex(c)]
		if out.Bounds == nil {
			out.Bounds = &views.Bounds{MinLat: pl.lat, MinLng: pl.lng, MaxLat: pl.lat, MaxLng: pl.lng}
		}
		b := out.Bounds
		b.MinLat, b.MinLng = min(b.MinLat, pl.lat), min(b.MinLng, pl.lng)
		b.MaxLat, b.MaxLng = max(b.MaxLat, pl.lat), max(b.MaxLng, pl.lng)
	}
	return out, nil
}

func plotIndex(cell string) int {
	for i, pl := range plots {
		if pl.ref.Cell == cell {
			return i
		}
	}
	return -1
}

// Plot implements views.Source. Only fixture cells exist; 88658bd157fffff is
// blank. Neighbours are known only for 88658bd11dfffff (Pattaya Beach).
func (f *Fake) Plot(ctx context.Context, cell string) (*views.Plot, error) {
	pi := plotIndex(strings.ToLower(cell))
	if pi < 0 {
		return nil, views.ErrNotFound
	}
	pl := plots[pi]
	idx := f.d.byPlot[pi]
	title := pl.ref.Place + ", " + pl.ref.Region.NameEn
	out := &views.Plot{
		Page:       base(title, "/plot/"+pl.ref.Cell, "", ""),
		Plot:       pl.ref,
		Surveyed:   len(idx) > 0,
		PhotoCount: len(idx),
		Lat:        pl.lat, Lng: pl.lng,
	}
	for _, i := range idx {
		out.Photos = append(out.Photos, f.d.photos[i])
	}
	if out.Surveyed {
		first := out.Photos[0]
		out.FirstSurveyor = &first.Surveyor
		out.ClaimedAt = &first.UploadedAt
		out.Description = fmt.Sprintf("A plot near %s, first surveyed by %s. %d photos.", pl.ref.Place, first.Surveyor.Name, len(idx))
	} else {
		out.NoIndex = true
		out.Description = fmt.Sprintf("A plot near %s that nobody has surveyed yet.", pl.ref.Place)
	}
	if pi == 0 {
		for _, n := range plots[1:7] {
			out.Neighbours = append(out.Neighbours, views.CellStatus{Cell: n.ref.Cell, Surveyed: len(f.d.byPlot[plotIndex(n.ref.Cell)]) > 0})
		}
	}
	return out, nil
}

// Photo implements views.Source.
func (f *Fake) Photo(ctx context.Context, id string) (*views.Photo, error) {
	i, ok := f.d.byID[strings.ToLower(id)]
	if !ok {
		return nil, views.ErrNotFound
	}
	ph := f.d.photos[i]
	siblings := f.d.byPlot[plotIndex(ph.Plot.Cell)]
	out := &views.Photo{
		Page: base(fmt.Sprintf("%s, %s by %s", ph.Plot.Place, ph.Plot.Region.NameEn, ph.Surveyor.Name),
			"/photo/"+ph.ID, fmt.Sprintf("A photo near %s, %s, by %s. %s.", ph.Plot.Place, ph.Plot.Region.NameEn, ph.Surveyor.Name, views.CCBYSA4.Name), ""),
		Photo:   ph,
		Licence: views.CCBYSA4,
		Count:   len(siblings),
	}
	for pos, si := range siblings {
		if si != i {
			continue
		}
		out.Position = pos + 1
		if pos > 0 {
			prev := f.d.photos[siblings[pos-1]]
			out.Prev = &prev
		}
		if pos < len(siblings)-1 {
			next := f.d.photos[siblings[pos+1]]
			out.Next = &next
		}
	}
	out.OG = views.OpenGraph{Type: "article", ImageKey: ph.Keys.Best(), ImageAlt: "Photo near " + ph.Plot.Place}
	if ph.Width > 0 && ph.Keys.Medium != "" {
		// medium is 1200 px on the longest side
		if ph.Width >= ph.Height {
			out.OG.ImageWidth, out.OG.ImageHeight = 1200, ph.Height*1200/ph.Width
		} else {
			out.OG.ImageWidth, out.OG.ImageHeight = ph.Width*1200/ph.Height, 1200
		}
	}
	return out, nil
}

func findRegion(slug string) (region, bool) {
	for _, r := range regions {
		if r.ref.Slug == strings.ToLower(slug) {
			return r, true
		}
	}
	return region{}, false
}

// Region implements views.Source.
func (f *Fake) Region(ctx context.Context, slug string, page int) (*views.Region, error) {
	r, ok := findRegion(slug)
	if !ok {
		return nil, views.ErrNotFound
	}
	idx := f.regionPhotos(r.ref.Slug)
	items, pg, err := paginate(f.newestFirst(idx), page, f.perPage())
	if err != nil {
		return nil, err
	}
	sum := f.regionSummary(r)
	path := "/region/" + r.ref.Slug
	out := &views.Region{
		Page: base(r.ref.NameEn, withPage(path, pg.Page),
			fmt.Sprintf("Survey coverage of %s: %.2f %% of %d plots.", r.ref.NameEn, sum.Coverage.Percent(), r.land), path+".rss"),
		Region:         r.ref,
		Coverage:       sum.Coverage,
		Bounds:         r.bounds,
		Latest:         items,
		Pagination:     pg,
		ActiveRecent:   f.activity(idx, Now.Add(-views.ActiveDays*24*time.Hour), views.RegionTop),
		ActiveAllTime:  f.activity(idx, time.Time{}, views.RegionTop),
		LastSurveyedAt: sum.LastSurveyedAt,
	}
	people := map[string]bool{}
	for _, i := range idx {
		out.Counters.Photos++
		if f.d.photos[i].FirstSurvey {
			out.Counters.Plots++
		}
		people[f.d.photos[i].Surveyor.Name] = true
	}
	out.Counters.Surveyors = len(people)
	return out, nil
}

// RegionRefs implements views.Source.
func (f *Fake) RegionRefs(ctx context.Context) ([]views.RegionRef, error) {
	out := make([]views.RegionRef, 0, len(regions))
	for _, r := range regions {
		out = append(out, r.ref)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NameEn < out[j].NameEn })
	return out, nil
}

func (f *Fake) feed(title, canonical, desc string, idx []int) *views.Feed {
	items := f.newestFirst(idx)
	if len(items) > views.FeedItems {
		items = items[:views.FeedItems]
	}
	fd := &views.Feed{Page: base(title, canonical, desc, ""), Items: items, Updated: Now}
	if len(items) > 0 {
		fd.Updated = items[0].UploadedAt
	}
	return fd
}

// LatestFeed implements views.Source.
func (f *Fake) LatestFeed(ctx context.Context) (*views.Feed, error) {
	return f.feed("Cartograph: latest surveys", "/latest", "The newest photos on Cartograph.", f.all()), nil
}

// RegionFeed implements views.Source.
func (f *Fake) RegionFeed(ctx context.Context, slug string) (*views.Feed, error) {
	r, ok := findRegion(slug)
	if !ok {
		return nil, views.ErrNotFound
	}
	return f.feed("Cartograph: "+r.ref.NameEn, "/region/"+r.ref.Slug, "The newest photos in "+r.ref.NameEn+".", f.regionPhotos(r.ref.Slug)), nil
}

// SurveyorFeed implements views.Source.
func (f *Fake) SurveyorFeed(ctx context.Context, name string) (*views.Feed, error) {
	si, ok := findSurveyor(name)
	if !ok {
		return nil, views.ErrNotFound
	}
	s := surveyors[si]
	return f.feed("Cartograph: "+s.name, "/surveyor/"+url.PathEscape(s.name), "The newest photos by "+s.name+".", f.d.bySurveyor[si]), nil
}
