-- Store the H3 parents of each plot at the two overview resolutions so that
-- coverage per parent cell is a simple indexed group-by.
-- The backend fills these columns when it creates a plot.

alter table plots add column parent5 text;
alter table plots add column parent6 text;

create index plots_parent5_idx on plots(parent5);
create index plots_parent6_idx on plots(parent6);
