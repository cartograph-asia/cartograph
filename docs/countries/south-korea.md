# Country scan: South Korea

Written by `go run ./cmd/countryscan -country kr` from `south-korea-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 16 relations, 15 with ISO 3166-2 KR-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 2 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 229 relations, none with an ISO 3166-2 code, covering 99.5 % of the country's land plots (0.01 % overlap, 911 plots in none). Per province: 0 to 31, median 15; 0 districts have their centre in no province.
- **Land plots:** 191,458.
- **Names (districts):** 1 without an English-capable name, 1 with an empty slug, 0 slugs used twice, 0 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r307756 (admin_level 2); 184,986 km² in 4 polygons.
- Cells at resolution 8 with their centre inside the outline: 346,893; 191,458 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **191,458**, about 141,104 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with KR-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 2 | 2 | extract | 0 | 2 | 1 | 0 | – | – | – | – |
| 4 | 19 | 16 | country | 15 | 16 | 0 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 6 | 243 | 229 | country | 0 | 228 | 0 | 0 | 99.5 % | 0.5 % | 0.01 % | 0 |
| 7 | 39 | 39 | country | 0 | 29 | 0 | 0 | 6.0 % | 94.0 % | 0.00 % | 0 |
| 8 | 2,084 | 2,080 | country | 0 | 1,818 | 0 | 0 | 50.5 % | 49.5 % | 0.10 % | 0 |
| 9 | 8 | 8 | country | 0 | 0 | 0 | 0 | 0.2 % | 99.8 % | 0.00 % | 0 |
| 10 | 1,289 | 1,284 | country | 0 | 995 | 0 | 0 | 4.7 % | 95.3 % | 0.01 % | 0 |

### Level 2

Countries in the extract: North Korea (조선민주주의인민공화국), South Korea (대한민국).

### Level 4

- 19 relations in the extract, 16 in scope (country). Left out: 3 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `KR` 15; with this country's prefix: 15 of 16.
- Names: name 16, name:en 16, int_name 0, Latin name 0, any English-capable name 16 (of 16).
- Script of `name`: `Hangul` 16.
- `name:*` tags: `name:en` 16, `name:ja` 16, `name:ko` 16, `name:zh-Hans` 16, `name:zh-Hant` 16, `name:ko-Hani` 15, `name:zh` 15, `name:cs` 14.
- Geometry: the relations add up to 184,986 km² (100.0 % of the outline); 2 land plots lie in none of them, 2 in two or more; 0 relations hold no land plot; land plots per relation: 899 to 34,660, median 14,268.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Busan (부산광역시), Daegu (대구광역시), Gangwon State (강원특별자치도), Incheon (인천광역시), Jeju (제주특별자치도), Jeonnam-Gwangju Special Metropolitan City (전남광주통합특별시), North Gyeongsang (경상북도), Sejong (세종특별자치시), South Chungcheong (충청남도), Ulsan (울산광역시).

### Level 6

- 243 relations in the extract, 229 in scope (country). Left out: 0 with another country's ISO 3166-2, 14 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 229.
- Names: name 229, name:en 228, int_name 0, Latin name 0, any English-capable name 228 (of 229).
- Script of `name`: `Hangul` 229.
- `name:*` tags: `name:ko` 229, `name:en` 228, `name:ko-Hani` 228, `name:zh-Hant` 228, `name:ja` 226, `name:zh` 224, `name:zh-Hans` 146, `name:de` 34.
- Geometry: the relations add up to 180,578 km² (97.6 % of the outline); 911 land plots lie in none of them, 16 in two or more; 0 relations hold no land plot; land plots per relation: 6 to 3,479, median 846.
- Rings not closed in the extract (a member way is missing): 0 in scope; 14 of the level's relations could not be assembled at all.
- Examples: Andong-si (안동시), Cheorwon-gun (철원군), Ganghwa-gun (강화군), Gumi-si (구미시), Hapcheon-gun (합천군), Jongno-gu (종로구), Ongjin-gun (옹진군), Seongnam-si (성남시), Yanggu-gun (양구군), 검단구.

### Level 7

- 39 relations in the extract, 39 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 39.
- Names: name 39, name:en 29, int_name 0, Latin name 0, any English-capable name 29 (of 39).
- Script of `name`: `Hangul` 39.
- `name:*` tags: `name:ko` 39, `name:ja` 34, `name:ko-Hani` 34, `name:zh` 34, `name:en` 29, `name:zh-Hant` 27, `name:zh-Hans` 24, `name:ko-Latn` 5.
- Geometry: the relations add up to 6,942 km² (3.8 % of the outline); 180,039 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 23 to 1,372, median 163.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Buk-gu (북구), Danwon-gu (단원구), Heungdeok-gu (흥덕구), Jinhae-gu (진해구), Masanhoewon-gu (마산회원구), Seobuk-gu (서북구), Sujeong-gu (수정구), 권선구, 만안구, 팔달구.

### Level 8

- 2,084 relations in the extract, 2,080 in scope (country). Left out: 0 with another country's ISO 3166-2, 4 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2,080.
- Names: name 2,079, name:en 1,818, int_name 0, Latin name 0, any English-capable name 1,834 (of 2,080).
- Script of `name`: `Hangul` 2,079.
- `name:*` tags: `name:zh` 2,012, `name:zh-Hant` 2,011, `name:ko-Hani` 2,008, `name:ko` 1,850, `name:en` 1,818, `name:zh-Hans` 1,059, `name:ja` 669, `name:de` 328.
- Geometry: the relations add up to 58,768 km² (31.8 % of the outline); 94,748 land plots lie in none of them, 197 in two or more; 83 relations hold no land plot; land plots per relation: 0 to 848, median 6.
- Rings not closed in the extract (a member way is missing): 0 in scope; 4 of the level's relations could not be assembled at all.
- Examples: -Jinseong-myeon (진성면), Cheongsong-eup (청송읍), Gaepo 3(sam)-dong (개포3동), Gyeyang 1(il)-dong (계양1동), Jangwi 1(il)-dong (장위1동), Myeongil 2(i)-dong (명일2동), Sanggyo-dong (상교동), Ssangmun 4(sa)-dong (쌍문4동), 광남2동, 흥업면.

### Level 9

- 8 relations in the extract, 8 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 8.
- Names: name 8, name:en 0, int_name 0, Latin name 0, any English-capable name 0 (of 8).
- Script of `name`: `Hangul` 8.
- `name:*` tags: `name:ko` 6, `name:el` 1.
- Geometry: the relations add up to 237 km² (0.1 % of the outline); 190,996 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 5 to 148, median 76.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: 기배동, 남양읍, 매송면, 백석동, 봉담읍, 비봉면, 새솔동, 정남면.

### Level 10

- 1,289 relations in the extract, 1,284 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1,284.
- Names: name 1,283, name:en 995, int_name 0, Latin name 0, any English-capable name 997 (of 1,284).
- Script of `name`: `Hangul` 1,283.
- `name:*` tags: `name:ko-Hani` 1,239, `name:en` 995, `name:ko` 988, `name:zh` 975, `name:zh-Hant` 963, `name:zh-Hans` 961, `name:ko-Latn` 859, `name:ja` 744.
- Geometry: the relations add up to 8,638 km² (4.7 % of the outline); 182,500 land plots lie in none of them, 27 in two or more; 273 relations hold no land plot; land plots per relation: 0 to 99, median 5.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: 10통, 24통, Bakji-ri (박지리), Deokso 12-ri (덕소12리), Goseong-ri (고성리), Jangnyeon-ri (장년리), Nampung-ri (남풍리), Seolmae-ri (설매리), Tando-ri (탄도리), 초원지리.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 16 relations, 15 with ISO 3166-2 KR-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 2 plots in none).
- Districts: level 6: 229 relations, none with an ISO 3166-2 code, covering 99.5 % of the country's land plots (0.01 % overlap, 911 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:ko-Latn`, `name` (when written in Latin letters). Local name: `name:ko`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 16 relations)

- English name from: `name:en` 16; none for 0.
- Local name from: `name:ko` 16; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: “도” 8, “시” 8, “광역시” 5.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 229 relations)

- English name from: `name:en` 228; none for 1.
- Local name from: `name:ko` 229; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: “군” 82, “시” 77, “구” 70.
- Slugs: 1 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-south-korea`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 22,371. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 15,118 | 15,118 | 15,105 |
| quarter | 3,592 | 3,590 | 3,554 |
| town | 1,412 | 1,412 | 1,408 |
| hamlet | 719 | 695 | 350 |
| locality | 620 | 617 | 244 |
| neighbourhood | 273 | 272 | 234 |
| isolated_dwelling | 105 | 61 | 4 |
| borough | 100 | 100 | 100 |
| city | 85 | 85 | 85 |
| county | 85 | 85 | 85 |
| square | 55 | 46 | 8 |
| island | 51 | 51 | 42 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Seoul (population 9,635,445): quarter 322 (321 English-capable), locality 18 (9 English-capable), borough 11 (11 English-capable), neighbourhood 11 (8 English-capable), isolated_dwelling 5 (1 English-capable), square 5 (4 English-capable).
- Busan (population 3,455,611): quarter 52 (51 English-capable), borough 7 (7 English-capable), locality 5 (0 English-capable), island 1 (1 English-capable), square 1 (0 English-capable).
- Incheon (population 2,953,883): quarter 87 (87 English-capable), borough 6 (6 English-capable), city 1 (1 English-capable).
- Daegu (population 2,468,222): quarter 143 (143 English-capable), neighbourhood 9 (9 English-capable), borough 6 (6 English-capable), locality 2 (0 English-capable), island 1 (0 English-capable).
- Gwangju (population 1,460,745): quarter 113 (113 English-capable), locality 7 (0 English-capable), borough 5 (5 English-capable), hamlet 2 (0 English-capable), neighbourhood 2 (1 English-capable).

## Size

- Land plots: 191,458. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 7 s and 175 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// South Korea: from south-korea-latest.osm.pbf
{Code: "kr", Extract: "south-korea",
	Province: Rule{AdminLevel: 4, ISOPrefix: "KR", Count: 16, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:ko-Latn", "name"}, LocalLang: "ko",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, []string{"도", "시"}}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('kr', '{quarter,neighbourhood}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
