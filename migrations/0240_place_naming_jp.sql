-- How plots in Japan are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/japan.md and the 127,511
-- places a local import stored (japan-latest.osm.pbf of 2026-11-01),
-- measured on every eighth of the 572,026 land plots (71,504) from the plot
-- centre.
--
-- What the kinds hold. Japanese OSM puts the municipality seats in
-- place=city, town and village (817, 743 and 190 nodes with an English
-- name, about one per municipality), so a rural Japanese village (a 大字,
-- oaza) is not a place=village: the oaza and the machi of the towns are
-- place=quarter
-- (46,692 with an English name, "Hiroo", "Ikaushi", "Takebecho-Nishibara"),
-- and place=neighbourhood holds mostly the chōme, the numbered blocks of a
-- machi (78,791; 58,066 with a number, "Sakyo 2-chome", "Kita-1-jo-Nishi
-- 1-chome"). Some prefectures tag their oaza and machi as neighbourhoods
-- instead: Toyama has 1 quarter and 2,084 neighbourhoods (451 numbered),
-- Niigata 130 and 2,695 (1,103), Kyoto's machi are neighbourhoods (6,692,
-- 567 numbered). place=suburb is mostly the wards of the designated cities
-- ("Kita Ward", "Omiya Ward"), which the district already says better.
--
-- So the quarter is Japan's village, and a neighbourhood stands in where no
-- quarter is near. Every prefecture is urban as a whole, which makes the
-- own-district rule apply everywhere: a plot is named after the nearest
-- quarter of its own municipality within 5 km, else a neighbourhood of it,
-- then the nearest of either outside it, and only then the tiers (a town or
-- village hall, then a city hall, within 5 km). Without the own-district
-- rule the nearest quarter would win also when it lies across the
-- municipal line, and the plot would carry a quarter of one municipality
-- and the name of another; with it, 1.7 % of the plots named after a
-- quarter get one more than 1 km farther than the nearest, which lies in
-- another municipality.
--
-- Measured (share of the sample named after a place):
--   Thailand's rule ({quarter,neighbourhood,suburb} within 3 km of a city
--   node, else village or town, then city, within 5 km)       26.5 %
--   all urban, {quarter} 2 km / 3 km / 4 km / 5 km  52.3 / 61.4 / 67.7 / 72.5 %
--   all urban, {quarter,neighbourhood} 3 km                    65.9 %
--   all urban, {quarter,neighbourhood} 5 km                    78.0 %
-- 5 km as Thailand's village radius: an oaza reaches from its settlement
-- far into the fields and forest around it, and 67 % of the land plots have
-- a quarter within 5 km (53 % within 3 km, median 2.36 km); the others are
-- named after municipality and prefecture. Adding the neighbourhood raises
-- Toyama from 30 % to 69 %, Niigata from 24 % to 47 %, Osaka and Kyoto to
-- 100 %; the price is a numbered block as the name of 2.3 % of the named
-- plots (1,277 of 55,750; 515 of them in Hokkaido, 164 in Osaka), where
-- the own municipality has no quarter within 5 km. Lowest after the rule: Niigata 47 %, Gifu 51 %, Yamagata 54 %, Miyagi 54 %
-- (oaza without an English name: 31,757 quarters and 50,796
-- neighbourhoods have none and are not stored); Tokyo 96 %, Osaka, Kyoto,
-- Hyogo, Okayama, Kagawa, Tottori 100 %.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('jp', '{quarter,neighbourhood}', 5, '{village|town,city}', 5,
   '{aichi-japan,akita-japan,aomori-japan,chiba-japan,ehime-japan,fukui-japan,fukuoka-japan,fukushima-japan,gifu-japan,gunma-japan,hiroshima-japan,hokkaido-japan,hyogo-japan,ibaraki-japan,ishikawa-japan,iwate-japan,kagawa-japan,kagoshima-japan,kanagawa-japan,kochi-japan,kumamoto-japan,kyoto-japan,mie-japan,miyagi-japan,miyazaki-japan,nagano-japan,nagasaki-japan,nara-japan,niigata-japan,oita-japan,okayama-japan,okinawa-japan,osaka-japan,saga-japan,saitama-japan,shiga-japan,shimane-japan,shizuoka-japan,tochigi-japan,tokushima-japan,tokyo-japan,tottori-japan,toyama-japan,wakayama-japan,yamagata-japan,yamaguchi-japan,yamanashi-japan}');

-- Oddities in the imported Japanese place names, to decide on later.
insert into place_overrides (country, name, kind, note) values
  ('jp', '2-chome', 'neighbourhood',
   'Ten neighbourhoods and one quarter are named by their block number alone, "2-chome", "3-chome", "1-chome", "3 chome", without the machi they belong to (the Japanese names are 二丁目, 三丁目, or 桜堤3, 下連雀3, which do name it). As a plot name "2-chome" says nothing. Keep, or replace by the machi and the number?'),
  ('jp', 'Kita-1-jo-Nishi 1-chome', 'neighbourhood',
   '58,066 of the 78,791 neighbourhoods are numbered blocks (chōme), "Kita-1-jo-Nishi 1-chome" (Sapporo), "Chuo 2-chome" (58 times), written four ways ("Sakyo 2-chome", "Midoricho nishi 1chome", "3 chome", "Sekinochō 1"). Where no quarter is within 5 km in the own municipality they name the plot: 2.3 % of the plots named after a place, 40 % of those in Hokkaido. Keep, or name such a plot after the machi without the number?');
