-- How plots in South Korea are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md,
-- WP H), chosen from the Places section of docs/countries/south-korea.md and
-- the 20,415 places a local import stored (south-korea-latest.osm.pbf of
-- 2026-11-01), measured over the 191,457 land plots from the plot centre.
--
-- What the kinds hold. Korean OSM puts the neighbourhoods of a city,
-- the dong, in place=quarter (3,554 nodes, 3,431 of them "…-dong", 399 "…-ga"
-- street blocks of the old dong), the villages, the ri, in place=village
-- (15,106, 15,020 "…-ri") and the township seats, eup and myeon, in
-- place=town (1,408, 916 with the word). place=neighbourhood is a mixture
-- (234 nodes; 159 have digits: "24-tong", "Jingwan 2-ri", "Jongno 2-ga"),
-- place=suburb has 28 nodes.
--
-- Inside a city. 11,921 land plots (6.2 %) have a place=city within 5 km.
-- For them a quarter lies within 3 km for 90.9 % (2 km: 79.9 %, 5 km:
-- 98.1 %; median 0.89 km), a village or town for 57.0 %. Adding
-- neighbourhood to the urban kinds gains 0.6 points ({quarter,neighbourhood}
-- 91.5 %) at the price of the numbered tong, suburb 0.1 more; so the urban
-- kind is the quarter alone, within 3 km as in Bangkok.
--
-- Urban provinces. A city node covers a circle of 5 km, which is little for
-- the big cities: only 30 % of Seoul's plots have one (each of the six has
-- exactly one place=city node), while 99 % have a quarter within 3 km. Named
-- through the plain rule, a sample of every second plot (95,729 in all) found
-- a place for 41 % of Seoul's plots, 60 % of Busan's, 71 % of Incheon's, 81 %
-- of Daejeon's, 88 % of Ulsan's and 97 % of Daegu's (77 % of the six
-- together), the others falling back to "Gangnam, Seoul 123" without a
-- place. So the districts of the six cities that are governed as provinces
-- are urban as a whole: Seoul, Busan, Daegu, Incheon, Daejeon, Ulsan; the
-- sample then finds a place for Seoul's, Daegu's, Daejeon's and Ulsan's 100 %
-- of plots, Busan's 99 % and Incheon's 95 % (98.8 % of the six together). A few of their districts are farmland (Gijang,
-- Dalseong, Gunwi, Ulju, Ganghwa, Ongjin); there a village still names the
-- plot where no quarter is within 3 km, and 6 % of those plots (151 of
-- 2,504) are named after a quarter instead of a village (2 before), at about
-- the same median distance (1.02 km against 0.98 km). Sejong has no
-- districts, and the old Gwangju lies in the province Jeonnam-Gwangju with
-- 22 rural districts, so neither is listed.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. Korea's OSM is dense (10.7 places per 100
-- plots; Thailand 3.3): 95.3 % of the land plots have a village or town
-- within 5 km (3 km: 88.9 %, 8 km: 97.9 %; Thailand 64 %, Laos 54 %), at a
-- median distance of 1.10 km; through the whole naming 98.1 % of the sample
-- is named after a place (Jeju 83 %, where the volcano's slopes hold no
-- village, the others 95 to 100 %).
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('kr', '{quarter}', 3, '{village|town,city}', 5,
   '{seoul-south-korea,busan-south-korea,daegu-south-korea,incheon-south-korea,daejeon-south-korea,ulsan-south-korea}');

-- Oddities in the imported Korean place names, to decide on later (nothing
-- reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('kr', 'Sinheung-dong 1(il)-ga', 'quarter',
   '399 of the 3,554 quarters are the street blocks of an old dong, "Sinheung-dong 1(il)-ga", "Anam-dong 5-ga", "Chungmuro 5-ga", and 46 more carry a number ("Daejeo 2(i)-dong", "Dongtan 1 New Town"), so a plot in the old centre of Incheon or Seoul is named after a block address. Keep, or name it after the dong without the number?'),
  ('kr', 'Dong', 'town',
   '492 of the 1,408 towns (the seats of eup and myeon) are written without the word: "Muan" (무안읍 and 무안면), "Gaya", and the four called plain "Dong" (동면, 동읍), the same as the Dong district of a metropolitan city. 916 have it ("Cheongyang-eup", "Samseung-myeon"). Keep as OSM has them, or write the word where the Korean name has 읍/면?'),
  ('kr', 'Gujiri', 'village',
   '86 of the 15,106 villages are not spelled "…-ri": a bare name ("Okgwa", Korean 옥과리), a fused one ("Gujiri", "Jangdongni", "Bokcheolli") or two with a capital ("Daman-Ri"), against 15,020 "…-ri". Keep them, or write them "…-ri"?'),
  ('kr', 'Dogok-ri - DITL', 'village',
   'One village name carries a mapping-project tag, " - DITL" (the Korean name is 도곡리). Replace it by "Dogok-ri"?');
