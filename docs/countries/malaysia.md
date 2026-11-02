# Country scan: Malaysia

Written by `go run ./cmd/countryscan -country my` from `malaysia-singapore-brunei-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 16 relations, all with ISO 3166-2 MY-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 130 relations, none with an ISO 3166-2 code, covering 97.3 % of the country's land plots (0.00 % overlap, 8,511 plots in none). Per province: 0 to 43, median 8; 0 districts have their centre in no province.
- **Land plots:** 318,281.
- Provinces whose rings do not close in the extract: 1; without a land plot: 0.

## Outline and land

- Outline: the union of 15 admin_level 4 relations with ISO 3166-2 MY-xx (the country relation is not whole in the extract); 1 more of them are not whole either and are missing from it, so the outline and the land plots are too small; 309,814 km² in 155 polygons.
- Cells at resolution 8 with their centre inside the outline: 382,935; 318,281 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **318,281**, about 234,573 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with MY-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 0 | 4 | – | – | – | – |
| 3 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 32 | 16 | country | 16 | 11 | 0 | 16 | 100.0 % | 0.0 % | 0.00 % | 1 |
| 5 | 37 | 14 | country | 0 | 13 | 0 | 14 | 52.0 % | 48.0 % | 0.00 % | 0 |
| 6 | 284 | 130 | country | 0 | 28 | 4 | 124 | 97.3 % | 2.7 % | 0.00 % | 0 |
| 7 | 356 | 22 | country | 0 | 9 | 0 | 22 | 5.6 % | 94.4 % | 0.00 % | 0 |
| 8 | 86 | 30 | country | 0 | 5 | 0 | 30 | 0.5 % | 99.5 % | 0.00 % | 0 |
| 10 | 380 | 130 | country | 0 | 108 | 0 | 130 | 0.2 % | 99.8 % | 0.00 % | 0 |

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
- Geometry: the relations add up to 309,814 km² (100.0 % of the outline); 0 land plots lie in none of them, 3 in two or more; 0 relations hold no land plot; land plots per relation: 59 to 164,333, median 9,241.
- Rings not closed in the extract (a member way is missing): 1 in scope; 13 of the level's relations could not be assembled at all. In scope: r3879783 Sabah.
- Examples: Johor, Kedah, Kuala Lumpur, Malacca (Melaka), Negeri Sembilan, Penang (Pulau Pinang), Perlis, Putrajaya, Sarawak, Terengganu.

### Level 5

- 37 relations in the extract, 14 in scope (country). Left out: 0 with another country's ISO 3166-2, 23 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 14.
- Names: name 14, name:en 13, int_name 0, Latin name 14, any English-capable name 14 (of 14).
- Script of `name`: `Latin` 14.
- `name:*` tags: `name:zh` 14, `name:en` 13, `name:ms` 13, `name:ar` 10, `name:id` 1, `name:ko` 1, `name:ms-Arab` 1, `name:th` 1.
- Geometry: the relations add up to 125,776 km² (40.6 % of the outline); 152,748 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 357 to 51,952, median 7,359.
- Rings not closed in the extract (a member way is missing): 0 in scope; 12 of the level's relations could not be assembled at all.
- Examples: Betong Division (Bahagian Betong), Bintulu Division (Bahagian Bintulu), George Town, Kuching Division (Bahagian Kuching), Limbang Division (Bahagian Limbang), Mukah Division (Bahagian Mukah), Samarahan Division (Bahagian Samarahan), Seberang Perai, Serian Division (Bahagian Serian), Sri Aman Division (Bahagian Sri Aman).

### Level 6

- 284 relations in the extract, 130 in scope (country). Left out: 0 with another country's ISO 3166-2, 154 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 130.
- Names: name 130, name:en 28, int_name 4, Latin name 124, any English-capable name 130 (of 130).
- Script of `name`: `Latin` 127, `Arabic` 3.
- `name:*` tags: `name:zh` 126, `name:ar` 116, `name:ta` 57, `name:ko` 29, `name:en` 28, `name:ms` 26, `name:ja` 21, `name:th` 18.
- Geometry: the relations add up to 251,218 km² (81.1 % of the outline); 8,511 land plots lie in none of them, 10 in two or more; 0 relations hold no land plot; land plots per relation: 152 to 21,534, median 1,267.
- Rings not closed in the extract (a member way is missing): 0 in scope; 38 of the level's relations could not be assembled at all.
- Examples: Alor Gajah, Betong, Hulu Perak, Kerian, Kuala Terengganu, Machang (Jajahan Machang), Pantu, Sabak Bernam, Simunjan, Yan.

### Level 7

- 356 relations in the extract, 22 in scope (country). Left out: 0 with another country's ISO 3166-2, 334 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 22.
- Names: name 22, name:en 9, int_name 0, Latin name 22, any English-capable name 22 (of 22).
- Script of `name`: `Latin` 22.
- `name:*` tags: `name:en` 9, `name:ms` 7, `name:zh` 7, `name:ar` 5, `name:ms-Arab` 5, `name:ko` 1, `name:th` 1.
- Geometry: the relations add up to 14,247 km² (4.6 % of the outline); 300,491 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 10 to 5,760, median 420.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Ampang Jaya Municipal Council (Majlis Perbandaran Ampang Jaya), Bario, Engkilili, Kajang Municipal Council (Majlis Perbandaran Kajang), Kota Bharu, Majlis Bandaraya Ipoh, Petaling Jaya, Sadong Jaya, Selayang Municipal Council (Majlis Perbandaran Selayang), Subang Jaya City Council (Majlis Bandaraya Subang Jaya).

### Level 8

- 86 relations in the extract, 30 in scope (country). Left out: 0 with another country's ISO 3166-2, 56 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 30.
- Names: name 30, name:en 5, int_name 0, Latin name 30, any English-capable name 30 (of 30).
- Script of `name`: `Latin` 30.
- `name:*` tags: `name:zh` 11, `name:ms` 9, `name:en` 5, `name:ar` 3, `name:zh-Hans` 3, `name:zh-Hant` 3, `name:ms-Arab` 2.
- Geometry: the relations add up to 1,328 km² (0.4 % of the outline); 316,757 land plots lie in none of them, 2 in two or more; 2 relations hold no land plot; land plots per relation: 0 to 195, median 29.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Ampang, Beriah, Bukit Mertajam, Cyberjaya, Mukim 18, Pekan Bukit Kemuning, Pulau Redang, Tanjong Pinang, Teluk Bahang, Ulu Kelang.

### Level 10

- 380 relations in the extract, 130 in scope (country). Left out: 0 with another country's ISO 3166-2, 249 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 130.
- Names: name 130, name:en 108, int_name 0, Latin name 130, any English-capable name 130 (of 130).
- Script of `name`: `Latin` 130.
- `name:*` tags: `name:en` 108, `name:zh` 64, `name:ms` 55, `name:zh-Hans` 28, `name:zh-Hant` 27, `name:ms-Arab` 2.
- Geometry: the relations add up to 444 km² (0.1 % of the outline); 317,791 land plots lie in none of them, 2 in two or more; 24 relations hold no land plot; land plots per relation: 0 to 34, median 2.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: 18A, PJS 6, Section 1 (Seksyen 1), Section 17 (Seksyen 17), Section 22 (Seksyen 22), Section 35 (Seksyen 35), Section 9 (Seksyen 9), Section U20 (Seksyen U20), SS 22, Taman Mastiara Fasa 2.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 16 relations, all with ISO 3166-2 MY-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
- Districts: level 6: 130 relations, none with an ISO 3166-2 code, covering 97.3 % of the country's land plots (0.00 % overlap, 8,511 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:ms-Latn`, `name` (when written in Latin letters). Local name: `name:ms`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 15 relations)

- English name from: `name:en` 11, `name` 4; none for 0.
- Local name from: `name:ms` 15; none for 0. English equals local for 13.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 130 relations)

- English name from: `name` 102, `name:en` 28; none for 0.
- Local name from: `name` 104, `name:ms` 26; none for 0. English equals local for 111.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-malaysia`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 12,185. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| neighbourhood | 3,835 | 3,830 | 3,824 |
| village | 3,658 | 3,581 | 3,581 |
| hamlet | 2,853 | 2,814 | 2,814 |
| suburb | 499 | 499 | 499 |
| town | 359 | 358 | 358 |
| locality | 342 | 339 | 339 |
| quarter | 331 | 331 | 331 |
| district | 84 | 84 | 84 |
| city | 64 | 64 | 64 |
| square | 54 | 53 | 53 |
| isolated_dwelling | 31 | 21 | 21 |
| region | 16 | 16 | 16 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Kuala Lumpur (population 1,982,112): neighbourhood 90 (90 English-capable), suburb 32 (32 English-capable), square 22 (22 English-capable), hamlet 16 (16 English-capable), quarter 16 (16 English-capable), city 2 (2 English-capable).
- Kajang (population 1,047,356): neighbourhood 74 (74 English-capable), hamlet 9 (9 English-capable), suburb 6 (6 English-capable), quarter 3 (3 English-capable), town 1 (1 English-capable).
- Subang Jaya (population 902,086): quarter 55 (55 English-capable), neighbourhood 46 (46 English-capable), suburb 18 (18 English-capable), hamlet 5 (5 English-capable), city 1 (1 English-capable), city_block 1 (1 English-capable).
- Klang (population 902,025): neighbourhood 58 (58 English-capable), quarter 11 (11 English-capable), suburb 11 (11 English-capable), hamlet 8 (8 English-capable), district 1 (1 English-capable).
- Johor Bahru (population 858,118): neighbourhood 41 (41 English-capable), hamlet 12 (12 English-capable), suburb 6 (6 English-capable), locality 1 (1 English-capable), village 1 (1 English-capable).

## Size

- Land plots: 318,281. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 13 s and 292 MiB. A rough figure; it was not measured for this country.

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
