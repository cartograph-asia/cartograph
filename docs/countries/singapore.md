# Country scan: Singapore

Written by `go run ./cmd/countryscan -country sg` from `malaysia-singapore-brunei-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 5: 5 relations, none with an ISO 3166-2 code, covering 99.7 % of the country's land plots (0.11 % overlap, 3 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 5, 6, 7.
- **Districts:** level 6: 55 relations, none with an ISO 3166-2 code, covering 99.5 % of the country's land plots (0.21 % overlap, 5 plots in none). Per province: 6 to 22, median 8; 0 districts have their centre in no province.
- **Land plots:** 941.

## Outline and land

- Outline: the country relation r536780 (admin_level 2); 1,650 km² in 3 polygons.
- Cells at resolution 8 with their centre inside the outline: 1,968; 941 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **941**, about 693 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with SG-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 0 | 4 | – | – | – | – |
| 3 | 3 | 1 | country | 0 | 1 | 0 | 1 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 32 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 5 | 37 | 5 | country | 0 | 5 | 0 | 5 | 99.7 % | 0.3 % | 0.11 % | 0 |
| 6 | 284 | 55 | country | 0 | 0 | 0 | 55 | 99.5 % | 0.5 % | 0.21 % | 0 |
| 7 | 356 | 331 | country | 0 | 3 | 0 | 331 | 91.2 % | 8.8 % | 0.43 % | 0 |
| 8 | 86 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 10 | 380 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Brunei, Indonesia, Malaysia, Philippines, Singapore, Thailand.

### Level 3

- 3 relations in the extract, 1 in scope (country). Left out: 2 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1.
- Names: name 1, name:en 1, int_name 0, Latin name 1, any English-capable name 1 (of 1).
- Script of `name`: `Latin` 1.
- `name:*` tags: `name:en` 1, `name:fr` 1, `name:ms` 1, `name:ru` 1, `name:ta` 1, `name:zh` 1.
- Geometry: the relations add up to 218 km² (13.2 % of the outline); 941 land plots lie in none of them, 0 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 0, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Pedra Branca.

### Level 4

- 32 relations in the extract, 0 in scope (country). Left out: 32 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).

### Level 5

- 37 relations in the extract, 5 in scope (country). Left out: 0 with another country's ISO 3166-2, 32 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 5.
- Names: name 5, name:en 5, int_name 0, Latin name 5, any English-capable name 5 (of 5).
- Script of `name`: `Latin` 5.
- `name:*` tags: `name:en` 5, `name:th` 5.
- Geometry: the relations add up to 789 km² (47.8 % of the outline); 3 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 141 to 313, median 161.
- Rings not closed in the extract (a member way is missing): 0 in scope; 12 of the level's relations could not be assembled at all.
- Examples: Central Region, East Region, North Region, North-east Region, West Region.

### Level 6

- 284 relations in the extract, 55 in scope (country). Left out: 0 with another country's ISO 3166-2, 229 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 55.
- Names: name 55, name:en 0, int_name 0, Latin name 55, any English-capable name 55 (of 55).
- Script of `name`: `Latin` 55.
- `name:*` tags: `name:th` 55, `name:ta` 47, `name:zh` 47, `name:ru` 2.
- Geometry: the relations add up to 789 km² (47.8 % of the outline); 5 land plots lie in none of them, 2 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 82, median 11.
- Rings not closed in the extract (a member way is missing): 0 in scope; 38 of the level's relations could not be assembled at all.
- Examples: Ang Mo Kio, Bukit Panjang, Clementi, Kallang, Museum, Pasir Ris, Rochor, Singapore River, Tengah, Yishun.

### Level 7

- 356 relations in the extract, 331 in scope (country). Left out: 0 with another country's ISO 3166-2, 25 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 331.
- Names: name 331, name:en 3, int_name 0, Latin name 331, any English-capable name 331 (of 331).
- Script of `name`: `Latin` 331.
- `name:*` tags: `name:zh` 97, `name:ta` 22, `name:en` 3.
- Geometry: the relations add up to 724 km² (43.9 % of the outline); 83 land plots lie in none of them, 4 in two or more; 58 relations hold no land plot; land plots per relation: 0 to 81, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Admiralty, Bukit Batok South, Dhoby Ghaut, International Business Park, Liu Fang, Newton Circus, Raffles Park, Simei, Toh Guan, Yunnan.

### Level 8

- 86 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 86 with their centre outside the country (or not assembled and without ISO code).

### Level 10

- 380 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 379 with their centre outside the country (or not assembled and without ISO code).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 5: 5 relations, none with an ISO 3166-2 code, covering 99.7 % of the country's land plots (0.11 % overlap, 3 plots in none).
- Districts: level 6: 55 relations, none with an ISO 3166-2 code, covering 99.5 % of the country's land plots (0.21 % overlap, 5 plots in none).

Levels that qualify: 5, 6, 7. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:en-Latn`, `name` (when written in Latin letters). Local name: `name:en`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 5, 5 relations)

- English name from: `name:en` 5; none for 0.
- Local name from: `name:en` 5; none for 0. English equals local for 5.
- English prefixes: none. Suffixes: “Region” 5.
- Local prefixes: none. Suffixes: “Region” 5.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 55 relations)

- English name from: `name` 55; none for 0.
- Local name from: `name` 55; none for 0. English equals local for 55.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-singapore`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 602. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| neighbourhood | 410 | 410 | 410 |
| suburb | 77 | 77 | 77 |
| locality | 47 | 47 | 47 |
| quarter | 41 | 41 | 41 |
| islet | 10 | 10 | 10 |
| borough | 5 | 5 | 5 |
| island | 4 | 4 | 4 |
| village | 4 | 4 | 4 |
| city | 1 | 1 | 1 |
| country | 1 | 1 | 1 |
| farm | 1 | 0 | 0 |
| plot | 1 | 1 | 1 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Singapore (population 5,535,002): neighbourhood 182 (182 English-capable), suburb 33 (33 English-capable), locality 20 (20 English-capable), quarter 12 (12 English-capable), borough 1 (1 English-capable).

## Size

- Land plots: 941. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 1 s and 100 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Singapore: from malaysia-singapore-brunei-latest.osm.pbf
{Code: "sg", Extract: "malaysia-singapore-brunei",
	Province: Rule{AdminLevel: 5, ISOPrefix: "SG", Count: 5, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:en-Latn", "name"}, LocalLang: "en",
	StripProvince: Strip{English: [2][]string{nil, []string{"Region"}}, Local: [2][]string{nil, []string{"Region"}}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('sg', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 1 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
