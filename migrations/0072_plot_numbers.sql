-- Plot numbers: a running number per province, so a plot can be named
-- "Pattaya · Chon Buri 17" and reached at /plot/chon-buri/17. The region
-- import numbers every land cell along a Hilbert curve (neighbouring plots
-- get neighbouring numbers); a plot claimed without a number gets the next
-- free one. Numbers are never changed or reused, so a re-import only adds.
-- Blank cells have numbers too, so the table does not reference plots.
create table plot_numbers (
  cell      text primary key,                   -- H3 res-8 index
  region_id int not null references regions(id),
  number    int not null check (number > 0),
  unique (region_id, number)
);
