-- The word a region is called by on its page ("prefecture", "state",
-- "region", "commune", ...), set by the import from the country's settings
-- (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, decision 3). NULL: the word follows
-- from kind, which stays structural ('province', 'district').
alter table regions add column kind_name text;
