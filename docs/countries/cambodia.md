# Country scan: Cambodia

Written by `go run ./cmd/countryscan -country kh` from `cambodia-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 25 relations, all with ISO 3166-2 KH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 4 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 200 relations, none with an ISO 3166-2 code, covering 92.8 % of the country's land plots (0.00 % overlap, 15,432 plots in none). Per province: 2 to 14, median 8; 0 districts have their centre in no province.
- **Land plots:** 213,169.

## Outline and land

- Outline: the country relation r49898 (admin_level 2); 204,611 km² in 1 polygons.
- Cells at resolution 8 with their centre inside the outline: 238,166; 213,169 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **213,169**, about 157,105 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with KH-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 4 | 4 | extract | 0 | 4 | 3 | 1 | – | – | – | – |
| 4 | 42 | 25 | country | 25 | 25 | 0 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 6 | 302 | 200 | country | 0 | 200 | 0 | 0 | 92.8 % | 7.2 % | 0.00 % | 0 |
| 8 | 100 | 100 | country | 0 | 100 | 0 | 0 | 2.3 % | 97.7 % | 0.00 % | 0 |
| 9 | 36 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Cambodia (ព្រះរាជាណាចក្រកម្ពុជា), Laos (ឡាវ), Thailand (ថៃ), Vietnam (វៀតណាម).

### Level 4

- 42 relations in the extract, 25 in scope (country). Left out: 17 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `KH` 25; with this country's prefix: 25 of 25.
- Names: name 25, name:en 25, int_name 0, Latin name 0, any English-capable name 25 (of 25).
- Script of `name`: `Khmer` 25.
- `name:*` tags: `name:en` 25, `name:fr` 25, `name:ru` 25, `name:ja` 24, `name:zh` 23, `name:km-Latn` 20, `name:ko` 10, `name:lt` 6.
- Geometry: the relations add up to 204,611 km² (100.0 % of the outline); 4 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 179 to 16,511, median 7,128.
- Rings not closed in the extract (a member way is missing): 0 in scope; 17 of the level's relations could not be assembled at all.
- Examples: Bantey Meanchey (ខេត្តបន្ទាយមានជ័យ), Kampong Cham (ខេត្តកំពង់ចាម), Kampong Thom (ខេត្តកំពង់ធំ), Kep (ខេត្តកែប), Koh Kong (ខេត្តកោះកុង), Oddar Meanchey (ខេត្តឧត្ដរមានជ័យ), Preah Vihear (ខេត្តព្រះវិហារ), Pursat (ខេត្តពោធិ៍សាត់), Stung Treng (ខេត្តស្ទឹងត្រែង), Tbong Khmum (ខេត្តត្បូងឃ្មុំ).

### Level 6

- 302 relations in the extract, 200 in scope (country). Left out: 0 with another country's ISO 3166-2, 102 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 200.
- Names: name 200, name:en 200, int_name 0, Latin name 0, any English-capable name 200 (of 200).
- Script of `name`: `Khmer` 200.
- `name:*` tags: `name:en` 200, `name:zh` 15, `name:ru` 13, `name:km` 12, `name:th` 1, `name:uk` 1.
- Geometry: the relations add up to 169,478 km² (82.8 % of the outline); 15,432 land plots lie in none of them, 9 in two or more; 0 relations hold no land plot; land plots per relation: 4 to 6,299, median 591.
- Rings not closed in the extract (a member way is missing): 0 in scope; 98 of the level's relations could not be assembled at all.
- Examples: Aek Phnum (ឯកភ្នំ), Bavel (បវេល), Doun Kaev (ដូនកែវ), Kep (កែប), Koun Mom (កូនមុំ), Pailin (ប៉ៃលិន), Prey Nob (ស្រុកព្រៃនប់), Samraong (សំរោង), Svay Chek (ស្វាយចេក), Veun Sai (វើនសៃ).

### Level 8

- 100 relations in the extract, 100 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 100.
- Names: name 100, name:en 100, int_name 0, Latin name 0, any English-capable name 100 (of 100).
- Script of `name`: `Khmer` 100.
- `name:*` tags: `name:en` 100, `name:km` 11, `name:zh` 9, `name:ru` 2.
- Geometry: the relations add up to 4,199 km² (2.1 % of the outline); 208,261 land plots lie in none of them, 4 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 248, median 40.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Ampil Commune (ឃុំអំពិល), Boeng Ta Prum Quarter (សង្កាត់បឹងតាព្រហ្ម), Doun Sa Commune (ឃុំដូនស), Kouk Pring Commune (ឃុំគោកព្រីង), Nhor Commune (ឃុំញរ), Preah Ponlea Commune (ឃុំព្រះពន្លា), Reach Montir Commune (ឃុំរាជមន្ទីរ), Sangkhoar Quarter (សង្កាត់សង្ឃ័រ), Svay Yea Commune (ស្វាយយា), Tuol Sdei Commune (ឃុំទួលស្ដី).

### Level 9

- 36 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 36 with their centre outside the country (or not assembled and without ISO code).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 25 relations, all with ISO 3166-2 KH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 4 plots in none).
- Districts: level 6: 200 relations, none with an ISO 3166-2 code, covering 92.8 % of the country's land plots (0.00 % overlap, 15,432 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:km-Latn`, `name` (when written in Latin letters). Local name: `name:km`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 25 relations)

- English name from: `name:en` 25; none for 0.
- Local name from: `name` 21, `name:km` 4; none for 0. English equals local for 0.
- English prefixes: “Kampong” 4. Suffixes: none.
- Local prefixes: “ខេត្ត” 24, “ខេត្តក” 9. Suffixes: “ង” 6.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 200 relations)

- English name from: `name:en` 200; none for 0.
- Local name from: `name` 188, `name:km` 12; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-cambodia`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 1,990. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 1,021 | 1,006 | 765 |
| hamlet | 564 | 559 | 533 |
| town | 197 | 197 | 179 |
| locality | 70 | 69 | 65 |
| neighbourhood | 62 | 61 | 59 |
| city | 28 | 28 | 28 |
| suburb | 23 | 23 | 22 |
| state | 7 | 7 | 7 |
| quarter | 4 | 4 | 4 |
| district | 2 | 2 | 2 |
| farm | 2 | 2 | 1 |
| isolated_dwelling | 2 | 1 | 1 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Phnom Penh (population 2,009,264): village 66 (66 English-capable), hamlet 50 (50 English-capable), locality 16 (15 English-capable), town 10 (10 English-capable), neighbourhood 9 (9 English-capable), suburb 4 (3 English-capable).
- Battambang (population 196,709): village 16 (16 English-capable), locality 3 (2 English-capable), city 1 (1 English-capable), hamlet 2 (2 English-capable), town 2 (2 English-capable).
- Siem Reap (population 171,800): locality 5 (5 English-capable), village 2 (2 English-capable), neighbourhood 1 (1 English-capable), state 1 (1 English-capable).
- Serei Saophoan (population 132,264): village 6 (6 English-capable), hamlet 3 (3 English-capable).
- Kampong Cham (population 118,242): village 24 (24 English-capable), town 1 (1 English-capable).

## Size

- Land plots: 213,169. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 8 s and 195 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Cambodia: from cambodia-latest.osm.pbf
{Code: "kh", Extract: "cambodia",
	Province: Rule{AdminLevel: 4, ISOPrefix: "KH", Count: 25, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:km-Latn", "name"}, LocalLang: "km",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{[]string{"ខេត្ត"}, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('kh', '{neighbourhood}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
