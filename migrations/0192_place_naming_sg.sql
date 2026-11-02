-- How plots in Singapore are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md,
-- WP H), chosen from the Places section of docs/countries/singapore.md, the
-- 529 imported places inside Singapore's regions and Thailand's row in
-- 0075_place_naming.sql. Measured on all 938 land plots.
--
-- Singapore is one city: all five regions are urban (urban_regions), so
-- every planning area counts as an urban district. With a place=city
-- within radius_km as the only test, 11 % of the land plots would be urban
-- and 36.5 % named after a place; with all regions urban, 97.5 %.
--
-- Urban kinds: the neighbourhoods are the URA's subzones and estates
-- (Alexandra North, Ghim Moh, Tiong Bahru): 410 of them, 61 % of the land
-- plots have one within 1 km. The quarters (41) are older names (Sin Ming,
-- Telok Kurau) and the suburbs (77) mostly repeat the planning areas'
-- names (Bedok, Novena, Orchard), which the plot name shows as its
-- district anyway; so neighbourhood, quarter, suburb, in that order.
-- Thailand's order (quarter first) would take a quarter 3 km away over the
-- neighbourhood next door (median distance to the naming place 1.06 km
-- instead of 0.74 km). Thailand's 3 km: 97.5 % of the land plots are named
-- after a place of their own planning area or one nearby (91 % of them
-- within their own); 2 km would leave 7 % to the planning area's name
-- alone. The rest (Western Islands, the catchments, Tekong) falls back to
-- the villages (Ubin's kampung), then the city.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('sg', '{neighbourhood,quarter,suburb}', 3, '{village|town,city}', 5,
   '{central-region-singapore,east-region-singapore,north-region-singapore,north-east-region-singapore,west-region-singapore}');

-- Oddities in the imported place names of Singapore, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('sg', 'Marina East (mp)', 'neighbourhood',
   'Three neighbourhood names carry a qualifier in brackets: Marina East (mp) (the Marine Parade part of Marina East), Lakeside (business) and Lakeside (leisure) in Jurong. Keep, or write Marina East and Lakeside?');
