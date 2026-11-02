// Command commons fetches free, geotagged photos from Wikimedia Commons for
// development data (docs/SEED_PHOTOS_BRIEF.md): up to four per place over
// Thailand, only CC0, CC BY and CC BY-SA, as JPEG files of 2000 px with a
// manifest of position, author and licence, ready for `seed -commons`.
//
//	go run ./cmd/commons -out testdata/commons
//
// The output directory is git-ignored; nothing it holds enters the repository.
// Requests go one at a time with a pause between them, and files that are
// already there are not downloaded again.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/commons"
)

const (
	api       = "https://commons.wikimedia.org/w/api.php"
	sparql    = "https://query.wikidata.org/sparql"
	userAgent = "cartograph-dev-seed/1.0 (https://github.com/cartograph-asia/cartograph; development data)"
	thumbPx   = 2000
	minWidth  = 1600
	minApart  = 400 // metres between two chosen photos, so they fall in different plots
)

// place is somewhere to search around.
type place struct {
	name     string
	lat, lng float64
}

var places = []place{
	{"Bangkok, Grand Palace", 13.7500, 100.4913},
	{"Bangkok, Chatuchak", 13.8000, 100.5500},
	{"Ayutthaya", 14.3532, 100.5776},
	{"Kanchanaburi", 14.0228, 99.5328},
	{"Hua Hin", 12.5684, 99.9577},
	{"Pattaya", 12.9346, 100.8840},
	{"Chanthaburi", 12.6100, 102.1000},
	{"Trat, Ko Chang", 12.0500, 102.3300},
	{"Nakhon Ratchasima", 14.9799, 102.0978},
	{"Ubon Ratchathani", 15.2287, 104.8564},
	{"Khon Kaen", 16.4322, 102.8236},
	{"Udon Thani", 17.4138, 102.7870},
	{"Phitsanulok", 16.8211, 100.2659},
	{"Sukhothai", 17.0178, 99.7036},
	{"Nan", 18.7800, 100.7800},
	{"Chiang Rai", 19.9105, 99.8406},
	{"Chiang Mai", 18.7883, 98.9853},
	{"Pai", 19.3590, 98.4400},
	{"Mae Hong Son", 19.3010, 97.9690},
	{"Surat Thani, Khao Sok", 8.9000, 98.5300},
	{"Ko Samui", 9.5120, 100.0136},
	{"Nakhon Si Thammarat", 8.4300, 99.9600},
	{"Phuket", 7.8840, 98.3923},
	{"Krabi", 8.0300, 98.8190},
	{"Songkhla", 7.2000, 100.5950},
}

func main() {
	out := flag.String("out", "testdata/commons", "directory for the photos, the manifest and CREDITS.md")
	per := flag.Int("per-place", 4, "photos to keep per place")
	radius := flag.Int("radius", 10000, "search radius in metres (at most 10000)")
	wikidata := flag.Bool("wikidata", false, "also take images of Wikidata items near each place (landmarks with an image)")
	pause := flag.Duration("pause", 700*time.Millisecond, "pause between requests")
	flag.Parse()
	c := &client{http: &http.Client{Timeout: 60 * time.Second}, pause: *pause}
	if err := run(c, *out, *per, min(*radius, 10000), *wikidata); err != nil {
		slog.Error("commons", "err", err)
		os.Exit(1)
	}
}

type client struct {
	http  *http.Client
	pause time.Duration
	last  time.Time
}

// get fetches u, waiting between requests and honouring Retry-After a few times.
func (c *client) get(u string) ([]byte, error) {
	for try := 0; ; try++ {
		if wait := c.pause - time.Since(c.last); wait > 0 {
			time.Sleep(wait)
		}
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := c.http.Do(req)
		c.last = time.Now()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		switch {
		case resp.StatusCode == http.StatusOK && transient(apiError(b)) && try < 6:
			// The servers are lagging or busy; the API says so with status 200.
			time.Sleep(8 * time.Second)
		case resp.StatusCode == http.StatusOK:
			if code := apiError(b); code != "" {
				return nil, fmt.Errorf("%s: API error %q", u, code)
			}
			return b, nil
		case (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && try < 3:
			secs, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			time.Sleep(time.Duration(max(secs, 10)) * time.Second)
		default:
			return nil, fmt.Errorf("%s: %s", u, resp.Status)
		}
	}
}

// apiError returns the code of an API error in body ({"error":{"code":…}}), or "".
func apiError(body []byte) string {
	var r struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &r) != nil {
		return ""
	}
	return r.Error.Code
}

// transient reports whether an API error code means "try again in a moment".
func transient(code string) bool {
	return code == "maxlag" || code == "cirrussearch-too-busy-error" || code == "ratelimited"
}

type hit struct {
	Title   string  `json:"title"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Source  string  `json:"-"` // "commons" (the file's own position) or "wikidata" (the item's)
	Subject string  `json:"-"` // the Wikidata item the image belongs to
}

type meta map[string]struct {
	Value string `json:"value"`
}

type info struct {
	PageID  int    `json:"pageid"` // of the file page; set from the page, not the image info
	Thumb   string `json:"thumburl"`
	Width   int    `json:"width"`
	Mime    string `json:"mime"`
	Page    string `json:"descriptionurl"`
	Extmeta meta   `json:"extmetadata"`
}

// nearby asks Wikidata for items within radius metres of p that have an image
// (P18), nearest first: landmarks, temples, stations, whose photos are on
// Commons. Their position is the item's, not the file's.
func (c *client) nearby(p place, radius int) ([]hit, error) {
	q := fmt.Sprintf(`SELECT ?itemLabel ?loc ?image WHERE {
  SERVICE wikibase:around { ?item wdt:P625 ?loc . bd:serviceParam wikibase:center "Point(%f %f)"^^geo:wktLiteral .
    bd:serviceParam wikibase:radius "%d" . bd:serviceParam wikibase:distance ?dist . }
  ?item wdt:P18 ?image .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en,th". }
} ORDER BY ?dist LIMIT 40`, p.lng, p.lat, radius/1000)
	b, err := c.get(sparql + "?" + url.Values{"query": {q}, "format": {"json"}}.Encode())
	if err != nil {
		return nil, err
	}
	return parseNearby(b)
}

// parseNearby reads the SPARQL result of nearby.
func parseNearby(b []byte) ([]hit, error) {
	var r struct {
		Results struct {
			Bindings []struct {
				Label struct{ Value string } `json:"itemLabel"`
				Loc   struct{ Value string } `json:"loc"`
				Image struct{ Value string } `json:"image"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	var out []hit
	for _, x := range r.Results.Bindings {
		var lat, lng float64
		if _, err := fmt.Sscanf(x.Loc.Value, "Point(%f %f)", &lng, &lat); err != nil {
			continue
		}
		_, name, ok := strings.Cut(x.Image.Value, "Special:FilePath/")
		if !ok {
			continue
		}
		name, err := url.PathUnescape(name)
		if err != nil || name == "" {
			continue
		}
		out = append(out, hit{Title: "File:" + name, Lat: lat, Lon: lng, Source: "wikidata", Subject: x.Label.Value})
	}
	return out, nil
}

func (c *client) search(p place, radius int) ([]hit, error) {
	q := url.Values{
		"action": {"query"}, "list": {"geosearch"}, "format": {"json"}, "formatversion": {"2"}, "maxlag": {"5"},
		"gscoord": {fmt.Sprintf("%f|%f", p.lat, p.lng)}, "gsradius": {strconv.Itoa(radius)},
		"gsnamespace": {"6"}, "gslimit": {"50"},
	}
	b, err := c.get(api + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	var r struct {
		Query struct {
			Geosearch []hit `json:"geosearch"`
		} `json:"query"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	for i := range r.Query.Geosearch {
		r.Query.Geosearch[i].Source = "commons"
	}
	return r.Query.Geosearch, nil
}

// maxQuery is how long the titles of one request may be once encoded; Thai
// titles are long and a URL over a few thousand bytes is refused.
const maxQuery = 3000

// infos returns the image info of titles, by title, asking in batches.
func (c *client) infos(titles []string) (map[string]info, error) {
	out := map[string]info{}
	for len(titles) > 0 {
		n, size := 0, 0
		for n < len(titles) && (n == 0 || size+len(url.QueryEscape(titles[n]))+3 <= maxQuery) {
			size += len(url.QueryEscape(titles[n])) + 3
			n++
		}
		got, err := c.infoBatch(titles[:n])
		if err != nil {
			return nil, err
		}
		for k, v := range got {
			out[k] = v
		}
		titles = titles[n:]
	}
	return out, nil
}

func (c *client) infoBatch(titles []string) (map[string]info, error) {
	q := url.Values{
		"action": {"query"}, "prop": {"imageinfo"}, "format": {"json"}, "formatversion": {"2"}, "maxlag": {"5"},
		"titles": {strings.Join(titles, "|")}, "iiprop": {"url|size|mime|extmetadata"}, "iiurlwidth": {strconv.Itoa(thumbPx)},
		"iiextmetadatafilter": {"LicenseShortName|LicenseUrl|Artist|DateTimeOriginal"},
	}
	b, err := c.get(api + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	var r struct {
		Query struct {
			Pages []struct {
				PageID    int    `json:"pageid"`
				Title     string `json:"title"`
				ImageInfo []info `json:"imageinfo"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	out := map[string]info{}
	for _, p := range r.Query.Pages {
		if len(p.ImageInfo) > 0 {
			in := p.ImageInfo[0]
			in.PageID = p.PageID
			out[p.Title] = in
		}
	}
	return out, nil
}

var (
	tags  = regexp.MustCompile(`<[^>]*>`)
	space = regexp.MustCompile(`\s+`)
)

// plain turns Commons' HTML credit into one line of text.
func plain(s string) string {
	s = html.UnescapeString(tags.ReplaceAllString(s, " "))
	s = strings.TrimSpace(space.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > 120 {
		s = string(r[:120]) + "…"
	}
	return s
}

// bangkok is where the photos were taken, for dates without an offset.
var bangkok = time.FixedZone("ICT", 7*3600)

// taken parses Commons' DateTimeOriginal ("2019-03-04 10:22:11"); nil if absent,
// unparseable, in the future or older than ten years (docs/GAMIFICATION.md).
func taken(s string, now time.Time) *time.Time {
	s = strings.TrimSpace(plain(s))
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, bangkok); err == nil {
			if t.After(now) || t.Before(now.AddDate(-10, 0, 0)) {
				return nil
			}
			return &t
		}
	}
	return nil
}

// metres is the distance between two points.
func metres(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000
	rad := math.Pi / 180
	dLat, dLng := (lat2-lat1)*rad, (lng2-lng1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

func run(c *client, out string, per, radius int, wikidata bool) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	now := time.Now()
	// Earlier runs count: a place that has its photos is not searched again, so
	// running again only fills the gaps.
	m := &commons.Manifest{}
	if old, err := commons.Read(out); err == nil {
		m = old
	}
	have := map[string]int{}
	for _, ph := range m.Photos {
		have[ph.Place]++
	}
	for _, p := range places {
		if have[p.name] >= per {
			continue
		}
		hits, err := c.search(p, radius)
		if err != nil { // one place failing leaves the others; running again fills it in
			slog.Warn("search failed, skipped", "place", p.name, "err", short(err))
			continue
		}
		if wikidata {
			if wd, err := c.nearby(p, radius); err != nil {
				slog.Warn("wikidata failed", "place", p.name, "err", short(err))
			} else {
				hits = mergeHits(hits, wd)
			}
		}
		titles := make([]string, len(hits))
		for i, h := range hits {
			titles[i] = h.Title
		}
		got := map[string]info{}
		if len(titles) > 0 {
			if got, err = c.infos(titles); err != nil {
				slog.Warn("info failed, skipped", "place", p.name, "err", short(err))
				continue
			}
		}
		kept := have[p.name]
		for _, h := range hits { // nearest first
			if kept == per {
				break
			}
			in, ok := got[h.Title]
			ph := commons.Photo{
				File: fmt.Sprintf("p%d.jpg", in.PageID), Title: h.Title, Place: p.name, Lat: h.Lat, Lng: h.Lon,
				Source: h.Source, Subject: h.Subject,
				PageURL: in.Page, Author: plain(in.Extmeta["Artist"].Value), Licence: plain(in.Extmeta["LicenseShortName"].Value),
				LicenceURL: plain(in.Extmeta["LicenseUrl"].Value), Taken: taken(in.Extmeta["DateTimeOriginal"].Value, now),
			}
			if ph.Author == "" {
				ph.Author = "Unknown"
			}
			switch {
			case !ok || in.Mime != "image/jpeg" || in.Width < minWidth || in.Thumb == "":
				continue
			case ph.Validate() != nil:
				continue
			case tooClose(m.Photos, ph), has(m.Photos, ph.File):
				continue
			}
			if err := c.download(in.Thumb, filepath.Join(out, ph.File)); err != nil {
				slog.Warn("download", "title", h.Title, "err", err)
				continue
			}
			m.Photos = append(m.Photos, ph)
			kept++
		}
		slog.Info("place", "place", p.name, "found", len(hits), "kept", kept)
	}
	if len(m.Photos) == 0 {
		return errors.New("no photo matched")
	}
	if err := commons.Write(out, m); err != nil {
		return err
	}
	if err := writeCredits(out, m); err != nil {
		return err
	}
	slog.Info("done", "photos", len(m.Photos), "dir", out)
	return nil
}

// short is err cut to a line the log can show (the URLs in them are long).
func short(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 && len(s) > 160 {
		s = s[i+2:]
	}
	return s
}

// mergeHits puts the Wikidata hits after the Commons ones, without a title twice.
func mergeHits(commonsHits, wikidataHits []hit) []hit {
	seen := map[string]bool{}
	for _, h := range commonsHits {
		seen[h.Title] = true
	}
	for _, h := range wikidataHits {
		if !seen[h.Title] {
			seen[h.Title] = true
			commonsHits = append(commonsHits, h)
		}
	}
	return commonsHits
}

func subject(p commons.Photo) string {
	if p.Subject == "" {
		return ""
	}
	return " (" + p.Subject + ")"
}

func has(ps []commons.Photo, file string) bool {
	for _, p := range ps {
		if p.File == file {
			return true
		}
	}
	return false
}

func tooClose(have []commons.Photo, p commons.Photo) bool {
	for _, h := range have {
		if metres(h.Lat, h.Lng, p.Lat, p.Lng) < minApart {
			return true
		}
	}
	return false
}

// download saves url to path unless a file is already there.
func (c *client) download(u, path string) error {
	if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
		return nil
	}
	b, err := c.get(u)
	if err != nil {
		return err
	}
	if len(b) < 3 || b[0] != 0xFF || b[1] != 0xD8 {
		return errors.New("not a JPEG")
	}
	return os.WriteFile(path, b, 0o644)
}

func writeCredits(dir string, m *commons.Manifest) error {
	var b strings.Builder
	b.WriteString("# Credits\n\nPhotos from Wikimedia Commons for development data (docs/SEED_PHOTOS_BRIEF.md).\nNot for the repository, not for production, not for the public site.\n\n")
	for _, p := range m.Photos {
		fmt.Fprintf(&b, "- %s: %s, %s, %s%s\n", p.File, p.Author, p.Licence, p.PageURL, subject(p))
	}
	return os.WriteFile(filepath.Join(dir, "CREDITS.md"), []byte(b.String()), 0o644)
}
