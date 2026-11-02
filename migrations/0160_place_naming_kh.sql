-- How plots in Cambodia are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/cambodia.md, the imported
-- places (1,124 for Cambodia, 61 of them Vietnamese villages beyond the
-- border) and the measurements below, taken over the 213,166 plots of the
-- imported provinces with the distance from the plot centre.
--
-- Inside a city. Phnom Penh is mapped as Bangkok is: its 14 khan are
-- place=town nodes ("Khan Dangkao") and its sangkats, the quarters, are
-- place=village nodes ("Sangkat Wat Phnom"): 103 villages and 14 towns for
-- the 805 plots of the province, while it holds only 42 neighbourhood,
-- suburb and quarter nodes, which are mostly housing communities in the west
-- ("Community (DK12)", "DK 17") and say little about the centre. Measured
-- over Phnom Penh's plots, an urban kind found within 3 km:
--   neighbourhood, suburb, quarter   44 % of the plots (median 3.5 km),
--   village, town                    93 % (median 1.4 km; within 2 km: 76 %).
-- The sangkat nodes lie 1.3 km apart (median; 90th percentile 3.0 km), so
-- the village is the urban kind, then the town, and 3 km as in Bangkok the
-- radius. A plot is named after the nearest sangkat of its own khan, else
-- the nearest one anywhere within 3 km, else the khan's town node. Every
-- plot of Phnom Penh lies within reach of a sangkat and no khan is
-- farmland, so the province is urban as a whole, as Bangkok is.
-- The 27 other place=city nodes make the plots within 5 km of them urban
-- (2,184 plots outside Phnom Penh). There villages are rare, and the
-- provincial cities are divided into suburbs and quarters (Sihanoukville's
-- four sangkats are place=suburb "Quarter 1" to "Quarter 4", Svay Rieng's
-- and Kep's likewise): a village or town is found within 3 km
-- for 25 % of those plots, with suburb and quarter after them for 33 % (and
-- 58 % within 5 km, against 48 %). So the urban kinds are village, town,
-- suburb, quarter, in that order, which changes nothing in Phnom Penh (93 %
-- either way). Neighbourhood stays out: outside Phnom Penh it is mostly a
-- commune ("Khum Rovieng"), 14 of its 26 nodes outside Phnom Penh.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. Cambodia's OSM holds few places (about 0.5 per
-- 100 plots; Thailand 3.3), so 21.5 % of its plots have a village or town
-- within 5 km (8 km would give 37.9 %, 10 km 47.2 %; Thailand 64 % at 5 km,
-- Laos 54 %). The others are named after their district and province, or
-- after the province and the plot number alone where no district covers the
-- land (about 7 % of the plots). A wider radius would name a plot of flat
-- rice land after a village that is out of sight, so Thailand's 5 km stays;
-- Adrian may want 8 km, which changes only this row.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('kh', '{village,town,suburb,quarter}', 3, '{village|town,city}', 5, '{phnom-penh-cambodia}');

-- Oddities in the imported Cambodian place names, to decide on later
-- (nothing reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('kh', 'Sangkat Roluoh', 'village',
   'OSM names Phnom Penh''s sangkats (quarters) as place=village with the word "Sangkat" in front: 124 places are called "Sangkat …" (118 villages, 3 quarters, 2 neighbourhoods, 1 suburb; 103 of them in Phnom Penh), so a plot there is named "Sangkat Roluoh, Dangkao · Phnom Penh 12". Keep the word, or strip it like the khan word below?'),
  ('kh', 'Khan Dangkao', 'town',
   'The 14 khans of Phnom Penh are place=town nodes named "Khan …", and 9 district capitals elsewhere are place=town named "Srok …"; the districts themselves are imported without the word (Dangkao). Keep, or strip "Khan" and "Srok"?'),
  ('kh', 'Khum Rovieng', 'neighbourhood',
   '194 places are called "Khum …" (khum, commune): 178 villages, 14 neighbourhoods and 2 towns, 88 of them in Battambang and 54 in Bantey Meanchey. They name an administrative unit, not a settlement; keep, or strip "Khum"?'),
  ('kh', 'Preah Vihear Municipality', 'city',
   'The only place=city whose English name carries the word "Municipality"; the others are plain (Battambang, Kampot).'),
  ('kh', 'Quarter 1', 'suburb',
   'Sihanoukville''s four sangkats are place=suburb named "Quarter 1" to "Quarter 4" (Khmer: សង្កាត់មួយ …), so a plot there can be named "Quarter 2, Preah Sihanouk · Preah Sihanouk 5" where no village is nearer. Keep, or write "Sangkat 2" as elsewhere?'),
  ('kh', 'Community (DK12)', 'neighbourhood',
   'In western Phnom Penh 37 place=neighbourhood nodes are housing communities named after road codes ("Community (DK12)", "DK 17", "2 DK54") or "… Community"; neighbourhoods are not an urban kind here (the village is), so they name no plot inside the city. Keep as they are, or drop?');
