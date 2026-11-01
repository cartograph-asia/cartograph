-- Room for filtering and counting photos by tag, year and season. Region and
-- Surveyor already work (photos.cell -> plots.region_id, photos.surveyor_id).
-- Nothing tags photos yet; seasons are derived later from the local month and
-- the region, once real data exists.

-- Tags: one row per tag, attached to photos through photo_tags.
create table tags (
  id         int generated always as identity primary key,
  slug       text not null unique check (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  name       text not null,
  created_at timestamptz not null default now()
);

create table photo_tags (
  photo_id   uuid not null references photos(id) on delete cascade,
  tag_id     int not null references tags(id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key (photo_id, tag_id)
);

create index photo_tags_tag_idx on photo_tags(tag_id, photo_id);

-- Local capture time: DateTimeOriginal as the camera's wall clock, so year and
-- month do not shift when taken_at (UTC) crosses midnight. The offset comes
-- from OffsetTimeOriginal and is null when the photo has none (taken_at is
-- then the wall clock read as UTC). Prototype rows stay null.
alter table photos add column taken_at_local timestamp;
alter table photos add column taken_utc_offset smallint; -- minutes east of UTC

create index photos_taken_local_idx on photos(taken_at_local);
