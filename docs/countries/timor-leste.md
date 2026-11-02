# Country scan: Timor-Leste

Written by `go run ./cmd/countryscan -country tl` from `east-timor-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 14 relations, 13 with ISO 3166-2 TL-xx, covering 100.0 % of the country's land plots (0.01 % overlap, 7 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 5.
- **Districts:** level 5: 70 relations, none with an ISO 3166-2 code, covering 99.9 % of the country's land plots (0.02 % overlap, 10 plots in none). Per province: 3 to 7, median 5; 0 districts have their centre in no province.
- **Land plots:** 19,546.
- **Names (provinces):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 2 names whose slug loses letters without accent folding.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 2 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r305142 (admin_level 2); 29,377 km² in 2 polygons.
- Cells at resolution 8 with their centre inside the outline: 38,255; 19,546 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **19,546**, about 14,405 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with TL-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 2 | 2 | extract | 0 | 2 | 0 | 2 | – | – | – | – |
| 4 | 16 | 14 | country | 13 | 14 | 0 | 14 | 100.0 % | 0.0 % | 0.01 % | 0 |
| 5 | 74 | 70 | country | 0 | 34 | 0 | 70 | 99.9 % | 0.1 % | 0.02 % | 0 |
| 6 | 2 | 2 | country | 0 | 0 | 0 | 2 | 1.3 % | 98.7 % | 0.00 % | 0 |

### Level 2

Countries in the extract: East Timor (Timor-Leste), Indonesia (Indonésia).

### Level 4

- 16 relations in the extract, 14 in scope (country). Left out: 2 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `TL` 13; with this country's prefix: 13 of 14.
- Names: name 14, name:en 14, int_name 0, Latin name 14, any English-capable name 14 (of 14).
- Script of `name`: `Latin` 14.
- `name:*` tags: `name:en` 14, `name:tet` 14, `name:id` 13, `name:ja` 13, `name:pt` 13, `name:bg` 12, `name:fa` 11, `name:he` 11.
- Geometry: the relations add up to 14,973 km² (51.0 % of the outline); 7 land plots lie in none of them, 2 in two or more; 0 relations hold no land plot; land plots per relation: 186 to 2,460, median 1,597.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Aileu, Ainaro, Atauro (Ataúru), Bobonaro, Cova Lima, Ermera, Lautém, Manatuto, Manufahi, Viqueque.

### Level 5

- 74 relations in the extract, 70 in scope (country). Left out: 0 with another country's ISO 3166-2, 4 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 70.
- Names: name 70, name:en 34, int_name 0, Latin name 70, any English-capable name 70 (of 70).
- Script of `name`: `Latin` 70.
- `name:*` tags: `name:tet` 70, `name:en` 34, `name:zh` 19, `name:de` 1, `name:el` 1.
- Geometry: the relations add up to 14,973 km² (51.0 % of the outline); 10 land plots lie in none of them, 4 in two or more; 0 relations hold no land plot; land plots per relation: 4 to 848, median 262.
- Rings not closed in the extract (a member way is missing): 0 in scope; 4 of the level's relations could not be assembled at all.
- Examples: Aileu, Barique (Barike), Dom Aleixo, Hatulia, Laulara, Luro (Luru), Metinaro, Quelicai (Kelikai), Tutuala, Zumalai.

### Level 6

- 2 relations in the extract, 2 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2.
- Names: name 2, name:en 0, int_name 0, Latin name 2, any English-capable name 2 (of 2).
- Script of `name`: `Latin` 2.
- `name:*` tags: `name:tet` 2.
- Geometry: the relations add up to 200 km² (0.7 % of the outline); 19,291 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 116 to 139, median 139.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Mehara, Tutuala.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 14 relations, 13 with ISO 3166-2 TL-xx, covering 100.0 % of the country's land plots (0.01 % overlap, 7 plots in none).
- Districts: level 5: 70 relations, none with an ISO 3166-2 code, covering 99.9 % of the country's land plots (0.02 % overlap, 10 plots in none).

Levels that qualify: 4, 5. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:pt-Latn`, `name` (when written in Latin letters). Local name: `name:pt`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 14 relations)

- English name from: `name:en` 14; none for 0.
- Local name from: `name:pt` 13, `name` 1; none for 0. English equals local for 12.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
- **Accents:** a slug without folding (as before stage 2) drops accented letters; `osmfile.Slug` folds them (NFD, đ → d), which changes the slug of 2 names, e.g. Lautém: laut-m → lautem, Liquiçá: liqui → liquica.

### Districts (level 5, 70 relations)

- English name from: `name` 36, `name:en` 34; none for 0.
- Local name from: `name` 70; none for 0. English equals local for 42.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** a slug without folding (as before stage 2) drops accented letters; `osmfile.Slug` folds them (NFD, đ → d), which changes the slug of 2 names, e.g. Liquiçá: liqui → liquica, Lautém: laut-m → lautem.

The slug of a new country's province carries the country as a suffix (`-timor-leste`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 2,911. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| hamlet | 2,667 | 2,624 | 2,624 |
| village | 124 | 110 | 110 |
| neighbourhood | 56 | 56 | 56 |
| town | 53 | 53 | 53 |
| city | 4 | 4 | 4 |
| locality | 3 | 2 | 2 |
| country | 1 | 1 | 1 |
| county | 1 | 1 | 1 |
| plot | 1 | 0 | 0 |
| quarter | 1 | 1 | 1 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Dili (population 222,323): hamlet 57 (57 English-capable), neighbourhood 55 (55 English-capable), village 8 (8 English-capable), town 1 (1 English-capable).
- Gleno (population 8,133): hamlet 85 (85 English-capable), town 2 (2 English-capable), village 1 (1 English-capable).
- Pante-Macassar (population 4,730): hamlet 65 (65 English-capable), county 1 (1 English-capable), plot 1 (0 English-capable).

## Size

- Land plots: 19,546. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 1 s and 100 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Timor-Leste: from east-timor-latest.osm.pbf
{Code: "tl", Extract: "east-timor",
	Province: Rule{AdminLevel: 4, ISOPrefix: "TL", Count: 14, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 5},
	EnglishName: []string{"name:en", "int_name", "name:pt-Latn", "name"}, LocalLang: "pt",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('tl', '{neighbourhood}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 3 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
