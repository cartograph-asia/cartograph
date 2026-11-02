# Country scan: China

Written by `go run ./cmd/countryscan -country cn` from `china-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 31 relations, all with ISO 3166-2 CN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 5, 6.
- **Districts:** level 5: 316 relations, none with an ISO 3166-2 code, covering 97.2 % of the country's land plots (0.00 % overlap, 339,951 plots in none). Per province: 0 to 21, median 13; 1 districts have their centre in no province.
- **Land plots:** 11,976,594.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 2 names whose slug loses letters without accent folding.
- Level 8 has 19,789 relations, more than the 8,000 the scan builds geometry for: counted and read for names only.
- Level 10 has 9,379 relations, more than the 8,000 the scan builds geometry for: counted and read for names only.
- Provinces whose rings do not close in the extract: 3; without a land plot: 0.

## Outline and land

- Outline: the union of 28 admin_level 4 relations with ISO 3166-2 CN-xx (the country relation is not whole in the extract); 3 more of them are not whole either and are missing from it, so the outline and the land plots are too small; 8,295,078 km² in 78 polygons.
- Cells at resolution 8 with their centre inside the outline: 12,384,268; 11,976,594 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **11,976,594**, about 8,826,749 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with CN-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 16 | 16 | extract | 0 | 16 | 9 | 2 | – | – | – | – |
| 3 | 8 | 2 | country | 2 | 2 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 96 | 31 | country | 31 | 31 | 26 | 0 | 100.0 % | 0.0 % | 0.00 % | 3 |
| 5 | 365 | 316 | country | 0 | 316 | 21 | 0 | 97.2 % | 2.8 % | 0.00 % | 0 |
| 6 | 3,188 | 2,674 | country | 0 | 2,667 | 39 | 0 | 99.9 % | 0.1 % | 0.03 % | 0 |
| 7 | 238 | 153 | country | 0 | 56 | 0 | 0 | 0.8 % | 99.2 % | 0.01 % | 0 |
| 8 | 19,789 | 19,789 | extract | 0 | 11,131 | 42 | 28 | – | – | – | – |
| 9 | 394 | 288 | country | 0 | 43 | 0 | 0 | 0.2 % | 99.8 % | 0.00 % | 0 |
| 10 | 9,379 | 9,379 | extract | 0 | 1,234 | 0 | 5 | – | – | – | – |

### Level 2

Countries in the extract: Afghanistan (阿富汗), Bhutan (不丹), India (印度), Kyrgyzstan (吉尔吉斯斯坦), Laos (老挝;寮國), Myanmar (缅甸), North Korea (朝鲜), Pakistan (巴基斯坦), Taiwan (臺灣), Vietnam (越南).

### Level 3

- 8 relations in the extract, 2 in scope (country). Left out: 0 with another country's ISO 3166-2, 6 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `CN` 2; with this country's prefix: 2 of 2.
- Names: name 2, name:en 2, int_name 0, Latin name 0, any English-capable name 2 (of 2).
- Script of `name`: `Latin` 2.
- `name:*` tags: `name:af` 2, `name:am` 2, `name:an` 2, `name:ar` 2, `name:as` 2, `name:az` 2, `name:ba` 2, `name:be` 2.
- Geometry: the relations add up to 2,885 km² (0.0 % of the outline); 11,976,594 land plots lie in none of them, 0 in two or more; 2 relations hold no land plot; land plots per relation: 0 to 0, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 6 of the level's relations could not be assembled at all.
- Examples: Hong Kong (香港), Macau (澳門).

### Level 4

- 96 relations in the extract, 31 in scope (country). Left out: 63 with another country's ISO 3166-2, 2 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `CN` 31; with this country's prefix: 31 of 31.
- Names: name 31, name:en 31, int_name 26, Latin name 0, any English-capable name 31 (of 31).
- Script of `name`: `Han` 28, `other` 2, `Arabic` 1.
- `name:*` tags: `name:af` 31, `name:ar` 31, `name:bg` 31, `name:bn` 31, `name:bo` 31, `name:br` 31, `name:ca` 31, `name:cs` 31.
- Geometry: the relations add up to 8,295,078 km² (100.0 % of the outline); 0 land plots lie in none of them, 7 in two or more; 0 relations hold no land plot; land plots per relation: 13,207 to 1,958,373, median 300,133.
- Rings not closed in the extract (a member way is missing): 3 in scope; 67 of the level's relations could not be assembled at all. In scope: r153292 Xizang, r553303 Fujian, r2128285 Hainan.
- Examples: Anhui (安徽省), Fujian (福建省), Guangxi (广西壮族自治区), Heilongjiang (黑龙江省), Hunan (湖南省), Jiangxi (江西省), Qinghai (青海省), Shanghai (上海市), Tianjin (天津市), Zhejiang (浙江省).

### Level 5

- 365 relations in the extract, 316 in scope (country). Left out: 0 with another country's ISO 3166-2, 49 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 316.
- Names: name 316, name:en 316, int_name 21, Latin name 0, any English-capable name 316 (of 316).
- Script of `name`: `Han` 282, `other` 23, `Arabic` 10, `Latin` 1.
- `name:*` tags: `name:en` 316, `name:zh` 316, `name:ko` 315, `name:zh-Hant` 314, `name:cs` 311, `name:ja` 311, `name:uk` 264, `name:zh-Latn-pinyin` 247.
- Geometry: the relations add up to 8,005,683 km² (96.5 % of the outline); 339,951 land plots lie in none of them, 63 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 571,746, median 21,032.
- Rings not closed in the extract (a member way is missing): 0 in scope; 32 of the level's relations could not be assembled at all.
- Examples: Aksu Prefecture (阿克苏地区), Chaozhou (潮州市), Guangyuan (广元市), Huangshan (黄山市), Laibin (来宾市), Nanyang (南阳市), Shaoyang (邵阳市), Wuhai City (乌海市), Yichang (宜昌市), Ürümqi (乌鲁木齐市).

### Level 6

- 3,188 relations in the extract, 2,674 in scope (country). Left out: 0 with another country's ISO 3166-2, 514 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2,674.
- Names: name 2,674, name:en 2,667, int_name 39, Latin name 0, any English-capable name 2,667 (of 2,674).
- Script of `name`: `Han` 2,421, `other` 167, `Arabic` 86.
- `name:*` tags: `name:zh-Hant` 2,670, `name:en` 2,667, `name:zh` 2,658, `name:ko` 2,632, `name:zh-Latn-pinyin` 2,238, `name:zh-Hans` 1,188, `name:ja` 1,134, `name:ru` 935.
- Geometry: the relations add up to 8,257,303 km² (99.5 % of the outline); 11,434 land plots lie in none of them, 3,276 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 247,488, median 2,460.
- Rings not closed in the extract (a member way is missing): 0 in scope; 290 of the level's relations could not be assembled at all.
- Examples: Abag Banner (阿巴嘎旗), Dayu County (大余县), Gulou District (鼓楼区), Jiaxiang County (嘉祥县), Liuba County (留坝县), Pingba (平坝区), Shixing County (始兴县), Wuji County (无极县), Yicheng City (宜城市), 霍城垦区.

### Level 7

- 238 relations in the extract, 153 in scope (country). Left out: 0 with another country's ISO 3166-2, 85 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 153.
- Names: name 153, name:en 56, int_name 0, Latin name 0, any English-capable name 56 (of 153).
- Script of `name`: `Han` 152, `Myanmar` 1.
- `name:*` tags: `name:zh` 79, `name:ko` 70, `name:en` 56, `name:zh-Hant` 36, `name:zh-Hans` 21, `name:ja` 8, `name:zh-Latn-pinyin` 6, `name:nan` 2.
- Geometry: the relations add up to 96,026 km² (1.2 % of the outline); 11,882,417 land plots lie in none of them, 904 in two or more; 0 relations hold no land plot; land plots per relation: 2 to 6,503, median 259.
- Rings not closed in the extract (a member way is missing): 0 in scope; 69 of the level's relations could not be assembled at all.
- Examples: 12th Division of Xinjiang Production and Construction Corps (第十二师), Guangdong-Macao In-Depth Cooperation Zone in Hengqin (横琴粤澳深度合作区), Lutai Economic Development Zone (芦台经济开发区), Xiaolan Economic Development Area (小蓝经开区), 信阳高新技术产业开发区, 官庄工区, 承德高新技术产业开发区, 淮安经济技术开发区, 进贤经开区, 龙虎山区.

### Level 8

- 19,789 relations in the extract, 19,789 in scope (extract). Left out: 0 with another country's ISO 3166-2.
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 19,789.
- Names: name 19,784, name:en 11,131, int_name 42, Latin name 28, any English-capable name 11,135 (of 19,789).
- Script of `name`: `Han` 18,792, `other` 466, `Arabic` 453, `Cyrillic` 37, `Latin` 33.
- `name:*` tags: `name:zh` 13,255, `name:en` 11,131, `name:zh-Latn-pinyin` 5,829, `name:zh-Hant` 3,089, `name:ru` 2,547, `name:ja` 2,337, `name:zh-Hans` 2,274, `name:bo` 889.
- Examples: 147th Regiment (一四七团), Fengchunling (逢春岭乡), Kaishantun Town (开山屯镇), Qinglung township (青龙乡), Wufu (五夫镇), Zhoushang (洲上乡), 哈西街道办事处, 暮云街道, 红亮垦殖场, 𨻧隍镇.

### Level 9

- 394 relations in the extract, 288 in scope (country). Left out: 0 with another country's ISO 3166-2, 106 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 288.
- Names: name 288, name:en 43, int_name 0, Latin name 0, any English-capable name 43 (of 288).
- Script of `name`: `Han` 288.
- `name:*` tags: `name:zh` 143, `name:en` 43, `name:zh-Hant` 34, `name:zh-Latn-pinyin` 21, `name:zh-Hans` 13, `name:ko` 9, `name:ru` 5, `name:ja` 2.
- Geometry: the relations add up to 17,474 km² (0.2 % of the outline); 11,948,484 land plots lie in none of them, 27 in two or more; 10 relations hold no land plot; land plots per relation: 0 to 2,547, median 29.
- Rings not closed in the extract (a member way is missing): 0 in scope; 54 of the level's relations could not be assembled at all.
- Examples: Aksay Industrial Park (阿克塞工业园区), Shizijie Subdistrict (原十字街街道), 亨通村, 哈思山林场, 山丹马场二场, 林海林场, 清河办事处, 红沙窝林场, 金塔工业集中区, 龙王办事处.

### Level 10

- 9,379 relations in the extract, 9,379 in scope (extract). Left out: 0 with another country's ISO 3166-2.
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 9,379.
- Names: name 9,376, name:en 1,234, int_name 0, Latin name 5, any English-capable name 1,239 (of 9,379).
- Script of `name`: `Han` 9,281, `Hangul` 58, `Myanmar` 19, `Latin` 13, `Arabic` 5.
- `name:*` tags: `name:zh` 1,427, `name:en` 1,234, `name:zh-Hant` 633, `name:zh-Hans` 579, `name:ko` 143, `name:zh-Latn-pinyin` 115, `name:yue-Latn` 74, `name:yue` 48.
- Examples: 13街区, Xinsheng (新盛社区), 二郎庙社区, 厚余村, 富饶村委会, 新都社区, 洋港村, 红岭花园社区, 通盛社区, 구룡촌 (九龙村).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 31 relations, all with ISO 3166-2 CN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
- Districts: level 5: 316 relations, none with an ISO 3166-2 code, covering 97.2 % of the country's land plots (0.00 % overlap, 339,951 plots in none).

Levels that qualify: 4, 5, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:zh-Latn`, `name` (when written in Latin letters). Local name: `name:zh`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 28 relations)

- English name from: `name:en` 28; none for 0.
- Local name from: `name:zh` 28; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: “省” 20.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 5, 316 relations)

- English name from: `name:en` 316; none for 0.
- Local name from: `name:zh` 316; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “City” 85.
- Local prefixes: none. Suffixes: “市” 275.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 2 names, e.g. Ürümqi: r-mqi → urumqi, Lüliang City: l-liang-city → luliang-city. Stage 2 should fold accents.

The slug of a new country's province carries the country as a suffix (`-china`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 576,017. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 294,083 | 277,431 | 146,336 |
| hamlet | 191,261 | 187,648 | 81,513 |
| town | 30,454 | 30,451 | 29,988 |
| locality | 21,419 | 18,482 | 2,255 |
| quarter | 12,331 | 12,331 | 3,743 |
| suburb | 9,451 | 9,449 | 8,101 |
| neighbourhood | 8,014 | 7,983 | 2,555 |
| city | 2,968 | 2,968 | 2,965 |
| isolated_dwelling | 2,860 | 2,420 | 127 |
| county | 1,115 | 1,115 | 1,074 |
| square | 568 | 521 | 41 |
| district | 480 | 480 | 476 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Shanghai (population 24,870,895): quarter 119 (106 English-capable), suburb 63 (63 English-capable), neighbourhood 28 (10 English-capable), city 8 (8 English-capable), locality 3 (2 English-capable), hamlet 2 (0 English-capable).
- Beijing (population 21,893,095): quarter 295 (1 English-capable), suburb 59 (59 English-capable), neighbourhood 25 (3 English-capable), village 15 (6 English-capable), city 3 (3 English-capable), hamlet 3 (0 English-capable).
- Chengdu (population 20,937,757): quarter 95 (93 English-capable), suburb 43 (43 English-capable), neighbourhood 42 (0 English-capable), locality 7 (0 English-capable), city 4 (4 English-capable), hamlet 2 (2 English-capable).
- Shenzhen (population 17,560,061): quarter 154 (153 English-capable), neighbourhood 42 (13 English-capable), suburb 21 (21 English-capable), square 11 (10 English-capable), hamlet 10 (6 English-capable), locality 8 (3 English-capable).
- Guangzhou (population 14,043,500): suburb 66 (62 English-capable), neighbourhood 51 (27 English-capable), locality 33 (18 English-capable), village 22 (15 English-capable), quarter 19 (14 English-capable), hamlet 18 (13 English-capable).

## Size

- Land plots: 11,976,594. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 500 s and 11,007 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// China: from china-latest.osm.pbf
{Code: "cn", Extract: "china",
	Province: Rule{AdminLevel: 4, ISOPrefix: "CN", Count: 31, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 5},
	EnglishName: []string{"name:en", "int_name", "name:zh-Latn", "name"}, LocalLang: "zh",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, []string{"省"}}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, []string{"市"}}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('cn', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
