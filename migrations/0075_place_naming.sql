-- How plots are named, per country, as data instead of code. One row per
-- country; Thailand is the first.
--
-- A plot inside a city (its district's province is listed in urban_regions,
-- or a place=city lies within radius_km) is named after the nearest place of
-- the first of urban_kinds found within urban_radius_km. Otherwise, and when
-- none is found, the tiers in kinds apply within radius_km, first tier first;
-- "village|town" is one tier in which the nearest place wins. Without any,
-- the plot is named after its district and province only.
create table place_naming (
  country         text primary key,                -- ISO 3166-1 alpha-2, lower case
  urban_kinds     text[] not null,                 -- OSM place values inside cities, preferred first
  urban_radius_km real not null check (urban_radius_km > 0),
  kinds           text[] not null,                 -- tiers everywhere, preferred first; "a|b" shares a tier
  radius_km       real not null check (radius_km > 0),
  urban_regions   text[] not null default '{}'     -- province slugs whose districts are all urban
);

insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  -- In Thai OSM data place=quarter holds Bangkok's khwaeng, place=suburb its
  -- khet (shown from the district boundary anyway) and place=neighbourhood
  -- mostly housing estates; every district of Bangkok is a khet.
  ('th', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5, '{bangkok}');

-- Known oddities in place names, per country. A list to decide on later:
-- nothing reads it yet, so plots keep the names OSM gives them.
create table place_overrides (
  country     text not null references place_naming(country),
  name        text not null,                       -- place name_en as imported from OSM
  kind        text not null,                       -- its OSM place value
  action      text not null default 'undecided' check (action in ('undecided', 'keep', 'replace')),
  replacement text,                                -- the name to use when action is 'replace'
  note        text not null,
  primary key (country, name)
);

insert into place_overrides (country, name, kind, note) values
  ('th', 'Ban Bang Krabueang', 'village',
   'Old village name (place=village, node in Samut Prakan) that named plots along Sukhumvit in Watthana while that district was not recognised as urban. With every Bangkok district urban the khwaeng win there; it still names plots on the Samut Prakan side. Keep it, or replace such old village names by the khwaeng/tambon?');
