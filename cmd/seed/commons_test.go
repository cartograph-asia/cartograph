package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/commons"
	"github.com/cartograph-asia/cartograph/internal/importer"
	"github.com/cartograph-asia/cartograph/internal/photos"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// A JPEG from Commons has a JFIF header and no EXIF; withExif must still give
// it a position and a date that the upload pipeline reads.
func TestWithExifOnJFIF(t *testing.T) {
	plain, err := synthPhoto(3, 1, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Drop the synthetic EXIF segment and put a JFIF APP0 in its place.
	exifLen := int(plain[4])<<8 | int(plain[5])
	jfif := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 1, 0, 1, 0, 0}
	j := append(jfif, plain[4+exifLen:]...)
	if m := photos.ReadMeta(j); m.HasGPS {
		t.Fatal("test JPEG still has GPS")
	}
	taken := time.Date(2024, 3, 4, 3, 22, 11, 0, time.UTC)
	out, err := withExif(j, 18.7883, 98.9853, taken)
	if err != nil {
		t.Fatal(err)
	}
	m := photos.ReadMeta(out)
	if !m.HasGPS || m.Lat < 18.788 || m.Lat > 18.789 || m.Lng < 98.985 || m.Lng > 98.986 || m.TakenAt == nil || !m.TakenAt.Equal(taken) {
		t.Fatalf("meta %+v", m)
	}
	if _, err := photos.Process(out); err != nil {
		t.Fatal(err)
	}
}

func TestSeedCommons(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	osmPath, waterPath := importer.Fixture()
	if _, err := importer.Regions(ctx, d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Places(ctx, d, importer.PlaceOptions{PBF: osmPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}

	// Twelve photos in Pattaya, dealt to six Surveyors, and one in Chiang Mai,
	// where no region is imported.
	dir := t.TempDir()
	m := &commons.Manifest{}
	add := func(i int, lat, lng float64) {
		raw, err := synthPhoto(uint64(100+i), 0, 0, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		file := fmt.Sprintf("p%d.jpg", i)
		if err := os.WriteFile(filepath.Join(dir, file), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		m.Photos = append(m.Photos, commons.Photo{
			File: file, Title: "File:X.jpg", PageURL: "https://commons.wikimedia.org/wiki/File:X.jpg", Place: "test",
			Lat: lat, Lng: lng, Author: "Someone", Licence: "CC BY-SA 4.0",
		})
	}
	for i, sp := range plan(12, 12) {
		add(i, sp.lat, sp.lng)
	}
	add(12, 18.7883, 98.9853)
	if err := commons.Write(dir, m); err != nil {
		t.Fatal(err)
	}

	store := storage.NewMemory("http://img.test")
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	st, err := SeedCommons(ctx, d, store, now, dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Surveyors != 6 || st.Photos != 12 || st.Outside != 1 || len(store.Keys()) != 36 {
		t.Fatalf("first run %+v, %d keys", st, len(store.Keys()))
	}
	st, err = SeedCommons(ctx, d, store, now.Add(time.Hour), dir)
	if err != nil {
		t.Fatal(err)
	}
	// The photo outside the regions is tried again, everything else is there.
	if st.Surveyors != 0 || st.Photos != 0 || st.Skipped != 12 || st.Outside != 1 || len(store.Keys()) != 36 {
		t.Fatalf("second run %+v", st)
	}

	v := d.Views().WithNow(func() time.Time { return now })
	for _, c := range commonsSurveyors {
		s, err := v.Surveyor(ctx, c.name, views.SurveyorQuery{})
		if err != nil || s.Stats.Photos != 2 || s.Profile.Display != c.display {
			t.Fatalf("%s: %+v %v", c.name, s, err)
		}
		for _, p := range s.Photos {
			if p.TakenAt == nil || p.UploadedAt.After(now) || p.UploadedAt.Before(s.Profile.JoinedAt) {
				t.Fatalf("%s: photo %+v", c.name, p)
			}
		}
	}
	// A manifest naming a file outside the directory is refused.
	m.Photos[0].File = "../p0.jpg"
	if err := commons.Write(dir, m); err != nil {
		t.Fatal(err)
	}
	if _, err := SeedCommons(ctx, d, store, now, dir); err == nil {
		t.Fatal("manifest with a path accepted")
	}
}
