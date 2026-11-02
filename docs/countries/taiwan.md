# Country scan: Taiwan

Written by `go run ./cmd/countryscan -country tw` from `taiwan-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Boundaries completed with the ways and nodes of the neighbours' cuts `china-latest.osm.pbf.cut.osm.pbf`, `philippines-latest.osm.pbf.cut.osm.pbf` (relations and places from the own file only). Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 22 relations, all with ISO 3166-2 TW-xx, covering 99.9 % of the country's land plots (0.01 % overlap, 32 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 9.
- **Districts:** level 9: 7,776 relations, none with an ISO 3166-2 code, covering 99.6 % of the country's land plots (0.19 % overlap, 214 plots in none). Per province: 22 to 1038, median 277; 3 districts have their centre in no province.
- **Land plots:** 55,564.
- **Names (districts):** 2 without an English-capable name, 2 with an empty slug, 612 slugs used twice, 0 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r449220 (admin_level 2); 93,757 km² in 13 polygons.
- Cells at resolution 8 with their centre inside the outline: 141,879; 55,564 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **55,564**, about 40,950 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with TW-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 2 | 2 | extract | 0 | 2 | 2 | 0 | – | – | – | – |
| 4 | 23 | 22 | country | 22 | 22 | 0 | 0 | 99.9 % | 0.1 % | 0.01 % | 0 |
| 5 | 7 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 13 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 7 | 170 | 170 | country | 0 | 170 | 0 | 0 | 31.0 % | 69.0 % | 0.02 % | 0 |
| 8 | 201 | 199 | country | 0 | 198 | 1 | 0 | 68.9 % | 31.1 % | 0.02 % | 0 |
| 9 | 7,781 | 7,776 | country | 0 | 7,774 | 0 | 0 | 99.6 % | 0.4 % | 0.19 % | 0 |
| 10 | 3 | 3 | country | 0 | 3 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: China (中国), Taiwan (臺灣).

### Level 4

- 23 relations in the extract, 22 in scope (country). Left out: 1 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `TW` 22; with this country's prefix: 22 of 22.
- Names: name 22, name:en 22, int_name 0, Latin name 0, any English-capable name 22 (of 22).
- Script of `name`: `Han` 22.
- `name:*` tags: `name:cs` 22, `name:en` 22, `name:ja` 22, `name:ko` 22, `name:nan` 22, `name:nan-Hant` 22, `name:nan-Latn-pehoeji` 22, `name:nan-Latn-tailo` 22.
- Geometry: the relations add up to 42,375 km² (45.2 % of the outline); 32 land plots lie in none of them, 4 in two or more; 0 relations hold no land plot; land plots per relation: 48 to 6,954, median 2,796.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Changhua County (彰化縣), Chiayi County (嘉義縣), Hsinchu County (新竹縣), Keelung (基隆市), Lienchiang County (連江縣), Nantou County (南投縣), Pingtung County (屏東縣), Tainan (臺南市), Taitung County (臺東縣), Yunlin County (雲林縣).

### Level 5

- 7 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 7 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 13 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 13 with their centre outside the country (or not assembled and without ISO code).

### Level 7

- 170 relations in the extract, 170 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 170.
- Names: name 170, name:en 170, int_name 0, Latin name 0, any English-capable name 170 (of 170).
- Script of `name`: `Han` 170.
- `name:*` tags: `name:en` 170, `name:hak` 170, `name:ja` 170, `name:ko` 170, `name:nan` 170, `name:nan-Hant` 170, `name:nan-Latn-pehoeji` 170, `name:nan-Latn-tailo` 170.
- Geometry: the relations add up to 14,967 km² (16.0 % of the outline); 38,320 land plots lie in none of them, 9 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 1,547, median 65.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Alian District (阿蓮區), Daliao District (大寮區), Guanmiao District (關廟區), Liugui District (六龜區), Neihu District (內湖區), Ren'ai (仁愛區), Shulin District (樹林區), Wufeng District (霧峰區), Xitun District (西屯區), Zuozhen District (左鎮區).

### Level 8

- 201 relations in the extract, 199 in scope (country). Left out: 0 with another country's ISO 3166-2, 2 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 199.
- Names: name 199, name:en 198, int_name 1, Latin name 0, any English-capable name 198 (of 199).
- Script of `name`: `Han` 198, `Latin` 1.
- `name:*` tags: `name:en` 198, `name:hak` 198, `name:ko` 198, `name:nan` 198, `name:zh` 198, `name:ja` 197, `name:nan-Latn-pehoeji` 197, `name:nan-Latn-tailo` 197.
- Geometry: the relations add up to 25,640 km² (27.3 % of the outline); 17,305 land plots lie in none of them, 11 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 2,466, median 97.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Alishan Township (阿里山鄉), Dacheng Township (大城鄉), Fangyuan Township (芳苑鄉), Hukou (湖口鄉), Linnei Township (林內鄉), Neipu (內埔鄉), Shuishang Township (水上鄉), Wujie (五結鄉), Yilan City (宜蘭市), 大坵嶼.

### Level 9

- 7,781 relations in the extract, 7,776 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 7,776.
- Names: name 7,776, name:en 7,774, int_name 0, Latin name 0, any English-capable name 7,774 (of 7,776).
- Script of `name`: `Han` 7,776.
- `name:*` tags: `name:zh` 7,776, `name:en` 7,774, `name:zh-Hans` 2,427, `name:zh-Hant` 2,427, `name:ko` 2,013, `name:zh-Latn-pinyin` 972, `name:zh-Bopo` 821, `name:nan` 768.
- Geometry: the relations add up to 40,545 km² (43.2 % of the outline); 214 land plots lie in none of them, 104 in two or more; 2162 relations hold no land plot; land plots per relation: 0 to 804, median 2.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Adan Village (阿丹里), Dapu Village (大埔里), Fushan Village (福山里), Hucuo Village (胡厝里), Longping Village (龍平里), Qinglian Village (清蓮里), Songya Village (松雅里), Xincen Village (新岑里), Yongzhen Village (永鎮村), 興安里.

### Level 10

- 3 relations in the extract, 3 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 3.
- Names: name 3, name:en 3, int_name 0, Latin name 0, any English-capable name 3 (of 3).
- Script of `name`: `Han` 3.
- `name:*` tags: `name:en` 3, `name:zh` 3.
- Geometry: the relations add up to 2 km² (0.0 % of the outline); 55,561 land plots lie in none of them, 0 in two or more; 2 relations hold no land plot; land plots per relation: 0 to 3, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Neighborhood 12, Yijia Village (一甲里12鄰), Neighborhood 13, Yijia Village (一甲里13鄰), Neighborhood 5, Guishan Village (龜山里5鄰).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 22 relations, all with ISO 3166-2 TW-xx, covering 99.9 % of the country's land plots (0.01 % overlap, 32 plots in none).
- Districts: level 9: 7,776 relations, none with an ISO 3166-2 code, covering 99.6 % of the country's land plots (0.19 % overlap, 214 plots in none).

Levels that qualify: 4, 9. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:zh-Latn`, `name` (when written in Latin letters). Local name: `name:zh`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 22 relations)

- English name from: `name:en` 22; none for 0.
- Local name from: `name:zh` 22; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “County” 11.
- Local prefixes: none. Suffixes: “縣” 13, “市” 9.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 9, 7,776 relations)

- English name from: `name:en` 7,774; none for 2.
- Local name from: `name:zh` 7,776; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “Village” 7770.
- Local prefixes: none. Suffixes: “里” 5882.
- Slugs: 2 names have no slug (no English-capable name, or nothing in it that a slug can hold); 612 slugs are used more than once: 0/fucheng-village (2), 0/fuhua-village (2), 0/futai-village (2), 0/fuxing-village (2), 0/jingxing-village (2), 0/jintai-village (2), 0/linjiang-village (2), 0/xinan-village (2), 0/xinfu-village (2), 0/xingde-village (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-taiwan`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 17,709. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| hamlet | 14,034 | 14,032 | 12,544 |
| village | 1,958 | 1,958 | 1,272 |
| locality | 700 | 699 | 429 |
| suburb | 322 | 321 | 267 |
| town | 245 | 245 | 245 |
| neighbourhood | 182 | 182 | 145 |
| isolated_dwelling | 145 | 142 | 118 |
| quarter | 31 | 31 | 15 |
| square | 27 | 24 | 2 |
| islet | 19 | 10 | 2 |
| county | 15 | 15 | 15 |
| farm | 11 | 9 | 0 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- New Taipei (population 3,979,208): hamlet 245 (137 English-capable), locality 40 (17 English-capable), village 39 (31 English-capable), suburb 20 (20 English-capable), square 3 (0 English-capable), neighbourhood 2 (2 English-capable).
- Kaohsiung (population 2,779,371): suburb 69 (56 English-capable), neighbourhood 34 (14 English-capable), locality 21 (15 English-capable), quarter 21 (9 English-capable), village 8 (2 English-capable), hamlet 3 (3 English-capable).
- Taichung (population 2,767,239): hamlet 125 (116 English-capable), suburb 15 (12 English-capable), village 11 (8 English-capable), square 2 (0 English-capable).
- Taipei (population 2,695,704): hamlet 431 (208 English-capable), village 44 (41 English-capable), locality 42 (21 English-capable), suburb 26 (23 English-capable), square 5 (2 English-capable), quarter 3 (3 English-capable).
- Taoyuan (population 2,147,763): hamlet 246 (158 English-capable), village 15 (9 English-capable), suburb 10 (8 English-capable), town 4 (4 English-capable), locality 2 (1 English-capable).

## Size

- Land plots: 55,564. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 2 s and 100 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Taiwan: from taiwan-latest.osm.pbf
{Code: "tw", Extract: "taiwan",
	Province: Rule{AdminLevel: 4, ISOPrefix: "TW", Count: 22, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 9},
	EnglishName: []string{"name:en", "int_name", "name:zh-Latn", "name"}, LocalLang: "zh-Hant",
	StripProvince: Strip{English: [2][]string{nil, []string{"County"}}, Local: [2][]string{nil, []string{"縣"}}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, []string{"Village"}}, Local: [2][]string{nil, []string{"里"}}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('tw', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
