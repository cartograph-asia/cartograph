# Country scan: Taiwan

Written by `go run ./cmd/countryscan -country tw` from `taiwan-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 22 relations, all with ISO 3166-2 TW-xx, covering 100.0 % of the country's land plots (0.01 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 9.
- **Districts:** level 9: 6,861 relations, none with an ISO 3166-2 code, covering 99.6 % of the country's land plots (0.19 % overlap, 212 plots in none). Per province: 22 to 1036, median 264; 2 districts have their centre in no province.
- **Land plots:** 51,028.
- **Names (districts):** 2 without an English-capable name, 2 with an empty slug, 537 slugs used twice, 0 names whose slug loses letters without accent folding.
- Provinces whose rings do not close in the extract: 1; without a land plot: 0.

## Outline and land

- Outline: the union of 21 admin_level 4 relations with ISO 3166-2 TW-xx (the country relation is not whole in the extract); 1 more of them are not whole either and are missing from it, so the outline and the land plots are too small; 35,768 km² in 122 polygons.
- Cells at resolution 8 with their centre inside the outline: 54,409; 51,028 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **51,028**, about 37,607 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with TW-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 2 | 2 | extract | 0 | 2 | 2 | 0 | – | – | – | – |
| 4 | 23 | 22 | country | 22 | 22 | 0 | 0 | 100.0 % | 0.0 % | 0.01 % | 1 |
| 5 | 7 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 13 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 7 | 170 | 131 | country | 0 | 131 | 0 | 0 | 24.8 % | 75.2 % | 0.01 % | 0 |
| 8 | 201 | 197 | country | 0 | 196 | 1 | 0 | 74.8 % | 25.2 % | 0.02 % | 0 |
| 9 | 7,781 | 6,861 | country | 0 | 6,859 | 0 | 0 | 99.6 % | 0.4 % | 0.19 % | 0 |
| 10 | 3 | 3 | country | 0 | 3 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: China (中国), Taiwan (臺灣).

### Level 4

- 23 relations in the extract, 22 in scope (country). Left out: 1 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `TW` 22; with this country's prefix: 22 of 22.
- Names: name 22, name:en 22, int_name 0, Latin name 0, any English-capable name 22 (of 22).
- Script of `name`: `Han` 22.
- `name:*` tags: `name:cs` 22, `name:en` 22, `name:ja` 22, `name:ko` 22, `name:nan` 22, `name:nan-Hant` 22, `name:nan-Latn-pehoeji` 22, `name:nan-Latn-tailo` 22.
- Geometry: the relations add up to 35,768 km² (100.0 % of the outline); 0 land plots lie in none of them, 4 in two or more; 0 relations hold no land plot; land plots per relation: 48 to 6,954, median 2,176.
- Rings not closed in the extract (a member way is missing): 1 in scope; 2 of the level's relations could not be assembled at all. In scope: r2127079 Kaohsiung.
- Examples: Changhua County (彰化縣), Chiayi County (嘉義縣), Hsinchu County (新竹縣), Keelung (基隆市), Lienchiang County (連江縣), Nantou County (南投縣), Pingtung County (屏東縣), Tainan (臺南市), Taitung County (臺東縣), Yunlin County (雲林縣).

### Level 5

- 7 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 7 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 13 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 13 with their centre outside the country (or not assembled and without ISO code).

### Level 7

- 170 relations in the extract, 131 in scope (country). Left out: 0 with another country's ISO 3166-2, 39 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 131.
- Names: name 131, name:en 131, int_name 0, Latin name 0, any English-capable name 131 (of 131).
- Script of `name`: `Han` 131.
- `name:*` tags: `name:en` 131, `name:hak` 131, `name:ja` 131, `name:ko` 131, `name:nan` 131, `name:nan-Hant` 131, `name:nan-Latn-pehoeji` 131, `name:nan-Latn-tailo` 131.
- Geometry: the relations add up to 8,300 km² (23.2 % of the outline); 38,374 land plots lie in none of them, 6 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 1,547, median 66.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Anding District (安定區), Dadu District (大肚區), Fuxing District (復興區), Liuying District (柳營區), North District (北區), Shanhua District (善化區), Taishan District (泰山區), Wuqi District (梧棲區), Xitun District (西屯區), Zuozhen District (左鎮區).

### Level 8

- 201 relations in the extract, 197 in scope (country). Left out: 0 with another country's ISO 3166-2, 4 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 197.
- Names: name 197, name:en 196, int_name 1, Latin name 0, any English-capable name 196 (of 197).
- Script of `name`: `Han` 196, `Latin` 1.
- `name:*` tags: `name:en` 196, `name:hak` 196, `name:ko` 196, `name:nan` 196, `name:zh` 196, `name:ja` 195, `name:nan-Latn-pehoeji` 195, `name:nan-Latn-tailo` 195.
- Geometry: the relations add up to 25,590 km² (71.5 % of the outline); 12,849 land plots lie in none of them, 11 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 2,466, median 98.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Alishan Township (阿里山鄉), Dabi Township (大埤鄉), Fangshan (枋山鄉), Huatan Township (花壇鄉), Linluo (麟洛鄉), Nanzhuang (南庄鄉), Shuilin Township (水林鄉), Wujie (五結鄉), Yilan City (宜蘭市), 大坵嶼.

### Level 9

- 7,781 relations in the extract, 6,861 in scope (country). Left out: 0 with another country's ISO 3166-2, 918 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 6,861.
- Names: name 6,861, name:en 6,859, int_name 0, Latin name 0, any English-capable name 6,859 (of 6,861).
- Script of `name`: `Han` 6,861.
- `name:*` tags: `name:zh` 6,861, `name:en` 6,859, `name:zh-Hans` 2,382, `name:zh-Hant` 2,382, `name:ko` 1,820, `name:zh-Latn-pinyin` 933, `name:zh-Bopo` 786, `name:nan` 721.
- Geometry: the relations add up to 33,907 km² (94.8 % of the outline); 212 land plots lie in none of them, 96 in two or more; 1763 relations hold no land plot; land plots per relation: 0 to 804, median 2.
- Rings not closed in the extract (a member way is missing): 0 in scope; 4 of the level's relations could not be assembled at all.
- Examples: Adan Village (阿丹里), Dashi Village (大石村), Futian Village (福田里), Huliao Village (湖寮村), Longtian Village (龍田村), Qingshui Village (清水里), Tai'an Village (泰安里), Xincheng Village (新城村), Yongxing Village (永興村), 興安里.

### Level 10

- 3 relations in the extract, 3 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 3.
- Names: name 3, name:en 3, int_name 0, Latin name 0, any English-capable name 3 (of 3).
- Script of `name`: `Han` 3.
- `name:*` tags: `name:en` 3, `name:zh` 3.
- Geometry: the relations add up to 2 km² (0.0 % of the outline); 51,025 land plots lie in none of them, 0 in two or more; 2 relations hold no land plot; land plots per relation: 0 to 3, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Neighborhood 12, Yijia Village (一甲里12鄰), Neighborhood 13, Yijia Village (一甲里13鄰), Neighborhood 5, Guishan Village (龜山里5鄰).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 22 relations, all with ISO 3166-2 TW-xx, covering 100.0 % of the country's land plots (0.01 % overlap, 0 plots in none).
- Districts: level 9: 6,861 relations, none with an ISO 3166-2 code, covering 99.6 % of the country's land plots (0.19 % overlap, 212 plots in none).

Levels that qualify: 4, 9. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:zh-Latn`, `name` (when written in Latin letters). Local name: `name:zh`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 21 relations)

- English name from: `name:en` 21; none for 0.
- Local name from: `name:zh` 21; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “County” 11.
- Local prefixes: none. Suffixes: “縣” 13, “市” 8.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 9, 6,861 relations)

- English name from: `name:en` 6,859; none for 2.
- Local name from: `name:zh` 6,861; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “Village” 6858.
- Local prefixes: none. Suffixes: “里” 4976, “村” 1885.
- Slugs: 2 names have no slug (no English-capable name, or nothing in it that a slug can hold); 537 slugs are used more than once: 0/fucheng-village (2), 0/fuhua-village (2), 0/futai-village (2), 0/fuxing-village (2), 0/jingxing-village (2), 0/jintai-village (2), 0/linjiang-village (2), 0/xin-an-village (2), 0/xinfu-village (2), 0/xingde-village (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-taiwan`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 16,303. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| hamlet | 13,229 | 13,227 | 11,772 |
| village | 1,796 | 1,796 | 1,147 |
| locality | 537 | 536 | 315 |
| town | 244 | 244 | 244 |
| suburb | 203 | 202 | 165 |
| isolated_dwelling | 136 | 133 | 110 |
| neighbourhood | 62 | 62 | 55 |
| square | 25 | 22 | 2 |
| islet | 17 | 8 | 0 |
| county | 15 | 15 | 15 |
| farm | 11 | 9 | 0 |
| quarter | 10 | 10 | 6 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- New Taipei (population 3,979,208): hamlet 245 (137 English-capable), locality 40 (17 English-capable), village 39 (31 English-capable), suburb 20 (20 English-capable), square 3 (0 English-capable), neighbourhood 2 (2 English-capable).
- Taichung (population 2,767,239): hamlet 125 (116 English-capable), suburb 15 (12 English-capable), village 11 (8 English-capable), square 2 (0 English-capable).
- Taipei (population 2,695,704): hamlet 431 (208 English-capable), village 44 (41 English-capable), locality 42 (21 English-capable), suburb 26 (23 English-capable), square 5 (2 English-capable), quarter 3 (3 English-capable).
- Taoyuan (population 2,147,763): hamlet 246 (158 English-capable), village 15 (9 English-capable), suburb 10 (8 English-capable), town 4 (4 English-capable), locality 2 (1 English-capable).
- Tainan (population 1,886,033): hamlet 38 (3 English-capable), village 27 (3 English-capable), locality 19 (0 English-capable), suburb 13 (6 English-capable), neighbourhood 5 (1 English-capable), square 5 (0 English-capable).

## Size

- Land plots: 51,028. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
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
  ('tw', '{quarter,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
