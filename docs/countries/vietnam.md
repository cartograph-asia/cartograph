# Country scan: Vietnam

Written by `go run ./cmd/countryscan -country vn` from `vietnam-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 34 relations, all with ISO 3166-2 VN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 3,254 relations, none with an ISO 3166-2 code, covering 99.9 % of the country's land plots (0.03 % overlap, 501 plots in none). Per province: 38 to 167, median 99; 0 districts have their centre in no province.
- **Land plots:** 390,593.
- **Names (provinces):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 29 names whose slug loses letters without accent folding.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 8 slugs used twice, 3020 names whose slug loses letters without accent folding.
- Provinces whose rings do not close in the extract: 1; without a land plot: 0.

## Outline and land

- Outline: the union of 33 admin_level 4 relations with ISO 3166-2 VN-xx (the country relation is not whole in the extract); 1 more of them are not whole either and are missing from it, so the outline and the land plots are too small; 480,148 km² in 34 polygons.
- Cells at resolution 8 with their centre inside the outline: 579,113; 390,593 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **390,593**, about 287,867 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with VN-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 5 | 5 | extract | 0 | 5 | 4 | 2 | – | – | – | – |
| 3 | 1 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 59 | 34 | country | 34 | 34 | 0 | 34 | 100.0 % | 0.0 % | 0.00 % | 1 |
| 5 | 7 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 3,395 | 3,254 | country | 0 | 1,682 | 0 | 3,254 | 99.9 % | 0.1 % | 0.03 % | 0 |
| 8 | 74 | 18 | country | 0 | 0 | 0 | 18 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 9 | 5,703 | 5,648 | country | 0 | 0 | 0 | 5,648 | 5.4 % | 94.6 % | 0.01 % | 0 |
| 10 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Cambodia (Campuchia), China (Trung Quốc), Laos (Lào), Philippines, Vietnam (Việt Nam).

### Level 3

- 1 relation in the extract, 0 in scope (country). Left out: 1 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 59 relations in the extract, 34 in scope (country). Left out: 25 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `VN` 34; with this country's prefix: 34 of 34.
- Names: name 34, name:en 34, int_name 0, Latin name 34, any English-capable name 34 (of 34).
- Script of `name`: `Latin` 34.
- `name:*` tags: `name:en` 34, `name:ko` 34, `name:zh` 34, `name:ja` 32, `name:ru` 28, `name:zh-Hant` 28, `name:bg` 25, `name:zh-Hans` 25.
- Geometry: the relations add up to 480,148 km² (100.0 % of the outline); 0 land plots lie in none of them, 10 in two or more; 0 relations hold no land plot; land plots per relation: 3,111 to 29,134, median 10,514.
- Rings not closed in the extract (a member way is missing): 1 in scope; 26 of the level's relations could not be assembled at all. In scope: r1887959 Khánh Hòa Province.
- Examples: An Giang Province (Tỉnh An Giang), Cà Mau Province (Tỉnh Cà Mau), Huế (Thành phố Huế), Hải Phòng (Thành phố Hải Phòng), Lào Cai Province (Tỉnh Lào Cai), Ninh Bình Province (Tỉnh Ninh Bình), Quảng Trị Province (Tỉnh Quảng Trị), Thái Nguyên Province (Tỉnh Thái Nguyên), Điện Biên Province (Tỉnh Điện Biên), Đồng Tháp Province (Tỉnh Đồng Tháp).

### Level 5

- 7 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 7 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 3,395 relations in the extract, 3,254 in scope (country). Left out: 0 with another country's ISO 3166-2, 141 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 3,254.
- Names: name 3,254, name:en 1,682, int_name 0, Latin name 3,254, any English-capable name 3,254 (of 3,254).
- Script of `name`: `Latin` 3,254.
- `name:*` tags: `name:en` 1,682, `name:vi` 1,564, `name:zh` 443, `name:zh-Hant` 437, `name:zh-Hans` 428, `name:vi-Hani` 326, `name:lzh` 323, `name:de` 163.
- Geometry: the relations add up to 322,278 km² (67.1 % of the outline); 501 land plots lie in none of them, 128 in two or more; 2 relations hold no land plot; land plots per relation: 0 to 1,328, median 88.
- Rings not closed in the extract (a member way is missing): 0 in scope; 74 of the level's relations could not be assembled at all.
- Examples: A Dơi Commune (Xã A Dơi), Hà Đông Commune (Xã Hà Đông), Nam Triệu Ward (Phường Nam Triệu), Phường Long Trường, Tam Thanh Commune (Xã Tam Thanh), Văn Giang Commune (Xã Văn Giang), Xã Hàm Tân, Xã Phú Mỹ, Xã Vĩnh Bình, Ứng Hòa Commune (Xã Ứng Hòa).

### Level 8

- 74 relations in the extract, 18 in scope (country). Left out: 0 with another country's ISO 3166-2, 56 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 18.
- Names: name 18, name:en 0, int_name 0, Latin name 18, any English-capable name 18 (of 18).
- Script of `name`: `Latin` 18.
- `name:*` tags: `name:ja` 1, `name:ko` 1.
- Geometry: the relations add up to 106 km² (0.0 % of the outline); 390,461 land plots lie in none of them, 0 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 16, median 7.
- Rings not closed in the extract (a member way is missing): 0 in scope; 47 of the level's relations could not be assembled at all.
- Examples: Phường Tân Thành, Phường Vạn Hương, Thôn Hà Sơn, Thôn Làng Hào, Thôn Quý Nhân, Thôn Thượng Bản, Thôn Y Nhân, Thôn Đông Khê, Thôn Đồng Cảy, Xã Thanh Hà.

### Level 9

- 5,703 relations in the extract, 5,648 in scope (country). Left out: 0 with another country's ISO 3166-2, 52 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 5,648.
- Names: name 5,648, name:en 0, int_name 0, Latin name 5,648, any English-capable name 5,648 (of 5,648).
- Script of `name`: `Latin` 5,648.
- `name:*` tags: none.
- Geometry: the relations add up to 17,953 km² (3.7 % of the outline); 369,419 land plots lie in none of them, 37 in two or more; 2616 relations hold no land plot; land plots per relation: 0 to 311, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Buôn Dliê Ya A, Khu phố 18, Khu phố 29, Khu phố 47, Khu phố Hiệp An 1, Thôn Bình Thủy, Tổ dân phố An Hòa 2, Ấp 3, Ấp Lương Bình, Ấp Đức Hiệp.

### Level 10

- 3 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 34 relations, all with ISO 3166-2 VN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
- Districts: level 6: 3,254 relations, none with an ISO 3166-2 code, covering 99.9 % of the country's land plots (0.03 % overlap, 501 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:vi-Latn`, `name` (when written in Latin letters). Local name: `name:vi`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 33 relations)

- English name from: `name:en` 33; none for 0.
- Local name from: `name` 19, `name:vi` 14; none for 0. English equals local for 1.
- English prefixes: none. Suffixes: “Province” 24, “City” 3.
- Local prefixes: “Tỉnh” 23, “Thành” 8. Suffixes: “Ninh” 3.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 29 names, e.g. Cao Bằng Province: cao-b-ng-province → cao-bang-province, Cà Mau Province: c-mau-province → ca-mau-province, Cần Thơ: c-n-th → can-tho. Stage 2 should fold accents.

### Districts (level 6, 3,254 relations)

- English name from: `name:en` 1,682, `name` 1,572; none for 0.
- Local name from: `name` 1,690, `name:vi` 1,564; none for 0. English equals local for 1572.
- English prefixes: “Xã” 1192, “Phường” 374. Suffixes: “Commune” 1357.
- Local prefixes: “Xã” 2550, “Phường” 693. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 8 slugs are used more than once: 16/xa-tan-thanh (2), 19/cam-giang-commune (2), 23/van-lang-commune (2), 30/xa-loc-thanh (2), 31/xa-thanh-an (2), 4/xa-tan-thanh (2), 9/xa-ba-to (2), 9/xa-son-ha (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 3020 names, e.g. Phường Phú Thuận: ph-ng-ph-thu-n → phuong-phu-thuan, Phường Tân Hưng: ph-ng-t-n-h-ng → phuong-tan-hung, Phường Tân Mỹ: ph-ng-t-n-m → phuong-tan-my. Stage 2 should fold accents.

The slug of a new country's province carries the country as a suffix (`-vietnam`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 21,997. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 8,515 | 8,376 | 8,373 |
| neighbourhood | 4,434 | 4,398 | 4,398 |
| hamlet | 3,295 | 3,115 | 3,106 |
| quarter | 2,949 | 2,941 | 2,940 |
| town | 1,256 | 1,255 | 1,255 |
| suburb | 1,119 | 1,117 | 1,117 |
| city | 134 | 134 | 134 |
| locality | 116 | 115 | 114 |
| square | 55 | 54 | 53 |
| city_block | 41 | 41 | 41 |
| state | 25 | 25 | 25 |
| island | 16 | 15 | 15 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Ho Chi Minh City (population 14,002,598): neighbourhood 1,125 (1,120 English-capable), suburb 42 (42 English-capable), locality 1 (1 English-capable).
- Hanoi (population 8,685,607): neighbourhood 178 (178 English-capable), quarter 35 (34 English-capable), city_block 33 (33 English-capable), suburb 33 (33 English-capable), hamlet 4 (4 English-capable), village 4 (4 English-capable).
- Dong Nai (population 4,836,798): quarter 69 (69 English-capable), neighbourhood 12 (12 English-capable), suburb 7 (7 English-capable).
- Haiphong (population 4,664,124): quarter 205 (205 English-capable), suburb 12 (12 English-capable), hamlet 1 (1 English-capable), square 1 (0 English-capable).
- Cần Thơ (population 4,199,824): quarter 22 (22 English-capable), suburb 8 (8 English-capable), neighbourhood 6 (6 English-capable), city 1 (1 English-capable).

## Size

- Land plots: 390,593. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 16 s and 358 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Vietnam: from vietnam-latest.osm.pbf
{Code: "vn", Extract: "vietnam",
	Province: Rule{AdminLevel: 4, ISOPrefix: "VN", Count: 34, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:vi-Latn", "name"}, LocalLang: "vi",
	StripProvince: Strip{English: [2][]string{nil, []string{"Province"}}, Local: [2][]string{[]string{"Tỉnh"}, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{[]string{"Xã"}, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('vn', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
