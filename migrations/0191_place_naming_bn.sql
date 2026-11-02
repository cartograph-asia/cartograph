-- How plots in Brunei are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/brunei.md, the 157
-- imported places inside Brunei's four districts and Thailand's row in
-- 0075_place_naming.sql. Measured on all 7,864 land plots.
--
-- Brunei has no district level (decision 6), so the rule that pairs an
-- urban place with the plot's own district never applies, and urban kinds
-- would be tried strictly one after the other: a neighbourhood 3 km away
-- would win over a village next door. Brunei's towns are made of kampong,
-- mapped as villages (near Bandar Seri Begawan 0.87 km apart) and, in
-- Brunei-Muara and Tutong, as neighbourhoods (0.48 km apart), both called
-- "Kampong …". So there are no urban kinds, and villages, neighbourhoods
-- and towns share the first tier: the nearest of them within 5 km names
-- the plot, else the city. That names 27.9 % of the land plots (without
-- neighbourhoods in the tier 27.3 %, and 387 instead of 30 of them get a
-- neighbourhood rather than a village further off); 756 of Brunei-Muara's
-- 777. 8 km would name 37.7 %, at a median distance of 2.7 km instead of
-- 1.9 km; the rest of Belait, Tutong and Temburong is rainforest and is
-- named after its district. urban_radius_km is not used without urban
-- kinds; it carries Thailand's 3.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('bn', '{}', 3, '{village|neighbourhood|town,city}', 5, '{}');

-- Oddities in the imported place names of Brunei, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('bn', 'Ayer Village', 'village',
   'One of 15 kampong whose name:en translates the name ("Ayer Village" for Kampong Ayer, "Tali River Village" for Kampong Sungai Tali, likewise the town "Muara Town" for Pekan Muara), while 43 others are named "Kampong …" in English too. The Malay name is stored as the local name. Keep the translations, or name the plots "Kampong Ayer"?');
