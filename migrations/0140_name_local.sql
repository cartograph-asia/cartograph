-- Names in the language of the country (docs/COUNTRY_IMPORT_BRIEF.md): the
-- Thai-only columns become language-neutral. A rename keeps every value; the
-- language follows from the country of the region (internal/countries), so
-- no language column is added.
alter table regions rename column name_th to name_local;
alter table places  rename column name_th to name_local;
alter table plots   rename column place_name_th to place_name_local;
