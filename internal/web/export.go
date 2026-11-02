package web

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// A Surveyor's plots as GPX 1.1 and GeoJSON. Both list the centre of each plot
// they have a photo in, with its name and link; they carry no photo, no time
// and no position taken from a photo's EXIF.

type gpxFile struct {
	XMLName xml.Name `xml:"gpx"`
	Version string   `xml:"version,attr"`
	Creator string   `xml:"creator,attr"`
	NS      string   `xml:"xmlns,attr"`
	Meta    gpxMeta  `xml:"metadata"`
	Points  []gpxPt  `xml:"wpt"`
}

type gpxMeta struct {
	Name string  `xml:"name"`
	Link gpxLink `xml:"link"`
}

type gpxPt struct {
	Lat  string  `xml:"lat,attr"`
	Lon  string  `xml:"lon,attr"`
	Name string  `xml:"name"`
	Desc string  `xml:"desc"`
	Link gpxLink `xml:"link"`
}

type gpxLink struct {
	Href string `xml:"href,attr"`
}

func coord(f float64) string { return strconv.FormatFloat(f, 'f', 6, 64) }

func photosOf(l *views.SurveyorPlotList, p views.SurveyorPlot) string {
	n := strconv.Itoa(p.Photos)
	if p.Photos == 1 {
		return n + " photo by " + l.Surveyor.Label()
	}
	return n + " photos by " + l.Surveyor.Label()
}

func (rd *Renderer) writeGPX(w io.Writer, l *views.SurveyorPlotList) error {
	g := gpxFile{
		Version: "1.1", Creator: views.Wordmark, NS: "http://www.topografix.com/GPX/1/1",
		Meta: gpxMeta{Name: views.Wordmark + ": plots of " + l.Surveyor.Label(), Link: gpxLink{Href: rd.abs(profilePath(l.Surveyor.Name))}},
	}
	for _, p := range l.Plots {
		g.Points = append(g.Points, gpxPt{
			Lat: coord(p.Lat), Lon: coord(p.Lng), Name: p.Plot.Name(), Desc: photosOf(l, p),
			Link: gpxLink{Href: rd.abs(p.Plot.Path())},
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(g); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

type geoCollection struct {
	Type     string       `json:"type"`
	Name     string       `json:"name"`
	Features []geoFeature `json:"features"`
}

type geoFeature struct {
	Type       string     `json:"type"`
	Geometry   geoPoint   `json:"geometry"`
	Properties geoDetails `json:"properties"`
}

type geoPoint struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"` // longitude, latitude
}

type geoDetails struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Photos int    `json:"photos"` // theirs in the plot
	First  bool   `json:"first"`  // they claimed the plot
}

func (rd *Renderer) writeGeoJSON(w io.Writer, l *views.SurveyorPlotList) error {
	c := geoCollection{Type: "FeatureCollection", Name: views.Wordmark + ": plots of " + l.Surveyor.Label(), Features: []geoFeature{}}
	for _, p := range l.Plots {
		c.Features = append(c.Features, geoFeature{
			Type:       "Feature",
			Geometry:   geoPoint{Type: "Point", Coordinates: [2]float64{round6(p.Lng), round6(p.Lat)}},
			Properties: geoDetails{Name: p.Plot.Name(), URL: rd.abs(p.Plot.Path()), Photos: p.Photos, First: p.First},
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

func round6(f float64) float64 {
	v, _ := strconv.ParseFloat(coord(f), 64)
	return v
}

func profilePath(name string) string { return "/surveyor/" + url.PathEscape(name) }

// Export formats of a profile: the file suffix after the Surveyor name.
const (
	exportGPX     = ".gpx"
	exportGeoJSON = ".geojson"
)

// exportPlots serves /surveyor/{name}.gpx or .geojson, redirecting to the
// canonical spelling of the name first, like the RSS feeds.
func (h *Handlers) exportPlots(w http.ResponseWriter, r *http.Request, name, suffix string) {
	l, err := h.src.SurveyorPlots(r.Context(), name)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if self := profilePath(l.Surveyor.Name) + suffix; r.URL.EscapedPath() != self {
		http.Redirect(w, r, self, http.StatusMovedPermanently)
		return
	}
	var buf bytes.Buffer
	ctype := "application/gpx+xml; charset=utf-8"
	if suffix == exportGeoJSON {
		ctype = "application/geo+json"
		err = h.rd.writeGeoJSON(&buf, l)
	} else {
		err = h.rd.writeGPX(&buf, l)
	}
	if err != nil {
		h.rd.log.Error("export", "path", r.URL.Path, "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", ctype)
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", l.Surveyor.Name+suffix))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf.Bytes())
	}
}
