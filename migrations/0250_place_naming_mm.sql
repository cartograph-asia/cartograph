-- How plots in Myanmar are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md,
-- WP H), chosen from the Places section of docs/countries/myanmar.md and the
-- 56,156 places a local import stored (myanmar-latest.osm.pbf of
-- 2026-11-01), measured over the 803,321 land plots from the plot centre.
--
-- What the kinds hold. Burmese OSM puts the wards of a town (ရပ်ကွက်) in
-- place=quarter: 2,051 nodes, 1,904 of them "… Ward" ("Myoma (1) Ward",
-- "Kyar Kan Ward"), 676 numbered only ("No. (5) Ward", "Ward (3)"), 708 of
-- them in Yangon Region. The villages are place=village (53,437) and the
-- towns place=town (505). place=suburb has 55 nodes, in Yangon and
-- Mandalay the townships themselves ("Bahan", "Kamayut", "Chanayethazan"),
-- which the plot name carries as its district anyway; place=neighbourhood
-- has 45, a mixture of housing estates, numbered blocks and refugee camps
-- ("May Kha Housing", "Block 10", "Camp 9"). Both stay out.
--
-- Inside a city. 5,455 land plots (0.7 %) have a place=city within 5 km
-- (62 city nodes, most of them small state and region capitals). For them a
-- ward lies within 2 km for 26.6 % (3 km 39.0 %), a village or town for
-- 68.2 % (3 km 85.7 %): the wards are mapped in Yangon, Mandalay and a few
-- towns, the villages almost everywhere. Near the city nodes a ward is
-- 0.48 km from the next one at the median (Yangon 0.32 km, Mandalay
-- 0.62 km, Myitkyina 0.66 km, Mawlamyine 0.77 km), so 2 km, not Bangkok's
-- 3 km, as in Laos: with wards half a kilometre apart, 3 km would prefer a
-- ward of the own township far off to one across the township line nearby.
-- Of the urban plots 496 are named after a numbered ward with 2 km (713
-- with 3 km; 68 by the plain tiers), unique within the township that the
-- plot name carries.
--
-- No province is urban as a whole: Yangon Region (11,294 plots) is mostly
-- farmland and delta (Twante, Kungyangon, Kawhmu, Taikkyi, Htantabin,
-- Cocokyun), Mandalay Region and Naypyitaw too. So Yangon's outer townships,
-- 5 to 15 km from the city node (694 plots), are named by the plain tiers: a
-- village or town within 2 km for 57.3 %, while a ward would be nearer for
-- 48.4 % (median 1.06 km against 1.58 km). In Mandalay's ring (751 plots)
-- villages are nearer (median 0.97 km against 3.72 km for a ward). Naming
-- Yangon's city townships by their wards would need urban districts, which
-- urban_regions (provinces only) cannot express.
--
-- Everywhere else Thailand's tiers and radius: the nearest village or town
-- within 5 km, else a city. 69.0 % of Myanmar's land plots have a village or
-- town within 5 km (3 km 52.0 %, 8 km 81.5 %; median 2.19 km; Thailand 64 %,
-- Laos 54 %); the others are named after township and state or region,
-- most of them in the forested hills (a place within 5 km: Kachin 38.9 %,
-- Tanintharyi 50.2 %, Kayah 57.9 %, Rakhine 59.6 %; Yangon 92.1 %,
-- Ayeyarwady and Magway 88.9 %). 1,234 hamlets stay out as in Thailand.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('mm', '{quarter}', 2, '{village|town,city}', 5, '{}');

-- Oddities in the imported Burmese place names, to decide on later (nothing
-- reads place_overrides yet).
insert into place_overrides (country, name, kind, note) values
  ('mm', 'No. (5) Ward', 'quarter',
   'One of 676 of the 2,051 quarters (wards) named by a number only: "No. (5) Ward", "Ward (3)", "No (112) Ward"; 1,904 end in " Ward". They are unique only within their township, which the plot name carries ("No. (5) Ward, Ahlone · Yangon 12"). Keep as OSM has them, drop " Ward", or prefer a named place a little further away?'),
  ('mm', '(1) Hlaw Taw Chaung, (2) Ga Nan Aw, (3) Taung Pyo, (4) Kyar Kaik Chaung', 'village',
   'A village node (n7429963471) that lists four villages in one name, in English and Burmese. Also with a list or a note in the name: "Aung Kyin Mying ( Mrokhami, Burma, Rakhine)" (n7796301175) and "(Wet Toe + Pan Taung)" (n7416656975). Keep, or replace by the first village name?'),
  ('mm', 'Taung Nyo', 'village',
   'One of 9 places whose Burmese name is not in Unicode order: 6 are Zawgyi-encoded (ေတာင္ညို for တောင်ညို; also Nat Maw Village, Taungpyone Nge, Lann Sone Gyi, Pho Win Taung and the second half of Zee Kone (South)), 3 in Kachin start with a zero-width non-joiner and the vowel ေ (Kawng San, Auk Sa Bway, Lawt Jar). They render wrongly with a Unicode font. Of about 59,000 Burmese names in the extract no other is Zawgyi, and none of a province or township. Convert to Unicode in the name_local, or leave as OSM has them until fixed there?');
