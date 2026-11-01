-- Where a plot is: its region (the one containing the cell's centre) and its
-- display name (nearest place within 5 km, else the region's name), both set
-- on first claim. The names are copied, not referenced, so re-importing
-- places never renames a plot.

alter table plots add column region_id int references regions(id);
alter table plots add column place_name text;      -- English display name
alter table plots add column place_name_th text;   -- Thai display name; may be null

create index plots_region_idx on plots(region_id);
create index plots_claimed_idx on plots(claimed_at desc);
