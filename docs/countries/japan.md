# Country scan: Japan

Written by `go run ./cmd/countryscan -country jp` from `japan-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 47 relations, all with ISO 3166-2 JP-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 9 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 7.
- **Districts:** level 7: 1,739 relations, none with an ISO 3166-2 code, covering 99.8 % of the country's land plots (0.01 % overlap, 950 plots in none). Per province: 15 to 179, median 32; 0 districts have their centre in no province.
- **Land plots:** 572,035.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 12 slugs used twice, 27 names whose slug loses letters without accent folding.
- Level 10 has 18,254 relations, more than the 8,000 the scan builds geometry for: counted and read for names only.

## Outline and land

- Outline: the country relation r382313 (admin_level 2); 763,935 km² in 24 polygons.
- Cells at resolution 8 with their centre inside the outline: 1,172,640; 572,035 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **572,035**, about 421,589 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with JP-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 3 | 3 | extract | 0 | 3 | 2 | 0 | – | – | – | – |
| 3 | 2 | 1 | country | 0 | 1 | 0 | 0 | 19.5 % | 80.5 % | 0.00 % | 0 |
| 4 | 50 | 47 | country | 47 | 47 | 0 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 18 | 18 | country | 0 | 18 | 0 | 0 | 19.6 % | 80.4 % | 0.00 % | 0 |
| 6 | 384 | 372 | country | 0 | 372 | 0 | 0 | 40.4 % | 59.6 % | 0.00 % | 0 |
| 7 | 1,742 | 1,739 | country | 0 | 1,739 | 0 | 0 | 99.8 % | 0.2 % | 0.01 % | 0 |
| 8 | 173 | 172 | country | 0 | 172 | 0 | 0 | 3.3 % | 96.7 % | 0.00 % | 0 |
| 9 | 4,541 | 4,398 | country | 0 | 3,016 | 0 | 29 | 2.6 % | 97.4 % | 0.02 % | 0 |
| 10 | 18,254 | 18,254 | extract | 0 | 8,178 | 0 | 20 | – | – | – | – |

### Level 2

Countries in the extract: Japan (日本), Russia (ロシア), South Korea (大韓民国).

### Level 3

- 2 relations in the extract, 1 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1.
- Names: name 1, name:en 1, int_name 0, Latin name 0, any English-capable name 1 (of 1).
- Script of `name`: `Han` 1.
- `name:*` tags: `name:ain` 1, `name:ar` 1, `name:cs` 1, `name:de` 1, `name:en` 1, `name:eo` 1, `name:es` 1, `name:fr` 1.
- Geometry: the relations add up to 139,154 km² (18.2 % of the outline); 460,744 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 111,291 to 111,291, median 111,291.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Hokkaido Region (北海道地方).

### Level 4

- 50 relations in the extract, 47 in scope (country). Left out: 3 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `JP` 47; with this country's prefix: 47 of 47.
- Names: name 47, name:en 47, int_name 0, Latin name 0, any English-capable name 47 (of 47).
- Script of `name`: `Han` 47.
- `name:*` tags: `name:cs` 47, `name:de` 47, `name:en` 47, `name:es` 47, `name:fr` 47, `name:ja-Hira` 47, `name:ja-Latn` 47, `name:ko` 47.
- Geometry: the relations add up to 763,935 km² (100.0 % of the outline); 9 land plots lie in none of them, 8 in two or more; 0 relations hold no land plot; land plots per relation: 3,099 to 111,291, median 9,414.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Aichi Prefecture (愛知県), Fukui Prefecture (福井県), Hiroshima Prefecture (広島県), Iwate Prefecture (岩手県), Kumamoto Prefecture (熊本県), Nagano Prefecture (長野県), Okayama Prefecture (岡山県), Shiga Prefecture (滋賀県), Tokyo (東京), Yamanashi Prefecture (山梨県).

### Level 5

- 18 relations in the extract, 18 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 18.
- Names: name 18, name:en 18, int_name 0, Latin name 0, any English-capable name 18 (of 18).
- Script of `name`: `Han` 17, `Kana` 1.
- `name:*` tags: `name:en` 18, `name:ja` 18, `name:ja-Hira` 18, `name:ja-Latn` 18, `name:ko` 18, `name:ru` 18, `name:zh` 18, `name:zh-Hans` 18.
- Geometry: the relations add up to 179,532 km² (23.5 % of the outline); 460,131 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 111 to 15,165, median 5,740.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Hachijō Subprefecture (八丈支庁), Hidaka Subprefecture (日高振興局), Iburi Subprefecture (胆振総合振興局), Kamikawa Subprefecture (上川総合振興局), Miyake Subprefecture (三宅支庁), Ogasawara Subprefecture (小笠原支庁), Oshima Subprefecture (大島支庁), Rumoi Subprefecture (留萌振興局), Sorachi Subprefecture (空知総合振興局), Tokachi Subprefecture (十勝総合振興局).

### Level 6

- 384 relations in the extract, 372 in scope (country). Left out: 0 with another country's ISO 3166-2, 12 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 372.
- Names: name 372, name:en 372, int_name 0, Latin name 0, any English-capable name 372 (of 372).
- Script of `name`: `Han` 372.
- `name:*` tags: `name:en` 372, `name:ja` 372, `name:ja-Hira` 372, `name:ja-Latn` 372, `name:zh` 372, `name:zh-Hans` 372, `name:zh-Hant` 372, `name:ko` 371.
- Geometry: the relations add up to 299,509 km² (39.2 % of the outline); 341,140 land plots lie in none of them, 5 in two or more; 0 relations hold no land plot; land plots per relation: 6 to 5,512, median 389.
- Rings not closed in the extract (a member way is missing): 0 in scope; 9 of the level's relations could not be assembled at all.
- Examples: Abashiri County (網走郡), Futaba County (双葉郡), Inukami County (犬上郡), Kanoashi County (鹿足郡), Kitatsugaru County (北津軽郡), Minamiuonuma County (南魚沼郡), Nishimuro County (西牟婁郡), Senboku County (仙北郡), Tamura County (田村郡), Yuki County (結城郡).

### Level 7

- 1,742 relations in the extract, 1,739 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1,739.
- Names: name 1,739, name:en 1,739, int_name 0, Latin name 0, any English-capable name 1,739 (of 1,739).
- Script of `name`: `Han` 1,666, `Kana` 73.
- `name:*` tags: `name:en` 1,739, `name:ja` 1,739, `name:ja-Hira` 1,739, `name:zh` 1,739, `name:zh-Hant` 1,739, `name:ja-Latn` 1,738, `name:ko` 1,738, `name:zh-Hans` 1,738.
- Geometry: the relations add up to 746,141 km² (97.7 % of the outline); 950 land plots lie in none of them, 85 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 3,377, median 190.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Abashiri (網走市), Furano (富良野市), Ikusaka (生坂村), Kawamoto (川本町), Matsuno (松野町), Nakatsu (中津市), Otoyo (大豊町), Shobara (庄原市), Toyooka (豊丘村), Ōtake (大竹市).

### Level 8

- 173 relations in the extract, 172 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 172.
- Names: name 172, name:en 172, int_name 0, Latin name 0, any English-capable name 172 (of 172).
- Script of `name`: `Han` 171, `Kana` 1.
- `name:*` tags: `name:en` 172, `name:ja-Hira` 171, `name:ja` 148, `name:ja-Latn` 118, `name:ru` 64, `name:fr` 60, `name:ko` 58, `name:ja_rm` 54.
- Geometry: the relations add up to 19,442 km² (2.5 % of the outline); 553,174 land plots lie in none of them, 5 in two or more; 0 relations hold no land plot; land plots per relation: 3 to 1,639, median 47.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Abeno Ward (阿倍野区), Chuo Ward (中央区), Higashinari Ward (東成区), Kita Ward (北区), Midori Ward (緑区), Minato Ward (港区), Naniwa Ward (浪速区), Sakae Ward (栄区), Teine Ward (手稲区), Ōmiya Ward (大宮区).

### Level 9

- 4,541 relations in the extract, 4,398 in scope (country). Left out: 0 with another country's ISO 3166-2, 123 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 4,398.
- Names: name 4,339, name:en 3,016, int_name 0, Latin name 29, any English-capable name 3,158 (of 4,398).
- Script of `name`: `Han` 4,082, `Kana` 228, `Latin` 29.
- `name:*` tags: `name:ja` 3,415, `name:en` 3,016, `name:ja-Hira` 2,879, `name:ja-Latn` 1,020, `name:ja_rm` 782, `name:fr` 362, `name:zh` 328, `name:zh-Hant` 326.
- Geometry: the relations add up to 15,747 km² (2.1 % of the outline); 557,139 land plots lie in none of them, 100 in two or more; 1379 relations hold no land plot; land plots per relation: 0 to 617, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 123 of the level's relations could not be assembled at all.
- Examples: Abenosuji (阿倍野筋), Hayashi Chō (林町), Kami-Hayashi Chō (上林町), Matsunami Chō (松並町), Nishi-Une (西畦), Shimoiino (下飯野), Tsumada-nishi (妻田西), 厚別町上野幌, 東早稲田, 龍ノ口.

### Level 10

- 18,254 relations in the extract, 18,254 in scope (extract). Left out: 0 with another country's ISO 3166-2.
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 18,254.
- Names: name 18,254, name:en 8,178, int_name 0, Latin name 20, any English-capable name 8,207 (of 18,254).
- Script of `name`: `Han` 17,219, `Kana` 1,015, `Latin` 20.
- `name:*` tags: `name:ja` 8,787, `name:en` 8,178, `name:ja-Hira` 4,488, `name:ja-Latn` 2,055, `name:fr` 948, `name:zh` 565, `name:zh-Hant` 554, `name:zh-Hans` 553.
- Examples: 1-chome (一丁目), Higashi-Kaminokura 3-chome (東神の倉三丁目), Kurosawa 3 (黒沢三丁目), Omori Kita 1 (大森北一丁目), Tsunashima-kamicho (綱島上町), 北19条西8, 南9条西20, 平野西五丁目, 田島六丁目, 龍造寺町.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 47 relations, all with ISO 3166-2 JP-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 9 plots in none).
- Districts: level 7: 1,739 relations, none with an ISO 3166-2 code, covering 99.8 % of the country's land plots (0.01 % overlap, 950 plots in none).

Levels that qualify: 4, 7. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:ja-Latn`, `name` (when written in Latin letters). Local name: `name:ja`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 47 relations)

- English name from: `name:en` 47; none for 0.
- Local name from: `name:ja` 46, `name` 1; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “Prefecture” 46.
- Local prefixes: none. Suffixes: “県” 43.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 7, 1,739 relations)

- English name from: `name:en` 1,739; none for 0.
- Local name from: `name:ja` 1,739; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: “市” 790, “町” 742.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 12 slugs are used more than once: 11/misato (2), 25/fuchu (2), 29/kiso (2), 29/matsukawa (2), 3/echizen (2), 41/shimanto (2), 41/tosa (2), 45/esashi (2), 45/kushiro (2), 45/shibetsu (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** a slug without folding (as before stage 2) drops accented letters; `osmfile.Slug` folds them (NFD, đ → d), which changes the slug of 27 names, e.g. Kyōtango: ky-tango → kyotango, Higashiōsaka: higashi-saka → higashiosaka, Ōmihachiman: mihachiman → omihachiman.

The slug of a new country's province carries the country as a suffix (`-japan`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 220,693. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| neighbourhood | 129,587 | 129,582 | 78,493 |
| quarter | 78,449 | 78,444 | 46,641 |
| hamlet | 5,883 | 5,882 | 1,009 |
| locality | 2,884 | 2,845 | 747 |
| city | 817 | 817 | 817 |
| town | 743 | 743 | 743 |
| suburb | 370 | 356 | 278 |
| county | 356 | 356 | 356 |
| islet | 305 | 302 | 248 |
| city_block | 283 | 283 | 4 |
| square | 208 | 201 | 51 |
| village | 195 | 195 | 190 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Tokyo (population 13,613,660): neighbourhood 1,190 (1,190 English-capable), quarter 244 (244 English-capable), city 12 (12 English-capable), locality 6 (6 English-capable), square 5 (4 English-capable), city_block 4 (4 English-capable).
- Yokohama (population 3,777,491): neighbourhood 906 (906 English-capable), quarter 33 (30 English-capable), suburb 9 (9 English-capable), locality 6 (3 English-capable), square 3 (0 English-capable).
- Osaka (population 2,691,185): neighbourhood 1,916 (1,916 English-capable), square 23 (10 English-capable), suburb 20 (20 English-capable), locality 5 (3 English-capable), city 2 (2 English-capable), quarter 1 (1 English-capable).
- Nagoya (population 2,236,000): neighbourhood 2,363 (2,181 English-capable), quarter 866 (631 English-capable), suburb 11 (11 English-capable), hamlet 7 (2 English-capable), city 2 (2 English-capable), locality 2 (2 English-capable).
- Sapporo (population 1,973,395): neighbourhood 4,437 (4,437 English-capable), quarter 43 (43 English-capable), suburb 7 (7 English-capable), isolated_dwelling 2 (0 English-capable), hamlet 1 (1 English-capable).

## Size

- Land plots: 572,035. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 23 s and 525 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Japan: from japan-latest.osm.pbf
{Code: "jp", Extract: "japan",
	Province: Rule{AdminLevel: 4, ISOPrefix: "JP", Count: 47, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 7},
	EnglishName: []string{"name:en", "int_name", "name:ja-Latn", "name"}, LocalLang: "ja",
	StripProvince: Strip{English: [2][]string{nil, []string{"Prefecture"}}, Local: [2][]string{nil, []string{"県"}}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('jp', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
