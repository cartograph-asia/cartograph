-- Camera data of a photo (docs/CAMERA_BRIEF.md): make, model, lens and the
-- exposure numbers, read from EXIF at upload and kept here because the
-- variants have their EXIF removed and the original is not kept. A table of
-- its own: photos and the view visible_photos stay as they are. A row exists
-- only for a photo that had at least one of the fields; no serial numbers, no
-- owner name, no software.
create table photo_camera (
  photo_id          uuid primary key references photos(id) on delete cascade,
  camera            text not null check (char_length(camera) between 1 and 200), -- "Apple iPhone 17 Pro": what statistics group by
  make              text check (char_length(make) <= 100),
  model             text check (char_length(model) <= 100),
  lens              text check (char_length(lens) <= 100),
  focal_length_mm   double precision check (focal_length_mm > 0 and focal_length_mm < 5000),
  focal_length_35mm smallint check (focal_length_35mm > 0 and focal_length_35mm < 5000),
  f_number          double precision check (f_number > 0 and f_number < 100),
  exposure_time_s   double precision check (exposure_time_s > 0 and exposure_time_s <= 86400),
  iso               int check (iso > 0 and iso < 5000000),
  flash_fired       boolean
);

create index photo_camera_camera_idx on photo_camera (lower(camera));
