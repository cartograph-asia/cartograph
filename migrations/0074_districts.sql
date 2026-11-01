-- Districts (OSM admin_level 6: Bangkok's khet, elsewhere amphoe) as regions
-- of kind 'district' below their province (parent_id). They name plots
-- ("Hua Mak, Bang Kapi · Bangkok 1000") and tell city from countryside:
-- a khet is urban. Plots and coverage stay per province; districts have no
-- numbers, counts or pages of their own yet.
alter table regions drop constraint regions_kind_check;
alter table regions add constraint regions_kind_check
  check (kind in ('country', 'province', 'district', 'city'));
alter table regions add column urban boolean not null default false; -- a district that is part of a city (khet)

alter table plots add column district_id int references regions(id) on delete set null;
