-- How plots in Timor-Leste are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md,
-- WP H), chosen from the Places section of docs/countries/timor-leste.md,
-- the 233 imported places and Thailand's row in 0075_place_naming.sql.
--
-- What the import holds: 119 villages, 53 towns, 4 cities, 56 neighbourhoods
-- and a quarter. The other 92 % of the place nodes in the extract (2,624
-- named hamlets, the aldeia) are not a place kind we import, as in Thailand
-- and Laos. They are what the plots of the countryside would be named
-- after; without them many plots below fall back to their district.
--
-- Inside a city. Only Dili has the kind that makes a city's plots: 55 of
-- the 56 neighbourhoods (bairro: Bairo Pite, Balide, Colmera, Taibesi, …)
-- lie within 5 km of its place=city node, half of them less than 0.34 km
-- and nine tenths less than 0.47 km from the next one; there is no suburb
-- and no quarter in Dili. So the neighbourhood is the urban kind, within
-- Thailand's 3 km. The three other cities (Gleno, Liquiçá, Pante-Macassar)
-- have none within 5 km; their plots go on to the villages and towns. A
-- plot is urban when a place=city lies within radius_km (296 of the 19,539
-- land plots); no municipality is urban as a whole, since Dili's eastern
-- post Metinaro and the hills south of the city are farmland and forest.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. Villages and towns are far apart here (the
-- median distance to the next village is 2.4 km, to the next town 10.6 km;
-- the 90th percentile 12.6 and 16.8 km), so 5 km finds a place for 44 % of
-- the land plots (villages and towns alone: 43 %, Thailand 64 %, Laos
-- 54 %); 8 km would find one for 77 % and 10 km for 90 %,
-- at a median of 4.5 and 5.1 km to the named place instead of 3.1 km. The
-- other plots are named after their administrative post and municipality,
-- which are real places here. A wider radius would name a plot after a
-- village over the next ridge; the lever that fits is the hamlets (a place
-- kind of its own, as for Taiwan), which Adrian decides.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('tl', '{neighbourhood}', 3, '{village|town,city}', 5, '{}');

-- Oddities in the imported place names of Timor-Leste, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('tl', 'Sede Suco Hoholau', 'village',
   'One of four village nodes named after the seat of a suco or aldeia ("sede") instead of the place: Sede Suco Hoholau, Sede Suco Laisorolai De Cima, SEDE SUCO LAISOROLAI DE BAIXO (in capitals, on the same spot) and Sede Aldeia Hatulai. Keep them as OSM has them, or drop "Sede Suco" / "Sede Aldeia" from the plot name?'),
  ('tl', 'Suco Bobocase', 'village',
   'Three village nodes carry the word of the unit in front of the name: Suco Bobocase, Suco Naimeco (both in Oe-Cusse Ambeno) and Aldeia Loidua. Keep, or write Bobocase, Naimeco and Loidua?'),
  ('tl', 'Posto Adminitrativo Quelicai', 'village',
   'A place=village node (n1785440270, -8.603, 126.560) named for the administrative post, with a misspelling ("Adminitrativo"), beside the town Quelicai (n1785407107). Keep, or replace by "Quelicai"?');
