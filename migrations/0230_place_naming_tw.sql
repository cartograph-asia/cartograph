-- How plots in Taiwan are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/taiwan.md and the 1,953
-- places a local import stored (taiwan-latest.osm.pbf of 2026-11-01; 1,272
-- villages, 267 suburbs, 245 towns, 145 neighbourhoods, 15 quarters, 9
-- cities), measured over the 55,532 land plots from the plot centre and,
-- through the naming code itself, on every second plot (27,766).
--
-- What the kinds hold. place=town is the seat of a township or district
-- (245 nodes, one per district almost; 140 carry the word in English:
-- "Baozhong Township", "Daxi District", "Magong City"), place=village the
-- old villages and settlements (1,272 English-named of 1,958), place=suburb
-- in Taipei the old neighbourhoods (Tianmu, Muzha, Jingmei, Shipai) and the
-- districts, in Taichung only its 29 district nodes ("Beitun District"),
-- place=neighbourhood in Kaohsiung and Tainan old villages and military
-- dependents' villages ("Huangpu New Village"), place=quarter a few
-- commercial zones. The 9 city nodes are the six special municipalities,
-- Keelung, Hsinchu and Chiayi.
--
-- Inside a city. Only 1,032 land plots (1.9 %) have a city node within
-- 5 km. For them a suburb lies within 3 km for 77.6 % (2 km: 59.0 %;
-- median spacing of suburbs near the cities 1.35 km), quarters and
-- neighbourhoods add 0 to 0.4 points; so Thailand's urban kinds and radius.
-- In the sample they name 412 plots (338 suburbs, 41 quarters, 33
-- neighbourhoods); 174 of them are named after their own district's node
-- (Taichung), the rest after Taipei's and Kaohsiung's neighbourhoods. Taking
-- suburbs out (quarter and neighbourhood within 2 km) removes those 174 and
-- the 164 good names with them; the share of named plots is 64.8 % either
-- way. No province is urban as a
-- whole: Taipei's plots are all named through the plain rule, and the other
-- five municipalities are mostly mountains and farmland.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. 64.6 % of the land plots have a village or town
-- within 5 km (3 km: 45.7 %, 8 km: 78.5 %; Thailand 64 %, Laos 54 %; median
-- 2.14 km), from 99 % in Changhua, Keelung, Kinmen and Taipei to 40 % in
-- Hualien and 42 % in Taichung, whose mountains hold no named village; the
-- others are named after their district and province. An 8 km radius would
-- name 78.7 % of the sample. Villages before towns (two tiers) would halve
-- the plots named after their own district's seat (12.4 % → 4.9 %) at the
-- price of naming 2,362 plots in the sample after a village farther away
-- than the town they lie in, so village and town stay one tier.
--
-- Hamlets are not read (14,034 nodes, 12,544 named in English, Taiwan's
-- most frequent place kind; adding them is open for Adrian, and this
-- migration leaves places_kind_check as it is): with them 84.0 % of the land
-- plots would have a place within 5 km, at a median of 0.87 km.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('tw', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5, '{}');

-- Oddities in the imported Taiwanese place names, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('tw', 'Jinsha Town', 'town',
   '140 of the 245 towns (the seats of the townships and districts), 67 suburbs, 4 villages and 3 quarters carry the administrative word in English: "Jinsha Town", "Baozhong Township", "Daxi District", "Magong City". The district names drop it, so a plot reads "Jinsha Town, Jinsha · Kinmen"; 12.4 % of the plots of the sample are named after their own district''s seat or node. Strip the word from place names, or keep as OSM has them?'),
  ('tw', 'Beitun District', 'suburb',
   'Taichung''s 29 suburbs are its district nodes ("Beitun District", "Central District"), so its urban plots are named after their own district ("Beitun District, Beitun · Taichung"). Drop such nodes from naming, or keep?'),
  ('tw', 'Hái-bué / Haiwei', 'neighbourhood',
   '26 places carry two romanisations, one of them Taiwanese Hokkien with tone marks: "Hái-bué / Haiwei", "Tiàm-á-tíng / Dianzihding", "Katatipulr / Zhiben" (Puyuma). Keep as OSM has them, or keep the Mandarin form only?'),
  ('tw', 'He Ch''un New Village', 'neighbourhood',
   'Three military dependents'' villages are written in Wade-Giles ("He Ch''un New Village", "Ch''ang Tsao New Village", "Tz''u Hui 9th Village"), and 合群新村 is mapped twice, as "He Ch''un New Village" and "Hequn New Village". Keep, or write them in Hanyu Pinyin like the rest?');
