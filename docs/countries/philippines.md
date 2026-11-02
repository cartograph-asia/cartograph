# Country scan: Philippines

Written by `go run ./cmd/countryscan -country ph` from `philippines-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 3: 18 relations, 17 with ISO 3166-2 PH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 2 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 3, 4.
- **Districts:** level 4: 83 relations, 82 with ISO 3166-2 PH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none). Per province: 0 to 7, median 5; 0 districts have their centre in no province.
- **Land plots:** 367,912.
- Provinces whose rings do not close in the extract: 0; without a land plot: 1.

## Outline and land

- Outline: the union of 82 admin_level 4 relations with ISO 3166-2 PH-xx (the country relation is not whole in the extract); 597,382 km² in 103 polygons.
- Cells at resolution 8 with their centre inside the outline: 770,070; 367,912 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **367,912**, about 271,151 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with PH-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 3 | 4 | – | – | – | – |
| 3 | 18 | 18 | country | 17 | 17 | 0 | 18 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 4 | 89 | 83 | country | 82 | 79 | 0 | 83 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 16 | 8 | country | 0 | 0 | 0 | 8 | 2.3 % | 97.7 % | 0.00 % | 0 |
| 6 | 911 | 862 | country | 0 | 178 | 0 | 862 | 52.1 % | 47.9 % | 0.01 % | 0 |
| 7 | 40 | 1 | country | 0 | 0 | 0 | 1 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 8 | 118 | 31 | country | 0 | 0 | 0 | 31 | 0.2 % | 99.8 % | 0.00 % | 0 |
| 9 | 55 | 13 | country | 0 | 0 | 0 | 13 | 0.1 % | 99.9 % | 0.00 % | 0 |
| 10 | 5,042 | 3,420 | country | 0 | 7 | 0 | 3,413 | 5.5 % | 94.5 % | 0.02 % | 0 |

### Level 2

Countries in the extract: China (中国), Indonesia, Malaysia, Philippines, Taiwan (臺灣), Vietnam.

### Level 3

- 18 relations in the extract, 18 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `PH` 17; with this country's prefix: 17 of 18.
- Names: name 18, name:en 17, int_name 0, Latin name 18, any English-capable name 18 (of 18).
- Script of `name`: `Latin` 18.
- `name:*` tags: `name:ko` 18, `name:en` 17, `name:tl` 17, `name:es` 16, `name:ja` 16, `name:ru` 14, `name:ceb` 8, `name:ilo` 4.
- Geometry: the relations add up to 615,478 km² (103.0 % of the outline); 2 land plots lie in none of them, 3 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 36,290, median 20,929.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Bangsamoro, Bicol Region, Calabarzon, Central Luzon, Cordillera Administrative Region, Eastern Visayas, Metro Manila, Negros Island Region, Soccsksargen, Zamboanga Peninsula.

### Level 4

- 89 relations in the extract, 83 in scope (country). Left out: 6 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `PH` 82; with this country's prefix: 82 of 83.
- Names: name 83, name:en 79, int_name 0, Latin name 83, any English-capable name 83 (of 83).
- Script of `name`: `Latin` 83.
- `name:*` tags: `name:ko` 82, `name:en` 79, `name:tl` 78, `name:ja` 77, `name:ru` 65, `name:eo` 63, `name:es` 23, `name:hu` 12.
- Geometry: the relations add up to 597,877 km² (100.1 % of the outline); 0 land plots lie in none of them, 10 in two or more; 0 relations hold no land plot; land plots per relation: 2 to 16,621, median 3,902.
- Rings not closed in the extract (a member way is missing): 0 in scope; 6 of the level's relations could not be assembled at all.
- Examples: Abra, Bataan, Camarines Norte, Davao del Norte, Ilocos Sur, Maguindanao del Norte, Northern Samar, Quirino, Special Geographic Area, Zamboanga Sibugay.

### Level 5

- 16 relations in the extract, 8 in scope (country). Left out: 0 with another country's ISO 3166-2, 8 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 8.
- Names: name 8, name:en 0, int_name 0, Latin name 8, any English-capable name 8 (of 8).
- Script of `name`: `Latin` 8.
- `name:*` tags: none.
- Geometry: the relations add up to 13,736 km² (2.3 % of the outline); 359,631 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 538 to 2,077, median 837.
- Rings not closed in the extract (a member way is missing): 0 in scope; 4 of the level's relations could not be assembled at all.
- Examples: 1st District, 1st District, 2nd District, 2nd District, 2nd District, 3rd District, 4th District, 5th District.

### Level 6

- 911 relations in the extract, 862 in scope (country). Left out: 0 with another country's ISO 3166-2, 49 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 862.
- Names: name 862, name:en 178, int_name 0, Latin name 862, any English-capable name 862 (of 862).
- Script of `name`: `Latin` 862.
- `name:*` tags: `name:ko` 223, `name:en` 178, `name:tl` 122, `name:es` 67, `name:etymology` 62, `name:ja` 59, `name:etymology:wikidata` 57, `name:prefix` 35.
- Geometry: the relations add up to 350,915 km² (58.7 % of the outline); 176,393 land plots lie in none of them, 27 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 3,074, median 144.
- Rings not closed in the extract (a member way is missing): 0 in scope; 5 of the level's relations could not be assembled at all.
- Examples: Abra de Ilog, Batangas City, Catbalogan, Hadji Mohammad Ajul, Libungan, Matanao, Pigcawayan, San Jose, Sominot, Zumarraga.

### Level 7

- 40 relations in the extract, 1 in scope (country). Left out: 0 with another country's ISO 3166-2, 39 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1.
- Names: name 1, name:en 0, int_name 0, Latin name 1, any English-capable name 1 (of 1).
- Script of `name`: `Latin` 1.
- `name:*` tags: none.
- Geometry: the relations add up to 1 km² (0.0 % of the outline); 367,910 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 2 to 2, median 2.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Poblacion, Calamba.

### Level 8

- 118 relations in the extract, 31 in scope (country). Left out: 0 with another country's ISO 3166-2, 87 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 31.
- Names: name 31, name:en 0, int_name 0, Latin name 31, any English-capable name 31 (of 31).
- Script of `name`: `Latin` 31.
- `name:*` tags: none.
- Geometry: the relations add up to 500 km² (0.1 % of the outline); 367,282 land plots lie in none of them, 1 in two or more; 2 relations hold no land plot; land plots per relation: 0 to 129, median 8.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Babak District, Burol, Freeport Area of Bataan, Locion District, Paliparan, Poblacion, Poblacion, Salitran, San Agustin, Santa Cruz.

### Level 9

- 55 relations in the extract, 13 in scope (country). Left out: 0 with another country's ISO 3166-2, 42 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 13.
- Names: name 13, name:en 0, int_name 0, Latin name 13, any English-capable name 13 (of 13).
- Script of `name`: `Latin` 13.
- `name:*` tags: none.
- Geometry: the relations add up to 360 km² (0.1 % of the outline); 367,428 land plots lie in none of them, 1 in two or more; 3 relations hold no land plot; land plots per relation: 0 to 219, median 3.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: East Tabacal, Forest, Poblacion, West Tabacal, Zone 1, Zone 2, Zone 4, Zone 5, Zone 6, Zone 8.

### Level 10

- 5,042 relations in the extract, 3,420 in scope (country). Left out: 0 with another country's ISO 3166-2, 1,605 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 3,420.
- Names: name 3,420, name:en 7, int_name 0, Latin name 3,413, any English-capable name 3,413 (of 3,420).
- Script of `name`: `Latin` 3,413.
- `name:*` tags: `name:prefix` 54, `name:ko` 24, `name:en` 7, `name:etymology:wikidata` 7, `name:etymology` 5, `name:pronunciation` 5, `name:tl` 4, `name:es` 3.
- Geometry: the relations add up to 15,961 km² (2.7 % of the outline); 347,766 land plots lie in none of them, 88 in two or more; 694 relations hold no land plot; land plots per relation: 0 to 806, median 3.
- Rings not closed in the extract (a member way is missing): 0 in scope; 124 of the level's relations could not be assembled at all.
- Examples: 1, Barangay 4, Butawanan, Dulangan, Lubukan, Oaqui No. 1, Puerto Rivas Lote, San Matias, Sawang, Zone VIII.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 3: 18 relations, 17 with ISO 3166-2 PH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 2 plots in none).
- Districts: level 4: 83 relations, 82 with ISO 3166-2 PH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).

Levels that qualify: 3, 4. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:fil-Latn`, `name` (when written in Latin letters). Local name: `name:fil`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 3, 18 relations)

- English name from: `name:en` 17, `name` 1; none for 0.
- Local name from: `name` 18; none for 0. English equals local for 18.
- English prefixes: none. Suffixes: “Region” 5, “Visayas” 3.
- Local prefixes: none. Suffixes: “Region” 5, “Visayas” 3.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 4, 83 relations)

- English name from: `name:en` 79, `name` 4; none for 0.
- Local name from: `name` 82, `name:fil` 1; none for 0. English equals local for 82.
- English prefixes: none. Suffixes: “Norte” 8, “Sur” 8.
- Local prefixes: none. Suffixes: “Norte” 8, “Sur” 8.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-philippines`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 57,767. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 26,950 | 26,659 | 26,657 |
| hamlet | 11,584 | 7,973 | 7,969 |
| neighbourhood | 9,993 | 9,502 | 9,502 |
| quarter | 5,055 | 5,054 | 5,054 |
| town | 1,489 | 1,489 | 1,489 |
| city_block | 738 | 729 | 729 |
| locality | 445 | 444 | 439 |
| suburb | 397 | 397 | 397 |
| island | 330 | 330 | 330 |
| zone | 221 | 220 | 220 |
| isolated_dwelling | 135 | 16 | 16 |
| city | 115 | 115 | 115 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Antipolo (population 887,399): neighbourhood 301 (297 English-capable), quarter 37 (37 English-capable), city_block 15 (15 English-capable), village 8 (8 English-capable), suburb 4 (4 English-capable), town 4 (4 English-capable).
- Dasmariñas (population 744,511): neighbourhood 95 (95 English-capable), quarter 76 (76 English-capable), city_block 75 (74 English-capable), suburb 10 (10 English-capable), hamlet 6 (6 English-capable), camp 1 (1 English-capable).
- Bacoor (population 661,381): quarter 135 (135 English-capable), neighbourhood 94 (94 English-capable), suburb 6 (6 English-capable), city 2 (2 English-capable), town 2 (2 English-capable), camp 1 (1 English-capable).
- San Jose del Monte (population 651,813): neighbourhood 76 (76 English-capable), quarter 72 (72 English-capable), city_block 47 (47 English-capable), suburb 11 (11 English-capable), plot 9 (9 English-capable), village 4 (4 English-capable).
- Biñan (population 584,479): quarter 85 (85 English-capable), neighbourhood 51 (50 English-capable), city 4 (4 English-capable), hamlet 3 (3 English-capable), city_block 1 (1 English-capable).

## Size

- Land plots: 367,912. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 15 s and 338 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Philippines: from philippines-latest.osm.pbf
{Code: "ph", Extract: "philippines",
	Province: Rule{AdminLevel: 3, ISOPrefix: "PH", Count: 18, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 4},
	EnglishName: []string{"name:en", "int_name", "name:fil-Latn", "name"}, LocalLang: "fil",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('ph', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
