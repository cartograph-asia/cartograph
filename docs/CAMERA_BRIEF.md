# Brief: Camera metadata of uploaded photos

Read `CLAUDE.md` (image pipeline: EXIF is stripped except the fields we keep in the DB; Alpha: data is kept), `docs/PRINCIPLES.md` and the Admin and Display name entries in `docs/decisions.md` first. This brief keeps a few **camera tags** of every photo at upload, so that they can be shown on the photo page and, later, counted in statistics ("cameras used", "typical focal lengths").

## Why

The upload reads the EXIF block for the position and the taken date and then throws the rest away: only the three variants are stored, with their EXIF removed, and the original is not kept. Whatever is not written to the database at that moment is gone for good, so the fields worth having later must be read now. Geograph shows camera data on its photo pages; for Cartograph it is also what a statistics page needs.

## Decisions (Adrian, 2026-11-02)

1. **A short list, nothing else.** Kept: camera make and model, lens model, focal length (and the 35 mm equivalent), aperture, exposure time, ISO, whether the flash fired. **Never read or kept:** body or lens serial numbers, camera owner name, firmware, software, MakerNote, thumbnails, comments, any GPS field beyond the position we already keep. Those can identify a person or a device.
2. **Text from a file is untrusted.** Make, model and lens are cleaned (control characters removed, spaces collapsed, cut at 100 characters) and printed escaped; they are never a link, a class or an address.
3. **One normalised label** per photo (`camera`, e.g. "Apple iPhone 17 Pro", "Canon EOS R5", "NIKON D850") is stored next to the raw make and model, so a statistics query groups by one column. The make loses corporate suffixes ("NIKON CORPORATION" becomes "NIKON"), and a model that already starts with the make is not prefixed twice. Capitalisation is kept as the camera wrote it; statistics compare `lower(camera)`.
4. **Only new uploads.** Photos uploaded before this have no camera data and cannot get any (no original is kept). They show nothing, not "unknown".
5. **Shown on the photo page** as one plain line under the licence, e.g. "Apple iPhone 17 Pro · 6.8 mm (48 mm equiv.) · f/1.8 · 1/16129 s · ISO 64", only when there is data. No statistics page and no per-camera pages in this brief.
6. **No effect on anything else:** no points, no ranking, no filter (Principles 2, 6 and 8).

## Data

One migration (propose 0130; the lead assigns the number, 0110 is reserved for comments and 0120 for sights). A table of its own, so `photos` and the view `visible_photos` stay as they are:

```
photo_camera (
  photo_id          uuid primary key references photos(id) on delete cascade,
  camera            text not null check (char_length(camera) between 1 and 200),
  make, model, lens text check (char_length <= 100),

  focal_length_35mm smallint check (> 0 and < 5000),
  f_number          real check (> 0 and < 100),
  exposure_time_s   real check (> 0 and <= 86400),
  iso               int check (> 0 and < 5000000),
  flash_fired       boolean
)
```

with an index on `lower(camera)`. A row exists only when the photo had at least a make, a model or one of the numbers. Values that do not parse or fall outside the checks are left null, they never fail an upload.

## Code

- `internal/photos`: `Meta.Camera *Camera` filled by `ReadMeta`/`ReadMetaFrom` from the same EXIF block, nil without any camera tag; the cleaning and the label (decision 3) live here.
- `internal/db`: `NewPhoto.Camera`, written by `ClaimPhoto` in the same transaction as the photo; `Views.Photo` reads it.
- `internal/api` (the upload) and `cmd/seed` pass `meta.Camera` on; the synthetic seed photos have none.
- `cmd/seed` (development data): `SeedCameras` gives the photos of the development Surveyors (addresses at `example.test`) that have no camera data fake camera data from a list of twelve models (phones, mirrorless cameras, a compact, an action camera) with plausible lens settings, ISO and exposure for daylight in Thailand. A Surveyor uses one camera in four photos of five; the same photo always gets the same data; photos that have data and the photos of real accounts are left alone, so a second run does nothing. It works on whatever photos are there, so the Commons seed (`docs/SEED_PHOTOS_BRIEF.md`) is covered without a change. The list has the label cases statistics find hard on purpose: "samsung" in lower case, "NIKON CORPORATION", "RICOH IMAGING COMPANY, LTD.", a model that repeats its make.
- `internal/views`: `Photo.Camera *Camera` with a `Line()` that formats the line of decision 5; `viewstest` gets a camera on the featured photo.
- `internal/web`: the line in `photo.html`.

## Tests

- **photos:** the iPhone photo `testdata/IMG_1911.jpeg` gives make, model, lens, 6.8 mm, 48 mm, f/1.8, 1/16129 s, ISO 64, flash off; a photo without camera tags gives nil; label cases (Apple + "iPhone 17 Pro", Canon + "Canon EOS R5", "NIKON CORPORATION" + "NIKON D850", only a model, only a make); control characters, spaces and 300 characters are cleaned; no serial or owner field is exposed by `Camera`.
- **db:** `ClaimPhoto` with and without a camera; the row goes with the photo; absurd numbers are refused by the checks and a null in their place never fails a claim; `Views.Photo` returns the camera.
- **seed:** every fake camera has a label and valid numbers; a photo always gets the same data; a Surveyor's own camera is in about four photos of five; `SeedCameras` fills the seeded photos once, does nothing the second time, keeps existing data and leaves a real account's photo without.
- **api:** an upload of a photo with camera tags stores them; one without stores no row.
- **web:** the line appears with data and not without; a model like `<script>alert(1)</script>` is shown as text; vocabulary and page tests; no `style` attribute.

## Done means

Uploading `testdata/IMG_1911.jpeg` through the API gives a photo page with "Apple iPhone 17 Pro · 6.8 mm (48 mm equiv.) · f/1.8 · 1/16129 s · ISO 64" and one row in `photo_camera`; an upload without camera tags gives a page without the line. `go test ./...` green with `TEST_DATABASE_URL` set; `/security-review` run.

## Open questions

- **Hide it.** A Surveyor might not want the camera shown (a rare or expensive device says something). A setting in `/settings` that hides the line, while still storing the data for counts, is easy to add; not done here.
- **Statistics.** Which numbers and where (a page `/about/cameras`, a region page, the profile) is for a later brief; they must be counts, never "best camera" rankings.
- **Lens and body for phones.** Phones put the lens in `LensModel` ("iPhone 17 Pro back triple camera 6.765mm f/1.78"); it repeats the other fields, so the page line leaves the lens out and the statistics can use it.
