package main

import (
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/commons"
)

func TestPlain(t *testing.T) {
	got := plain(`<a href="//commons.wikimedia.org/wiki/User:X" title="User:X">Jane  &amp; Joe</a>
	<span>Doe</span>`)
	if got != "Jane & Joe Doe" {
		t.Errorf("plain: %q", got)
	}
}

func TestTaken(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	if g := taken("2024-03-04 10:22:11", now); g == nil || g.UTC().Hour() != 3 {
		t.Errorf("Bangkok time to UTC: %v", g)
	}
	for _, s := range []string{"", "yesterday", "2030-01-01 00:00:00", "2001-05-05 10:00:00"} {
		if g := taken(s, now); g != nil {
			t.Errorf("%q: %v", s, g)
		}
	}
}

func TestLicences(t *testing.T) {
	for s, ok := range map[string]bool{
		"CC0": true, "CC BY 2.0": true, "CC BY-SA 4.0": true, "cc by-sa 3.0 de": true,
		"CC BY-NC 2.0": false, "CC BY-ND 2.0": false, "CC BY-NC-SA 4.0": false, "Public domain": false, "": false, "GFDL": false,
	} {
		if commons.AllowedLicence(s) != ok {
			t.Errorf("%q: want %v", s, ok)
		}
	}
}

func TestValidate(t *testing.T) {
	good := commons.Photo{File: "p1.jpg", PageURL: "https://commons.wikimedia.org/wiki/File:X.jpg", Author: "A", Licence: "CC BY 4.0", Lat: 18.79, Lng: 98.98}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mod := range map[string]func(*commons.Photo){
		"path in the name": func(p *commons.Photo) { p.File = "../p1.jpg" },
		"not a jpg":        func(p *commons.Photo) { p.File = "p1.png" },
		"licence":          func(p *commons.Photo) { p.Licence = "CC BY-NC 4.0" },
		"no author":        func(p *commons.Photo) { p.Author = "" },
		"outside Thailand": func(p *commons.Photo) { p.Lat, p.Lng = 51.5, -0.12 },
	} {
		p := good
		mod(&p)
		if p.Validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestTooClose(t *testing.T) {
	have := []commons.Photo{{Lat: 18.7883, Lng: 98.9853}}
	if !tooClose(have, commons.Photo{Lat: 18.7885, Lng: 98.9855}) {
		t.Error("30 m apart is too close")
	}
	if tooClose(have, commons.Photo{Lat: 18.8000, Lng: 98.9853}) {
		t.Error("1.3 km apart is fine")
	}
}

func TestAPIError(t *testing.T) {
	if got := apiError([]byte(`{"error":{"code":"maxlag","info":"Waiting for a database server"}}`)); got != "maxlag" {
		t.Errorf("maxlag: %q", got)
	}
	for _, ok := range []string{`{"query":{"geosearch":[]}}`, `not json`, `\xff\xd8`} {
		if got := apiError([]byte(ok)); got != "" {
			t.Errorf("%q: %q", ok, got)
		}
	}
}
