-- Places per country (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, Places silo): a
-- country's import replaces only its own places. The country of a stored
-- place is that of its region (a province), else Thailand, the only country
-- imported so far. The backfill only fills the new column.
alter table places add column country text;

update places p set country = r.country from regions r where r.id = p.region_id;
update places set country = 'th' where country is null;

alter table places alter column country set not null;

create index places_country_idx on places(country);
