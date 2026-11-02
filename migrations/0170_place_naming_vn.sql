-- How plots in Vietnam are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/vietnam.md and the 18,405
-- places a local import stored (vietnam-latest.osm.pbf of 2026-11-01).
--
-- Inside a city. Since the 2025 reform a plot's district is its ward or
-- commune, so the plot name already carries the ward ("…, Bến Thành · Ho Chi
-- Minh 12"); what is wanted below it is the old village or quarter. In
-- Vietnamese OSM data place=quarter holds those (tổ dân phố, old villages
-- inside the wards: 2,942 nodes, 22 % of them numbered, median 0.69 km to the
-- next quarter near a city), place=suburb mostly the wards before 2025
-- (1,143 nodes, median 1.94 km apart), and place=neighbourhood is in Ho Chi
-- Minh City almost only numbered blocks: 564 of the 568 neighbourhoods within
-- 5 km of its city node are "Khu phố 24" and the like, which repeat in every
-- ward. So quarter first, then suburb, and no neighbourhood. Counted on the
-- 10,734 land plots with a place=city within 5 km (2.7 % of 400,405), the
-- nearest within 3 km, before the own-district rule:
--   {quarter,suburb}               7,010 named, 461 by a number, 1,092 "… Ward"
--   {suburb,quarter}               7,010 named, 147 by a number, 2,415 "… Ward"
--   {quarter,neighbourhood,suburb} 7,437 named, 700 by a number,   868 "… Ward"
-- (Thailand's order). Suburb first names a third of the plots after a node of
-- their own new ward ("Nam Triệu Ward, Nam Triệu"; see place_overrides).
-- 3 km as in Bangkok: within 2 km a suburb is found for 32.5 % of the urban
-- plots, within 3 km for 52.8 %. No province is urban as a whole: the six
-- cities governed from the centre (Hà Nội, Hồ Chí Minh, Hải Phòng, Đà Nẵng,
-- Cần Thơ, Huế) took in rural provinces in 2025 (Hồ Chí Minh City reaches
-- Côn Đảo). Their wards (kind_name 'ward', 709 of 3,319 districts) are urban
-- by law, but urban_regions takes provinces only.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. 51.5 % of Vietnam's land plots have a village or
-- town within 5 km (Thailand 64 %, Laos 54 %; 8 km would give 74.1 %); the
-- others are named after their commune and province. 3,295 hamlets stay out
-- as in Thailand and Laos.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('vn', '{quarter,suburb}', 3, '{village|town,city}', 5, '{}');

-- Oddities in the imported Vietnamese place names, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('vn', 'Thôn 1', 'village',
   'One of 1,717 villages (20 % of 8,506) named by a number only: Thôn 1 (145 times), Thôn 2 (145), Ấp 1 (27), Bản 3 …; they are unique only within their commune, which the plot name carries. Keep as OSM has them, or prefer a named village a little further away?'),
  ('vn', 'Nam Triệu Ward', 'suburb',
   'One of 628 places (389 suburbs, 213 towns, 20 cities, 3 quarters, 2 neighbourhoods, 1 village) whose English name ends in " Ward" or " Commune": nodes of the wards and communes of 2025 (191 match a district name exactly) or of the old towns that became wards. They name plots inside the same district twice ("Nam Triệu Ward, Nam Triệu · Hải Phòng"). Strip the word, drop such nodes from naming, or keep?');
