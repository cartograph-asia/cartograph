# Country scan: Thailand

Written by `go run ./cmd/countryscan -country th` from `thailand-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 77 relations, all with ISO 3166-2 TH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 14 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 927 relations, none with an ISO 3166-2 code, covering 99.9 % of the country's land plots (0.01 % overlap, 622 plots in none). Per province: 3 to 50, median 10; 0 districts have their centre in no province.
- **Land plots:** 598,411.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 1 slugs used twice, 0 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r2067731 (admin_level 2); 630,101 km² in 1 polygons.
- Cells at resolution 8 with their centre inside the outline: 725,236; 598,411 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **598,411**, about 441,028 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with TH-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 5 | 5 | extract | 0 | 5 | 3 | 1 | – | – | – | – |
| 3 | 1 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 103 | 77 | country | 77 | 77 | 0 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 6 | 1,009 | 927 | country | 0 | 927 | 0 | 0 | 99.9 % | 0.1 % | 0.01 % | 0 |
| 7 | 569 | 549 | country | 1 | 500 | 0 | 0 | 3.5 % | 96.5 % | 0.00 % | 0 |
| 8 | 856 | 853 | country | 0 | 789 | 0 | 0 | 5.3 % | 94.7 % | 0.00 % | 0 |
| 9 | 4 | 4 | country | 0 | 2 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 10 | 217 | 214 | country | 0 | 190 | 0 | 0 | 0.1 % | 99.9 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Cambodia (ประเทศกัมพูชา), Laos (ประเทศลาว), Malaysia (ประเทศมาเลเซีย), Myanmar (ประเทศเมียนมา), Thailand (ประเทศไทย).

### Level 3

- 1 relation in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 103 relations in the extract, 77 in scope (country). Left out: 25 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `TH` 77; with this country's prefix: 77 of 77.
- Names: name 77, name:en 77, int_name 0, Latin name 0, any English-capable name 77 (of 77).
- Script of `name`: `Thai` 77.
- `name:*` tags: `name:en` 77, `name:es` 77, `name:fr` 77, `name:ja` 77, `name:pl` 77, `name:ru` 77, `name:th` 77, `name:zh` 77.
- Geometry: the relations add up to 630,101 km² (100.0 % of the outline); 14 land plots lie in none of them, 15 in two or more; 0 relations hold no land plot; land plots per relation: 481 to 26,014, median 6,568.
- Rings not closed in the extract (a member way is missing): 0 in scope; 26 of the level's relations could not be assembled at all.
- Examples: Amnat Charoen Province (จังหวัดอำนาจเจริญ), Chanthaburi Province (จังหวัดจันทบุรี), Khon Kaen Province (จังหวัดขอนแก่น), Nakhon Nayok Province (จังหวัดนครนายก), Nong Bua Lam Phu Province (จังหวัดหนองบัวลำภู), Phetchaburi Province (จังหวัดเพชรบุรี), Ranong Province (จังหวัดระนอง), Saraburi Province (จังหวัดสระบุรี), Surin Province (จังหวัดสุรินทร์), Yasothon Province (จังหวัดยโสธร).

### Level 6

- 1,009 relations in the extract, 927 in scope (country). Left out: 0 with another country's ISO 3166-2, 82 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 927.
- Names: name 925, name:en 927, int_name 0, Latin name 0, any English-capable name 927 (of 927).
- Script of `name`: `Thai` 925.
- `name:*` tags: `name:en` 927, `name:th` 890, `name:th-Latn` 80, `name:zh` 20, `name:ja` 18, `name:zh-Hant` 18, `name:ru` 12, `name:fr` 7.
- Geometry: the relations add up to 517,767 km² (82.2 % of the outline); 622 land plots lie in none of them, 88 in two or more; 0 relations hold no land plot; land plots per relation: 2 to 5,551, median 522.
- Rings not closed in the extract (a member way is missing): 0 in scope; 76 of the level's relations could not be assembled at all.
- Examples: Akat Amnuai District (อำเภออากาศอำนวย), Bung Khla District (อำเภอบุ่งคล้า), In Buri District (อำเภออินทร์บุรี), Krasang District (อำเภอกระสัง), Mueang Mae Hong Son District (อำเภอเมืองแม่ฮ่องสอน), Non Sa-at District (อำเภอโนนสะอาด), Pho Si Suwan District (อำเภอโพธิ์ศรีสุวรรณ), San Sai District (อำเภอสันทราย), Tha Sae District (อำเภอท่าแซะ), Yi-ngo District (อำเภอยี่งอ).

### Level 7

- 569 relations in the extract, 549 in scope (country). Left out: 0 with another country's ISO 3166-2, 20 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `TH` 1; with this country's prefix: 1 of 549.
- Names: name 549, name:en 500, int_name 0, Latin name 0, any English-capable name 500 (of 549).
- Script of `name`: `Thai` 549.
- `name:*` tags: `name:en` 500, `name:th` 149, `name:ja` 4, `name:uk` 4, `name:zh` 3, `name:zh-Hant` 3, `name:de` 2, `name:th-Latn` 2.
- Geometry: the relations add up to 18,224 km² (2.9 % of the outline); 577,570 land plots lie in none of them, 9 in two or more; 3 relations hold no land plot; land plots per relation: 0 to 621, median 19.
- Rings not closed in the extract (a member way is missing): 0 in scope; 20 of the level's relations could not be assembled at all.
- Examples: Aranyik Subdistrict Municipality (เทศบาลตำบลอรัญญิก), Bang Kadi Town Municipality (เทศบาลเมืองบางกะดี), Bo Ta Lo Subdistrict Municipality (เทศบาลตำบลบ่อตาโล่), Khlong Khoi Subdistrict Administrative Organization (องค์การบริหารส่วนตำบลคลองข่อย), Lam Sai Subdistrict Municipality (เทศบาลตำบลลำไทร), Nong Khanak Subdistrict Administrative Organization (องค์การบริหารส่วนตำบลหนองขนาก), Phukhao Thong Subdistrict Administrative Organization (องค์การบริหารส่วนตำบลภูเขาทอง), Sawang Dane Din Township Municipality (เทศบาลตำบลสว่างแดนดิน), Wang Chula Subdistrict Administrative Organization (องค์การบริหารส่วนตำบลวังจุฬา), เทศบาลเมืองแม่โจ้.

### Level 8

- 856 relations in the extract, 853 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 853.
- Names: name 852, name:en 789, int_name 0, Latin name 0, any English-capable name 789 (of 853).
- Script of `name`: `Thai` 852.
- `name:*` tags: `name:en` 789, `name:th` 131, `name:th-Latn` 55, `name:de` 17, `name:ru` 17, `name:etymology:wikidata` 1, `name:ja` 1, `name:vi` 1.
- Geometry: the relations add up to 27,657 km² (4.4 % of the outline); 566,686 land plots lie in none of them, 15 in two or more; 16 relations hold no land plot; land plots per relation: 0 to 621, median 18.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Amarit Subdistrict (ตำบลอมฤต), Bang Khru Subdistrict (ตำบลบางครุ), Bu Ruesi Subdistrict (ตำบลบุฤๅษี), Khao Mao Subdistrict (ตำบลข้าวเม่า), Lak Chai Subdistrict (ตำบลหลักชัย), Nong Luang Subdistrict (ตำบลหนองหลวง), Rangsit Subdistrict (ตำบลรังสิต), Suan Chitlada Subdistrict (แขวงสวนจิตรลดา), Wang Mai Subdistrict (แขวงวังใหม่), บ้านอ่าวนาง.

### Level 9

- 4 relations in the extract, 4 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 4.
- Names: name 4, name:en 2, int_name 0, Latin name 0, any English-capable name 2 (of 4).
- Script of `name`: `Thai` 4.
- `name:*` tags: `name:en` 2.
- Geometry: the relations add up to 41 km² (0.0 % of the outline); 598,362 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 6 to 15, median 15.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Kawila Subdistrict (แขวงกาวิละ), Mengrai Subdistrict (แขวงเม็งราย), แขวงนครพิงค์, แขวงศรีวิชัย.

### Level 10

- 217 relations in the extract, 214 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 214.
- Names: name 214, name:en 190, int_name 0, Latin name 0, any English-capable name 190 (of 214).
- Script of `name`: `Thai` 214.
- `name:*` tags: `name:en` 190.
- Geometry: the relations add up to 289 km² (0.0 % of the outline); 598,080 land plots lie in none of them, 1 in two or more; 72 relations hold no land plot; land plots per relation: 0 to 15, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Moo 1 (หมู่ 1), Moo 8 (หมู่ 8), Mu 1 (หมู่ที่ 1), Mu 2 (หมู่ที่ 2), Mu 3 (หมู่ที่ 3), Mu 4 (หมู่ที่ 4), Mu 5 (หมู่ที่ 5), Mu 7 (หมู่ที่ 7), Mu 9 (หมู่ที่ 9), หมู่ 9.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 77 relations, all with ISO 3166-2 TH-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 14 plots in none).
- Districts: level 6: 927 relations, none with an ISO 3166-2 code, covering 99.9 % of the country's land plots (0.01 % overlap, 622 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:th-Latn`, `name` (when written in Latin letters). Local name: `name:th`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 77 relations)

- English name from: `name:en` 77; none for 0.
- Local name from: `name:th` 77; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “Province” 76.
- Local prefixes: “จังหวั” 76. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 927 relations)

- English name from: `name:en` 927; none for 0.
- Local name from: `name:th` 890, `name` 36; none for 1. English equals local for 0.
- English prefixes: none. Suffixes: “District” 924.
- Local prefixes: “อำเภอ” 876. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 1 slugs are used more than once: 41/bang-sai-district (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-thailand`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 32,352. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 21,836 | 18,055 | 17,010 |
| hamlet | 5,414 | 4,118 | 3,822 |
| municipality | 1,680 | 1,680 | 1,680 |
| neighbourhood | 1,405 | 1,405 | 893 |
| town | 1,039 | 1,039 | 1,035 |
| quarter | 384 | 384 | 374 |
| county | 229 | 229 | 215 |
| locality | 77 | 71 | 54 |
| province | 76 | 76 | 76 |
| suburb | 55 | 55 | 55 |
| city | 37 | 37 | 37 |
| district | 35 | 35 | 35 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Bangkok (population 5,527,948): neighbourhood 328 (153 English-capable), quarter 89 (89 English-capable), suburb 22 (22 English-capable), city_block 20 (19 English-capable), village 12 (12 English-capable), locality 6 (4 English-capable).
- Nonthaburi (population 251,026): neighbourhood 106 (23 English-capable), village 80 (80 English-capable), locality 20 (20 English-capable), quarter 10 (10 English-capable), municipality 7 (7 English-capable), county 5 (5 English-capable).
- Pak Kret (population 190,272): village 94 (94 English-capable), municipality 8 (8 English-capable), neighbourhood 5 (5 English-capable), county 4 (4 English-capable), city 1 (1 English-capable), locality 2 (2 English-capable).
- Hat Yai (population 156,802): village 10 (10 English-capable), municipality 3 (3 English-capable), hamlet 2 (2 English-capable), town 2 (2 English-capable).
- Surat Thani (population 132,040): village 4 (4 English-capable), municipality 1 (1 English-capable), town 1 (1 English-capable).

## Size

- Land plots: 598,411. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 25 s and 550 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Thailand: from thailand-latest.osm.pbf
{Code: "th", Extract: "thailand",
	Province: Rule{AdminLevel: 4, ISOPrefix: "TH", Count: 77, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:th-Latn", "name"}, LocalLang: "th",
	StripProvince: Strip{English: [2][]string{nil, []string{"Province"}}, Local: [2][]string{[]string{"จังหวั"}, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, []string{"District"}}, Local: [2][]string{[]string{"อำเภอ"}, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('th', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
