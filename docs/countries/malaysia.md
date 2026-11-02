# Country scan: Malaysia

Written by `go run ./cmd/countryscan -country my` from `malaysia-singapore-brunei-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Boundaries completed with the ways and nodes of the neighbours' cuts `philippines-latest.osm.pbf.cut.osm.pbf` (relations and places from the own file only). Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 16 relations, all with ISO 3166-2 MY-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 13 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 157 relations, none with an ISO 3166-2 code, covering 98.0 % of the country's land plots (0.00 % overlap, 8,582 plots in none). Per province: 0 to 43, median 9; 0 districts have their centre in no province.
- **Land plots:** 419,805.

## Outline and land

- Outline: the country relation r2108121 (admin_level 2); 506,654 km² in 3 polygons.
- Cells at resolution 8 with their centre inside the outline: 635,605; 419,805 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **419,805**, about 309,396 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with MY-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 0 | 4 | – | – | – | – |
| 3 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 32 | 16 | country | 16 | 11 | 0 | 16 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 37 | 19 | country | 0 | 18 | 0 | 19 | 63.6 % | 36.4 % | 0.00 % | 0 |
| 6 | 284 | 157 | country | 0 | 28 | 4 | 151 | 98.0 % | 2.0 % | 0.00 % | 0 |
| 7 | 356 | 22 | country | 0 | 9 | 0 | 22 | 4.2 % | 95.8 % | 0.00 % | 0 |
| 8 | 86 | 31 | country | 0 | 5 | 0 | 31 | 0.4 % | 99.6 % | 0.00 % | 0 |
| 10 | 380 | 141 | country | 0 | 108 | 0 | 141 | 0.1 % | 99.9 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Brunei, Indonesia, Malaysia, Philippines (Filipina), Singapore (Singapura), Thailand.

### Level 3

- 3 relations in the extract, 0 in scope (country). Left out: 2 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 32 relations in the extract, 16 in scope (country). Left out: 16 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `MY` 16; with this country's prefix: 16 of 16.
- Names: name 16, name:en 11, int_name 0, Latin name 16, any English-capable name 16 (of 16).
- Script of `name`: `Latin` 16.
- `name:*` tags: `name:ar` 16, `name:bg` 16, `name:eo` 16, `name:fa` 16, `name:he` 16, `name:hi` 16, `name:ja` 16, `name:ko` 16.
- Geometry: the relations add up to 440,412 km² (86.9 % of the outline); 13 land plots lie in none of them, 3 in two or more; 0 relations hold no land plot; land plots per relation: 59 to 164,333, median 10,847.
- Rings not closed in the extract (a member way is missing): 0 in scope; 10 of the level's relations could not be assembled at all.
- Examples: Johor, Kedah, Kuala Lumpur, Malacca (Melaka), Negeri Sembilan, Penang (Pulau Pinang), Perlis, Putrajaya, Sarawak, Terengganu.

### Level 5

- 37 relations in the extract, 19 in scope (country). Left out: 0 with another country's ISO 3166-2, 18 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 19.
- Names: name 19, name:en 18, int_name 0, Latin name 19, any English-capable name 19 (of 19).
- Script of `name`: `Latin` 19.
- `name:*` tags: `name:en` 18, `name:ar` 15, `name:ms` 15, `name:zh` 14, `name:id` 1, `name:ko` 1, `name:ms-Arab` 1, `name:th` 1.
- Geometry: the relations add up to 202,156 km² (39.9 % of the outline); 152,826 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 357 to 51,952, median 8,729.
- Rings not closed in the extract (a member way is missing): 0 in scope; 12 of the level's relations could not be assembled at all.
- Examples: Betong Division (Bahagian Betong), George Town, Kapit Division (Bahagian Kapit), Kudat Division (Bahagian Kudat), Miri Division (Bahagian Miri), Samarahan Division (Bahagian Samarahan), Sarikei Division (Bahagian Sarikei), Serian Division (Bahagian Serian), Sri Aman Division (Bahagian Sri Aman), West Coast Division (Bahagian Pantai Barat).

### Level 6

- 284 relations in the extract, 157 in scope (country). Left out: 0 with another country's ISO 3166-2, 127 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 157.
- Names: name 157, name:en 28, int_name 4, Latin name 151, any English-capable name 157 (of 157).
- Script of `name`: `Latin` 154, `Arabic` 3.
- `name:*` tags: `name:zh` 153, `name:ar` 142, `name:ta` 57, `name:ko` 29, `name:en` 28, `name:ms` 27, `name:ja` 21, `name:th` 18.
- Geometry: the relations add up to 327,605 km² (64.7 % of the outline); 8,582 land plots lie in none of them, 10 in two or more; 0 relations hold no land plot; land plots per relation: 41 to 21,534, median 1,516.
- Rings not closed in the extract (a member way is missing): 0 in scope; 37 of the level's relations could not be assembled at all.
- Examples: Alor Gajah, Bintulu, Jelebu, Kota Belud, Kuching, Marang, Penampang, Saratok, South-West (Barat Daya), Yan.

### Level 7

- 356 relations in the extract, 22 in scope (country). Left out: 0 with another country's ISO 3166-2, 334 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 22.
- Names: name 22, name:en 9, int_name 0, Latin name 22, any English-capable name 22 (of 22).
- Script of `name`: `Latin` 22.
- `name:*` tags: `name:en` 9, `name:ms` 7, `name:zh` 7, `name:ar` 5, `name:ms-Arab` 5, `name:ko` 1, `name:th` 1.
- Geometry: the relations add up to 14,247 km² (2.8 % of the outline); 402,015 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 10 to 5,760, median 420.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Ampang Jaya Municipal Council (Majlis Perbandaran Ampang Jaya), Bario, Engkilili, Kajang Municipal Council (Majlis Perbandaran Kajang), Kota Bharu, Majlis Bandaraya Ipoh, Petaling Jaya, Sadong Jaya, Selayang Municipal Council (Majlis Perbandaran Selayang), Subang Jaya City Council (Majlis Bandaraya Subang Jaya).

### Level 8

- 86 relations in the extract, 31 in scope (country). Left out: 0 with another country's ISO 3166-2, 55 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 31.
- Names: name 31, name:en 5, int_name 0, Latin name 31, any English-capable name 31 (of 31).
- Script of `name`: `Latin` 31.
- `name:*` tags: `name:zh` 11, `name:ms` 9, `name:en` 5, `name:ar` 3, `name:zh-Hans` 3, `name:zh-Hant` 3, `name:ms-Arab` 2.
- Geometry: the relations add up to 1,329 km² (0.3 % of the outline); 418,281 land plots lie in none of them, 2 in two or more; 3 relations hold no land plot; land plots per relation: 0 to 195, median 27.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Ampang, Beriah, Bukit Mertajam, Hulu Bernam Barat, Mukim 18, Pantai Acheh, Pulau Redang, Tanjong Pinang, Teluk Bahang, Ulu Kelang.

### Level 10

- 380 relations in the extract, 141 in scope (country). Left out: 0 with another country's ISO 3166-2, 238 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 141.
- Names: name 141, name:en 108, int_name 0, Latin name 141, any English-capable name 141 (of 141).
- Script of `name`: `Latin` 141.
- `name:*` tags: `name:en` 108, `name:zh` 64, `name:ms` 55, `name:zh-Hans` 28, `name:zh-Hant` 27, `name:ms-Arab` 2.
- Geometry: the relations add up to 445 km² (0.1 % of the outline); 419,315 land plots lie in none of them, 2 in two or more; 35 relations hold no land plot; land plots per relation: 0 to 34, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: 18A, PJS 5, Section 10 (Seksyen 10), Section 17A (Seksyen 17A), Section 25 (Seksyen 25), Section 5 (Seksyen 5), Section U12 (Seksyen U12), Section U8 (Seksyen U8), SS 5, Voluntary Association Complex.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 16 relations, all with ISO 3166-2 MY-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 13 plots in none).
- Districts: level 6: 157 relations, none with an ISO 3166-2 code, covering 98.0 % of the country's land plots (0.00 % overlap, 8,582 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:ms-Latn`, `name` (when written in Latin letters). Local name: `name:ms`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 16 relations)

- English name from: `name:en` 11, `name` 5; none for 0.
- Local name from: `name:ms` 16; none for 0. English equals local for 14.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 157 relations)

- English name from: `name` 129, `name:en` 28; none for 0.
- Local name from: `name` 130, `name:ms` 27; none for 0. English equals local for 137.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-malaysia`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 14,168. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 4,954 | 4,856 | 4,856 |
| neighbourhood | 4,058 | 4,053 | 4,047 |
| hamlet | 3,133 | 3,073 | 3,073 |
| suburb | 529 | 529 | 529 |
| town | 416 | 415 | 415 |
| locality | 372 | 369 | 369 |
| quarter | 335 | 335 | 335 |
| district | 86 | 86 | 86 |
| city | 69 | 69 | 69 |
| square | 55 | 54 | 54 |
| isolated_dwelling | 36 | 26 | 26 |
| city_block | 26 | 26 | 7 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Kuala Lumpur (population 1,982,112): neighbourhood 90 (90 English-capable), suburb 32 (32 English-capable), square 22 (22 English-capable), hamlet 16 (16 English-capable), quarter 16 (16 English-capable), city 2 (2 English-capable).
- Kajang (population 1,047,356): neighbourhood 74 (74 English-capable), hamlet 9 (9 English-capable), suburb 6 (6 English-capable), quarter 3 (3 English-capable), town 1 (1 English-capable).
- Subang Jaya (population 902,086): quarter 55 (55 English-capable), neighbourhood 46 (46 English-capable), suburb 18 (18 English-capable), hamlet 5 (5 English-capable), city 1 (1 English-capable), city_block 1 (1 English-capable).
- Klang (population 902,025): neighbourhood 58 (58 English-capable), quarter 11 (11 English-capable), suburb 11 (11 English-capable), hamlet 8 (8 English-capable), district 1 (1 English-capable).
- Johor Bahru (population 858,118): neighbourhood 41 (41 English-capable), hamlet 12 (12 English-capable), suburb 6 (6 English-capable), locality 1 (1 English-capable), village 1 (1 English-capable).

## Size

- Land plots: 419,805. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 17 s and 385 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Malaysia: from malaysia-singapore-brunei-latest.osm.pbf
{Code: "my", Extract: "malaysia-singapore-brunei",
	Province: Rule{AdminLevel: 4, ISOPrefix: "MY", Count: 16, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:ms-Latn", "name"}, LocalLang: "ms",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('my', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
