-- How plots in Indonesia are named (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, WP H),
-- chosen from the Places section of docs/countries/indonesia.md, the 82,196
-- places and the 2,423,954 land plots a local import stored
-- (indonesia-latest.osm.pbf of 2026-11-01) and Thailand's row in
-- 0075_place_naming.sql. Measured on every land plot, with the nearest place
-- of each kind; the own-district preference inside cities was not modelled.
--
-- What the place nodes are. The unit below the kecamatan, the desa
-- (village) or kelurahan (urban village), is what every address ends in.
-- OSM maps it as place=village in the countryside and, depending on the
-- city, as village (Jakarta 82 within 5 km of its city node, Surabaya 64,
-- Palembang 37) or as suburb (Semarang 88, Bandung 93, Medan 89, Makassar
-- 107); the 1,699 suburbs lie a median 0.64 km apart near cities. Below it,
-- place=neighbourhood holds lingkungan, kampung and housing estates ("Lingk.
-- Batang Hari", "Kmp. Gallu Dara", "Perumahan BTN Pagutan Permai", median
-- 0.28 km apart near cities), and place=quarter (200) mixes kampung and
-- estates; neither names a plot. The district level is the regency or city
-- (kabupaten, kota), so the plot name carries the city anyway.
--
-- Inside a city (a place=city within radius_km: 25,943 land plots, 1.1 %)
-- the suburb, where a city maps its kelurahan so, within 2 km and of the
-- plot's own regency or city first; elsewhere the tiers find the kelurahan
-- as a village. Counted on the urban plots (nearest only, without the
-- own-district rule):
--   {suburb} 2 km                         suburb 2,288, village 22,893;
--                                         180 a suburb with a village or
--                                         town more than 1 km nearer
--   {quarter,neighbourhood,suburb} 3 km   neighbourhood 2,712, quarter 506,
--   (Thailand)                            suburb 1,639; 1,493 so
--   {neighbourhood,quarter,suburb} 3 km   neighbourhood 3,024; 1,467 so
--   (Malaysia)
--   {suburb,quarter} 3 km                 700 so
-- 2 km covers the suburbs' spacing (nine tenths are less than 1.26 km
-- apart). No province is urban as a whole: Jakarta's five cities are, but
-- its Thousand Islands regency is not, and Jakarta's kelurahan are villages,
-- which the tiers find anyway.
--
-- Everywhere else the nearest village, suburb or town within 5 km, else a
-- city: the suburb joins the first tier because outside the big cities it
-- is the kelurahan of a town (8,102 land plots take one, 6,412 of them
-- outside the urban plots, instead of a village further off). 52.2 % of the
-- land plots have a village or town within 5 km, at a median of 2.16 km
-- (Thailand 64 %, Laos 54 %, Vietnam 51.5 %, Malaysia 32 %; 3 km 35.7 %,
-- 8 km 67.3 % at a median of 2.80 km and a 90th percentile of 6.47 km); the
-- others, Kalimantan's and Papua's forests and mountains, are named after
-- their regency and province. 27,345 hamlets (dusun) stay out as in
-- Thailand.
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions) values
  ('id', '{suburb}', 2, '{village|suburb|town,city}', 5, '{}');

-- Oddities in the imported Indonesian place names, to decide on later
-- (nothing reads place_overrides yet). Only villages, suburbs, towns and
-- cities are counted, the kinds that name plots.
insert into place_overrides (country, name, kind, note) values
  ('id', 'Kel. Wakumoro', 'village',
   'One of 241 villages, suburbs and towns whose name starts with the word for the unit: Kel./Kelurahan 124 (Kelurahan Sarae), Desa/Ds. 107 (Desa Lalole), Kecamatan 10 (Kecamatan Belo, a town node); the others carry the bare name. But "Desa Baru" (new village, four of them) is a name, not a word before one. Strip the word where it is one, or keep?'),
  ('id', 'LEMBANG MAKKODO', 'village',
   'One of 45 village, suburb and town names written in capitals (LAYA, BONDE, PASIRMULYA, DESA POCO DEDENG, KECAMATAN PEKAT). Keep, or write them in title case?'),
  ('id', 'Sido Mukti (sp1,3)', 'village',
   'One of 18 villages named with the code of their transmigration settlement units in brackets (sp = satuan pemukiman: Bangun Harjo (sp6,8), Air Putih Ulu (c1,2)), and five plantation divisions named "Afdeling II Rantau Prapat" and the like. Keep, or drop the code?'),
  ('id', '..............', 'village',
   'A place=village node in West Nusa Tenggara (n11290770163, -8.572, 118.698, near Bima) whose name is fourteen dots. Leave it out of plot names, or replace by its real name once known?');
