-- How plots in Malaysia are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/malaysia.md, the 10,267
-- imported places inside Malaysia's states and Thailand's row in
-- 0075_place_naming.sql. Measured on every fifth of the 419,792 land plots.
--
-- Inside a city. Malaysian towns are made of housing estates (taman,
-- kampung), mapped as place=neighbourhood: within 5 km of the city nodes
-- half of them lie less than 0.57 km from the next one in Kuala Lumpur,
-- 0.44 km in Kajang, 0.43 km in Ipoh, 0.23 km in Kota Kinabalu. Quarters
-- are the numbered sections of the planned cities (Section 17, SS 2, PJU 7
-- in Petaling Jaya, Shah Alam, Subang Jaya), dense there (0.55–0.88 km) and
-- nearly absent elsewhere (one within 5 km of Kuala Lumpur's centre);
-- suburbs (bandar, 1.5 km apart) are larger. So the neighbourhood comes
-- first, then quarter and suburb; Thailand's order (quarter first) would
-- name 245 of the sampled urban plots after a section up to 3 km away
-- instead of the estate next door (17 with this order). Thailand's 3 km:
-- 84 % of the urban plots get a neighbourhood, quarter or suburb (2 km:
-- 68 %; the rest go on to a town or the city). A plot is urban when a
-- place=city lies within radius_km (1.3 % of the land plots). No state is
-- urban as a whole; Kuala Lumpur, Putrajaya, Labuan and Perlis have no
-- districts, so urban_regions could not mark them anyway.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. Only 32 % of Malaysia's land plots have a
-- village or town within 5 km (Thailand 64 %, Laos 54 %, Timor-Leste 44 %;
-- 8 km would give 49 %, at a median distance of 3.7 km instead of 2.6 km):
-- Sarawak and Sabah, 63 % of the land plots, are mostly forest and
-- plantation, and the 2,853 hamlets are not a place kind we import. The
-- other plots are named after their district and state.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('my', '{neighbourhood,quarter,suburb}', 3, '{village|town,city}', 5, '{}');

-- Oddities in the imported Malaysian place names, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('my', 'Kg. Abul', 'village',
   'One of 161 villages and estates whose name abbreviates Kampung as "Kg." (154 in Sabah, 7 in Perak), while 3,207 others spell out "Kampung …". Keep them as OSM has them, or write "Kampung" in the plot name?'),
  ('my', 'KAMPUNG AIR MASJID', 'village',
   'One of 51 place names written in capitals (28 in Sabah, 20 in Selangor), e.g. KAMPUNG SUNGAI RAMBAI, KAMPUNG SINASAK BATU PITAS, 89100, KOTA MARUDU, SABAH. Keep, or write them in title case?'),
  ('my', 'ADVANCE ECO RESOURCES SDN BHD', 'village',
   'Six place=village nodes in Sabah (Tenom, Keningau) are companies or mills, not villages: ADVANCE ECO RESOURCES SDN BHD, EK HONG AGRICULTURE SDN BHD, SIME DARBY PLANTATION (SABAH) SDN. BHD. (MELALAP OIL MILL), TANER R&D PALM OIL MILL SDN BHD, TENOM AGRICULTURE DEVELOPMENT SDN BHD, KILANG SMR TENOM. Keep, or leave them out of plot names?'),
  ('my', '17, Jalan Belian 8, Taman Rinting, 81750, Masai, Johor', 'neighbourhood',
   'Six neighbourhood nodes in Masai, Johor (17 to 22, Jalan Belian 8) are postal addresses of single houses. Replace by "Taman Rinting", or leave them out?'),
  ('my', '399', 'neighbourhood',
   'Six neighbourhood nodes in Sungai Petani, Kedah (n10882737017 to n10882737022, 5.663, 100.475) are named only by a house number (399 to 404). Replace by the estate''s name once known, or leave them out?'),
  ('my', 'KAMPUNG BDC (DITINGGALKAN)', 'village',
   'A village node near Sandakan (n14241031890) marked "ditinggalkan", abandoned. Keep, or leave it out of plot names?');
