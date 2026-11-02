-- How plots in the Philippines are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md,
-- WP H), chosen from the Places section of docs/countries/philippines.md,
-- the 49,321 imported places, the 368,516 land plots and Thailand's row in
-- 0075_place_naming.sql.
--
-- What the place nodes are. The unit every Filipino address ends in is the
-- barangay; OSM maps it as place=village in the countryside and as
-- place=quarter (sometimes suburb) in towns and cities, with admin_level=10
-- on 21,811 villages, 5,742 quarters and 513 suburbs. place=town (1,491
-- with admin_level=6) and place=city (149) are the centres of the
-- municipalities and cities. place=neighbourhood is below the barangay:
-- 3,969 puroks ("Purok 4"), sitios and some 2,000 subdivisions ("BF Homes
-- Phase 3"); a plot is not named after them. The country has no district
-- level in the import (decision 2), so the rule that prefers places of the
-- plot's own district never applies here.
--
-- Inside a city. A plot is urban when a place=city lies within radius_km
-- (9,228 of the land plots; Metro Manila and the independent cities, whose
-- city nodes these are, have no plots of their own, see
-- internal/countries/import_ph.go). There the barangay is the quarter: half
-- of the quarters lie less than 0.50 km from the next one, nine tenths less
-- than 2.36 km. A quarter wins over every nearer village or town, so the
-- radius stays near that spacing: 2 km names 4,953 urban plots after a
-- quarter (1 km 2,646, 3 km 6,749, with more of them a quarter 2–3 km off
-- beside a village a few hundred metres away). Neighbourhoods are left out
-- (the scan's proposal of {quarter,neighbourhood,suburb} would name 1,639
-- plots after puroks and subdivisions), and so are suburbs, which mix
-- barangays with the districts of Manila and Davao.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. Barangay nodes are dense (the median distance
-- from a village to the next is 1.39 km), so 5 km finds a place for 81.8 %
-- of the land plots (Thailand 64 %, Laos 54 %), at a median distance of
-- 1.6 km; 8 km would find one for 91.9 %, at a median of 1.8 km and a 90th
-- percentile of 5.2 km instead of 3.8 km. The others lie in the mountains
-- and forests of Luzon, Mindanao and Palawan and are named after their
-- province alone, since there is no district. Quarters and suburbs stay out
-- of the tiers: in the towns they are the poblacion barangays, and with
-- them in the village|town tier 3,211 plots instead of 544 would be named
-- "Poblacion" rather than after the town node beside it.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('ph', '{quarter}', 2, '{village|town,city}', 5, '{}');

-- Oddities in the imported place names of the Philippines, to decide on
-- later (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('ph', 'Poblacion', 'quarter',
   'The town-centre barangay of a municipality. 667 nodes are named just "Poblacion" (489 quarters, 100 suburbs, 50 villages, 28 neighbourhoods) and 509 "Poblacion I", "Poblacion 2" and so on; by these rules they name about 900 plots. Without a district level the plot name does not say which town ("Poblacion · Batangas 12"). Keep them as OSM has them, or write "Poblacion, <town>" with the nearest place=town?'),
  ('ph', 'Barangay 1', 'quarter',
   'Numbered barangays and zones of the cities and some towns: 601 quarters, 317 neighbourhoods and 22 villages named "Barangay <n>" (34 quarters "Barangay 1" alone), 324 "Zone <n>". They name about 340 plots; most of Manila''s, Pasay''s and Caloocan''s lie in Metro Manila and name none yet. Keep, or name such plots after the next place with a real name?');
