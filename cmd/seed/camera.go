package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand/v2"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/photos"
)

// A mode is one lens setting of a fake camera.
type mode struct {
	lens      string
	focalMM   float64
	equiv     int
	fNumber   float64
	phone     bool // phones shoot at low ISO in daylight and have no flash fired in the sun
	exposures []float64
}

// fakeCamera is a camera model of the development data (docs/CAMERA_BRIEF.md):
// what a Surveyor in Thailand might carry.
type fakeCamera struct {
	make, model string
	modes       []mode
}

var daylight = []float64{1.0 / 125, 1.0 / 250, 1.0 / 500, 1.0 / 1000, 1.0 / 2000}

var fakeCameras = []fakeCamera{
	{"Apple", "iPhone 17 Pro", []mode{
		{"iPhone 17 Pro back triple camera 6.765mm f/1.78", 6.8, 48, 1.8, true, []float64{1.0 / 250, 1.0 / 1000, 1.0 / 4000, 1.0 / 16129}},
		{"iPhone 17 Pro back triple camera 2.22mm f/2.2", 2.2, 13, 2.2, true, []float64{1.0 / 500, 1.0 / 2000}},
	}},
	{"Apple", "iPhone 15", []mode{
		{"iPhone 15 back dual wide camera 5.96mm f/1.6", 6, 26, 1.6, true, []float64{1.0 / 120, 1.0 / 500, 1.0 / 2000}},
	}},
	{"samsung", "SM-S921B", []mode{ // Galaxy S24
		{"", 6.3, 24, 1.7, true, []float64{1.0 / 200, 1.0 / 800, 1.0 / 3000}},
		{"", 2.2, 13, 2.2, true, []float64{1.0 / 400, 1.0 / 1500}},
	}},
	{"Google", "Pixel 9", []mode{
		{"", 6.9, 25, 1.7, true, []float64{1.0 / 250, 1.0 / 1000, 1.0 / 2500}},
	}},
	{"Xiaomi", "2311DRK48G", []mode{ // 13T Pro
		{"", 6.5, 23, 1.9, true, []float64{1.0 / 150, 1.0 / 600, 1.0 / 2000}},
	}},
	{"Canon", "Canon EOS R6m2", []mode{
		{"RF24-105mm F4 L IS USM", 35, 35, 5.6, false, daylight},
		{"RF24-105mm F4 L IS USM", 85, 85, 4, false, daylight},
		{"RF24-105mm F4 L IS USM", 24, 24, 8, false, daylight},
	}},
	{"SONY", "ILCE-7M4", []mode{
		{"FE 24-70mm F2.8 GM II", 24, 24, 5.6, false, daylight},
		{"FE 24-70mm F2.8 GM II", 50, 50, 4, false, daylight},
		{"FE 24-70mm F2.8 GM II", 70, 70, 2.8, false, daylight},
	}},
	{"NIKON CORPORATION", "NIKON Z 6_3", []mode{
		{"NIKKOR Z 24-120mm f/4 S", 24, 24, 8, false, daylight},
		{"NIKKOR Z 24-120mm f/4 S", 70, 70, 5.6, false, daylight},
	}},
	{"FUJIFILM", "X100V", []mode{
		{"", 23, 35, 5.6, false, daylight},
		{"", 23, 35, 2, false, daylight},
	}},
	{"RICOH IMAGING COMPANY, LTD.", "RICOH GR III", []mode{
		{"", 18.3, 28, 5.6, false, daylight},
	}},
	{"OLYMPUS CORPORATION", "OM-1", []mode{
		{"M.Zuiko Digital ED 12-45mm F4.0 PRO", 12, 24, 5.6, false, daylight},
		{"M.Zuiko Digital ED 12-45mm F4.0 PRO", 25, 50, 4, false, daylight},
	}},
	{"GoPro", "HERO12 Black", []mode{
		{"", 2.5, 16, 2.5, true, []float64{1.0 / 240, 1.0 / 960}},
	}},
}

// primary is the camera a Surveyor mostly uses, the same one on every run.
func primary(name string) int {
	h := fnv.New32a()
	h.Write([]byte(name))
	return int(h.Sum32()) & 0x7fffffff % len(fakeCameras)
}

// cameraFor makes the fake camera data of one photo: its Surveyor's own camera
// in four photos of five, another one in the fifth; the lens setting, ISO and
// exposure vary per photo. The same photo always gets the same data.
func cameraFor(surveyor string, photo uuid.UUID) *photos.Camera {
	h := fnv.New64a()
	h.Write(photo[:])
	r := rand.New(rand.NewPCG(h.Sum64(), 0xCA))
	i := primary(surveyor)
	if r.IntN(5) == 0 {
		i = (i + 1 + r.IntN(len(fakeCameras)-1)) % len(fakeCameras)
	}
	fc := fakeCameras[i]
	m := fc.modes[r.IntN(len(fc.modes))]
	iso := []int{100, 100, 125, 200, 200, 400, 800}[r.IntN(7)]
	if m.phone {
		iso = []int{25, 32, 40, 50, 64, 80, 125, 250}[r.IntN(8)]
	}
	flash := false
	return &photos.Camera{
		Make: fc.make, Model: fc.model, Label: photos.Label(fc.make, fc.model), Lens: m.lens,
		FocalMM: m.focalMM, Focal35MM: m.equiv, FNumber: m.fNumber,
		ExposureS: m.exposures[r.IntN(len(m.exposures))], ISO: iso, Flash: &flash,
	}
}

// SeedCameras gives every photo of the development Surveyors (addresses at
// example.test) that has no camera data fake camera data, and returns how
// many it filled. Photos that have some are left alone, so a second run does
// nothing. It works on the photos that are there, whichever seed made them.
func SeedCameras(ctx context.Context, d *db.DB) (int, error) {
	rows, err := d.Pool.Query(ctx, `
		select ph.id, s.name
		from visible_photos ph join surveyors s on s.id = ph.surveyor_id
		where s.email like '%@example.test'
		  and not exists (select 1 from photo_camera pc where pc.photo_id = ph.id)
		order by ph.uploaded_at, ph.id`)
	if err != nil {
		return 0, fmt.Errorf("seed: cameras: %w", err)
	}
	type todo struct {
		id   uuid.UUID
		name string
	}
	var list []todo
	for rows.Next() {
		var t todo
		if err := rows.Scan(&t.id, &t.name); err != nil {
			rows.Close()
			return 0, fmt.Errorf("seed: cameras: %w", err)
		}
		list = append(list, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("seed: cameras: %w", err)
	}
	for _, t := range list {
		if err := d.AddCamera(ctx, t.id, cameraFor(t.name, t.id)); err != nil {
			return 0, err
		}
	}
	return len(list), nil
}
