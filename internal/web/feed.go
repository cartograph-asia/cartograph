package web

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// RSS 2.0 with an atom:link rel=self, as recommended by the RSS Advisory
// Board. Each item carries a title, the photo page link and the thumbnail;
// nothing more.

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Self          atomLink  `xml:"atom:link"`
	Items         []rssItem `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Description string  `xml:"description"`
}

type rssGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// rfc822 is RFC 822 with a four-digit year (RFC 1123), as RSS 2.0 expects.
func rfc822(t time.Time) string { return t.UTC().Format(time.RFC1123Z) }

// feedSelf is the feed's own path: its HTML page's path plus ".rss".
func feedSelf(f *views.Feed) string {
	p, _, _ := strings.Cut(f.Canonical, "?")
	if p == "" || p == "/" {
		p = "/latest"
	}
	return p + ".rss"
}

func (rd *Renderer) writeFeed(w io.Writer, f *views.Feed) error {
	updated := f.Updated
	if updated.IsZero() {
		updated = f.Now
	}
	ch := rssChannel{
		Title:         f.Title,
		Link:          rd.abs(f.Canonical),
		Description:   f.Description,
		Language:      "en",
		LastBuildDate: rfc822(updated),
		Self:          atomLink{Href: rd.abs(feedSelf(f)), Rel: "self", Type: "application/rss+xml"},
	}
	if ch.Title == "" {
		ch.Title = views.Wordmark
	}
	if ch.Description == "" {
		ch.Description = ch.Title
	}
	for _, p := range f.Items {
		link := rd.abs("/photo/" + p.ID)
		title := p.Plot.Name() + " by " + p.Surveyor.Name
		if p.FirstSurvey {
			title += " (first survey)"
		}
		d := dims(p, thumbPx)
		ch.Items = append(ch.Items, rssItem{
			Title:   title,
			Link:    link,
			GUID:    rssGUID{IsPermaLink: true, Value: link},
			PubDate: rfc822(p.UploadedAt),
			// HTML in description is entity-escaped by the XML encoder.
			Description: fmt.Sprintf(`<a href="%s"><img src="%s" width="%d" height="%d" alt="%s"></a>`,
				html.EscapeString(link), html.EscapeString(rd.absImg(p.Keys.Thumb)), d.W, d.H,
				html.EscapeString("Photo at "+p.Plot.Name()+" by "+p.Surveyor.Name)),
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(rss{Version: "2.0", Atom: "http://www.w3.org/2005/Atom", Channel: ch}); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
