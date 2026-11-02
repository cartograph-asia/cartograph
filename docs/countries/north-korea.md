# Country scan: North Korea

Written by `go run ./cmd/countryscan -country kp` from `north-korea-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 13 relations, all with ISO 3166-2 KP-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 171 relations, none with an ISO 3166-2 code, covering 99.1 % of the country's land plots (0.00 % overlap, 2,023 plots in none). Per province: 0 to 21, median 19; 0 districts have their centre in no province.
- **Land plots:** 214,569.
- **Names (provinces):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 1 names whose slug loses letters without accent folding.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 5 names whose slug loses letters without accent folding.
- Provinces whose rings do not close in the extract: 1; without a land plot: 0.

## Outline and land

- Outline: the union of 12 admin_level 4 relations with ISO 3166-2 KP-xx (the country relation is not whole in the extract); 1 more of them are not whole either and are missing from it, so the outline and the land plots are too small; 134,669 km² in 24 polygons.
- Cells at resolution 8 with their centre inside the outline: 259,914; 214,569 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **214,569**, about 158,137 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with KP-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 4 | 4 | extract | 0 | 4 | 3 | 0 | – | – | – | – |
| 3 | 1 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 19 | 13 | country | 13 | 13 | 1 | 0 | 100.0 % | 0.0 % | 0.00 % | 1 |
| 5 | 4 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 220 | 171 | country | 0 | 171 | 4 | 0 | 99.1 % | 0.9 % | 0.00 % | 0 |
| 7 | 18 | 17 | country | 0 | 17 | 0 | 0 | 2.7 % | 97.3 % | 0.00 % | 0 |
| 8 | 173 | 79 | country | 0 | 69 | 0 | 0 | 0.7 % | 99.3 % | 0.00 % | 0 |
| 9 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 10 | 4 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: China (중국), North Korea (조선민주주의인민공화국), Russia (러시아), South Korea (대한민국).

### Level 3

- 1 relation in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 19 relations in the extract, 13 in scope (country). Left out: 6 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `KP` 13; with this country's prefix: 13 of 13.
- Names: name 13, name:en 13, int_name 1, Latin name 0, any English-capable name 13 (of 13).
- Script of `name`: `Hangul` 13.
- `name:*` tags: `name:cs` 13, `name:cy` 13, `name:en` 13, `name:ja` 13, `name:ko` 13, `name:ko-Hani` 13, `name:ru` 13, `name:sk` 13.
- Geometry: the relations add up to 134,669 km² (100.0 % of the outline); 0 land plots lie in none of them, 4 in two or more; 0 relations hold no land plot; land plots per relation: 1,586 to 35,416, median 21,440.
- Rings not closed in the extract (a member way is missing): 1 in scope; 7 of the level's relations could not be assembled at all. In scope: r356540 North Pyongan.
- Examples: Chagang (자강도), Kaesong (개성시), Kangwon (강원도), North Hamgyong (함경북도), North Hwanghae (황해북도), North Pyongan (평안북도), Rason (라선시), Ryanggang (량강도), South Hamgyong (함경남도), South Pyongan (평안남도).

### Level 5

- 4 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 4 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 220 relations in the extract, 171 in scope (country). Left out: 0 with another country's ISO 3166-2, 49 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 171.
- Names: name 171, name:en 171, int_name 4, Latin name 0, any English-capable name 171 (of 171).
- Script of `name`: `Hangul` 171.
- `name:*` tags: `name:en` 171, `name:ko` 171, `name:ko-Hani` 170, `name:zh` 170, `name:zh-Hant` 170, `name:ja` 164, `name:fr` 146, `name:zh-Hans` 144.
- Geometry: the relations add up to 120,227 km² (89.3 % of the outline); 2,023 land plots lie in none of them, 5 in two or more; 0 relations hold no land plot; land plots per relation: 9 to 4,185, median 1,087.
- Rings not closed in the extract (a member way is missing): 0 in scope; 26 of the level's relations could not be assembled at all.
- Examples: Anak County (안악군), Chunggang County (중강군), Hyongjesan District (형제산구역), Kosan County (고산군), Myonggan County (명간군), Pungso County (풍서군), Samjiyon-si (삼지연시), Songyo District (선교구역), Tongdaewon District (동대원구역), Ŭnjŏng District (은정구역).

### Level 7

- 18 relations in the extract, 17 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 17.
- Names: name 17, name:en 17, int_name 0, Latin name 0, any English-capable name 17 (of 17).
- Script of `name`: `Hangul` 17.
- `name:*` tags: `name:en` 17, `name:ja` 17, `name:ko-Hani` 17, `name:zh` 17, `name:zh-Hans` 17, `name:zh-Hant` 17, `name:ko` 16, `name:ko-Latn` 2.
- Geometry: the relations add up to 3,674 km² (2.7 % of the outline); 208,835 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 7 to 1,493, median 103.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Chongam District (청암구역), Haean District (해안구역), Hungdok District (흥덕구역), Phohang District (포항구역), Rajin District (라진구역), Ranam District (라남구역), Sinam District (신암구역), Songchongang District (성천강구역), Sunam District (수남구역), Ungok-chigu (운곡지구).

### Level 8

- 173 relations in the extract, 79 in scope (country). Left out: 0 with another country's ISO 3166-2, 94 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 79.
- Names: name 79, name:en 69, int_name 0, Latin name 0, any English-capable name 69 (of 79).
- Script of `name`: `Hangul` 79.
- `name:*` tags: `name:ko-Hani` 78, `name:ko` 76, `name:zh` 74, `name:zh-Hant` 74, `name:ja` 72, `name:zh-Hans` 72, `name:en` 69, `name:ru` 37.
- Geometry: the relations add up to 812 km² (0.6 % of the outline); 213,013 land plots lie in none of them, 0 in two or more; 28 relations hold no land plot; land plots per relation: 0 to 400, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 80 of the level's relations could not be assembled at all.
- Examples: Ansan 1-dong (안산1동), Haeun 1-dong (해운1동), Kyongrim-dong (경림동), Pip'a 1-dong (비파1동), Pugulgorii 1-dong (붉은거리1동), Ryonhwa-2dong (련화2동), Sinwon-dong (신원동), Taebo-dong (대보동), 고남리, 해선리.

### Level 9

- 3 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).

### Level 10

- 4 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 4 with their centre outside the country (or not assembled and without ISO code).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 13 relations, all with ISO 3166-2 KP-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
- Districts: level 6: 171 relations, none with an ISO 3166-2 code, covering 99.1 % of the country's land plots (0.00 % overlap, 2,023 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:ko-Latn`, `name` (when written in Latin letters). Local name: `name:ko`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 12 relations)

- English name from: `name:en` 12; none for 0.
- Local name from: `name:ko` 12; none for 0. English equals local for 0.
- English prefixes: “South” 3. Suffixes: none.
- Local prefixes: none. Suffixes: “도” 8, “시” 4, “남도” 3.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 1 names, e.g. P'yŏngyang: p-y-ngyang → p-yongyang. Stage 2 should fold accents.

### Districts (level 6, 171 relations)

- English name from: `name:en` 171; none for 0.
- Local name from: `name:ko` 171; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “County” 124, “District” 19.
- Local prefixes: none. Suffixes: “군” 124.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 5 names, e.g. Ryongsŏng District: ryongs-ng-district → ryongsong-district, Samsŏk District: sams-k-district → samsok-district, Ŭnjŏng District: nj-ng-district → unjong-district. Stage 2 should fold accents.

The slug of a new country's province carries the country as a suffix (`-north-korea`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 5,047. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 3,257 | 3,228 | 2,427 |
| hamlet | 736 | 538 | 365 |
| quarter | 613 | 613 | 572 |
| town | 197 | 195 | 183 |
| locality | 125 | 122 | 51 |
| neighbourhood | 63 | 63 | 61 |
| city | 25 | 25 | 25 |
| state | 8 | 8 | 8 |
| suburb | 8 | 8 | 7 |
| island | 7 | 6 | 3 |
| isolated_dwelling | 6 | 2 | 2 |
| country | 1 | 1 | 1 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Pyongyang (population 3,871,335): quarter 225 (221 English-capable), village 9 (9 English-capable), locality 1 (1 English-capable).
- Nampo (population 983,660): quarter 23 (23 English-capable), village 5 (5 English-capable).
- Hamhung (population 768,551): neighbourhood 43 (42 English-capable), village 27 (23 English-capable), suburb 1 (1 English-capable).
- Chŏngjin (population 667,929): quarter 41 (36 English-capable), village 8 (8 English-capable), suburb 4 (3 English-capable).
- Tanchon (population 345,875): village 16 (0 English-capable), quarter 11 (0 English-capable), town 1 (1 English-capable).

## Size

- Land plots: 214,569. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 8 s and 197 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// North Korea: from north-korea-latest.osm.pbf
{Code: "kp", Extract: "north-korea",
	Province: Rule{AdminLevel: 4, ISOPrefix: "KP", Count: 13, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:ko-Latn", "name"}, LocalLang: "ko",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, []string{"도"}}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, []string{"County"}}, Local: [2][]string{nil, []string{"군"}}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('kp', '{quarter,neighbourhood}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
