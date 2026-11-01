-- Initial schema for the prototype (see CLAUDE.md, "Data model").

create table plots (
  cell           text primary key,          -- H3 res-8 index
  first_photo    uuid,
  first_surveyor text,
  claimed_at     timestamptz,
  photo_count    int not null default 0
);

create table photos (
  id          uuid primary key,
  cell        text not null references plots(cell),
  surveyor    text not null,
  key_large   text not null,
  key_thumb   text not null,
  lat         double precision not null,
  lng         double precision not null,
  taken_at    timestamptz,
  uploaded_at timestamptz not null default now()
);

create index on photos(cell);
