-- Attach photos and plots to accounts and add the medium variant.
-- The text columns photos.surveyor and plots.first_surveyor stay as they are
-- (the prototype's record); the ids are filled for new uploads and, for
-- prototype photos, by the placeholder migration (0050+). Until then they
-- are nullable.

alter table photos add column surveyor_id uuid references surveyors(id);
alter table photos add column key_medium text;     -- longest side 1200 px; null for prototype photos
alter table photos add column width int;           -- pixels of the large variant; null if unknown
alter table photos add column height int;
alter table photos add column licence text not null default 'CC BY-SA 4.0';

alter table plots add column first_surveyor_id uuid references surveyors(id);

create index photos_surveyor_idx on photos(surveyor_id, uploaded_at desc);
create index photos_uploaded_idx on photos(uploaded_at desc, id desc);
create index photos_cell_uploaded_idx on photos(cell, uploaded_at, id);
create index plots_first_surveyor_idx on plots(first_surveyor_id);
