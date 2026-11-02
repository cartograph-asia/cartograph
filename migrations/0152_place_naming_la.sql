-- How plots in Laos are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP G),
-- chosen from the Places section of docs/countries/laos.md, the imported
-- places and Thailand's row in 0075_place_naming.sql.
--
-- Inside a city. Lao towns are made of villages (ban): within 5 km of
-- Vientiane's place=city node lie 85 English-named villages, half of them
-- less than 0.46 km from the next one (Pakse 0.32 km, Luang Prabang
-- 0.63 km), and not one quarter, suburb or neighbourhood; the few nodes of
-- those kinds lie in small towns (Muang Sing, the Golden Triangle zone) or
-- on the Thai bank. So the village is the urban kind, as the khwaeng is in
-- Bangkok: a plot is named after the nearest village of its own district
-- within 2 km, then a town. 2 km rather than Bangkok's 3 km: with villages
-- half a kilometre apart, 3 km would prefer a village of the own district
-- two kilometres off to one across the district line a few hundred metres
-- away. A plot is urban when a place=city lies within radius_km (1,277 of
-- Laos's land plots); no province is urban as a whole, since Vientiane
-- Prefecture's outer districts (Sangthong, Parkngum, Naxaithong) are
-- farmland and forest.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. Laos has about 70 % of Thailand's named places
-- per square kilometre, so 54 % of its land plots have a village or town
-- within 5 km (Thailand 64 %; 8 km would give 74 %); the others are named
-- after their district and province, which in the mountains is the honest
-- name. A wider radius would name a plot after a village beyond a ridge.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('la', '{village,town}', 2, '{village|town,city}', 5, '{}');

-- Oddities in the imported Lao place names, to decide on later (nothing
-- reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('la', 'B. Kalak', 'village',
   'One of 170 villages whose English name abbreviates Ban as "B." (87 in Savannakhet, 54 in Sekong, 26 in Xaisomboun; the Lao names write ບ. too), while 1,497 others spell out "Ban …". Keep them as OSM has them, or write "Ban" in the plot name?'),
  ('la', 'Village', 'village',
   'A place=village node in Savannakhet (n2396301394, 16.530, 105.800) whose name is just "Village", with no Lao name. Keep, or replace by its real name once known?');
