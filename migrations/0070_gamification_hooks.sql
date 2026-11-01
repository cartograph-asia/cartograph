-- Room for docs/GAMIFICATION.md ("What the data model must allow"). Nothing
-- writes to these yet; scoring is not built.

-- Points ledger: one row per scoring event, never updated. Ranks and
-- leaderboards are sums over it; there is no score column on surveyors.
-- A correction (e.g. points released or withdrawn after review) is a new row.
create table points (
  id          bigint generated always as identity primary key,
  surveyor_id uuid not null references surveyors(id),
  cell        text references plots(cell),   -- plot the event belongs to, if any
  photo_id    uuid references photos(id),     -- photo that earned it, if any
  action      text not null,                  -- e.g. 'first_survey', 'additional_photo'
  points      int not null,
  multiplier  numeric(4, 2) not null default 1,
  created_at  timestamptz not null default now()
);

create index points_surveyor_idx on points(surveyor_id, created_at);
create index points_created_idx on points(created_at);

-- Regions are a typed hierarchy (country -> province -> city) linked by
-- parent_id, so territories of every level are queried the same way.
-- admin_level stays the OSM value it was imported from.
alter table regions add column kind text check (kind in ('country', 'province', 'city'));
update regions set kind = case admin_level when 2 then 'country' when 4 then 'province' else 'city' end;
alter table regions alter column kind set not null;
create index regions_kind_idx on regions(kind, parent_id);

-- Perceptual hash of the large variant, for duplicate and copy checks
-- (64-bit, stored as a signed bigint); null until something computes it.
alter table photos add column phash bigint;
