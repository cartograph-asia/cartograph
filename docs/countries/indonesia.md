# Country scan: Indonesia

Written by `go run ./cmd/countryscan -country id` from `indonesia-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 38 relations, all with ISO 3166-2 ID-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 5.
- **Districts:** level 5: 502 relations, 1 with ISO 3166-2 ID-xx, covering 99.8 % of the country's land plots (0.00 % overlap, 3,509 plots in none). Per province: 4 to 38, median 11; 0 districts have their centre in no province.
- **Land plots:** 2,316,624.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 13 slugs used twice, 0 names whose slug loses letters without accent folding.
- Provinces whose rings do not close in the extract: 2; without a land plot: 0.

## Outline and land

- Outline: the union of 36 admin_level 4 relations with ISO 3166-2 ID-xx (the country relation is not whole in the extract); 2 more of them are not whole either and are missing from it, so the outline and the land plots are too small; 4,954,298 km² in 36 polygons.
- Cells at resolution 8 with their centre inside the outline: 6,332,232; 2,316,624 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **2,316,624**, about 1,707,351 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with ID-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 5 | 5 | extract | 0 | 5 | 1 | 5 | – | – | – | – |
| 3 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 60 | 38 | country | 38 | 38 | 0 | 38 | 100.0 % | 0.0 % | 0.00 % | 2 |
| 5 | 597 | 502 | country | 1 | 115 | 0 | 502 | 99.8 % | 0.2 % | 0.00 % | 0 |
| 6 | 2,194 | 2,153 | country | 0 | 97 | 0 | 2,152 | 29.0 % | 71.0 % | 0.00 % | 0 |
| 7 | 6,538 | 6,460 | country | 0 | 71 | 0 | 6,458 | 3.0 % | 97.0 % | 0.07 % | 0 |
| 8 | 1,179 | 1,163 | country | 0 | 10 | 0 | 1,161 | 0.1 % | 99.9 % | 0.00 % | 0 |
| 9 | 6,742 | 6,719 | country | 0 | 0 | 0 | 6,664 | 0.1 % | 99.9 % | 0.00 % | 0 |
| 10 | 6,621 | 4,796 | country | 0 | 0 | 0 | 4,756 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: East Timor (Timor Leste), Indonesia, Malaysia, Papua New Guinea (Papua Nugini), Singapore (Singapura).

### Level 3

- 3 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 60 relations in the extract, 38 in scope (country). Left out: 21 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `ID` 38; with this country's prefix: 38 of 38.
- Names: name 38, name:en 38, int_name 0, Latin name 38, any English-capable name 38 (of 38).
- Script of `name`: `Latin` 38.
- `name:*` tags: `name:ar` 38, `name:en` 38, `name:id` 38, `name:ko` 38, `name:ru` 38, `name:su` 38, `name:su-Latn` 38, `name:uk` 38.
- Geometry: the relations add up to 4,954,298 km² (100.0 % of the outline); 0 land plots lie in none of them, 5 in two or more; 0 relations hold no land plot; land plots per relation: 896 to 211,809, median 56,189.
- Rings not closed in the extract (a member way is missing): 2 in scope; 10 of the level's relations could not be assembled at all. In scope: r3797244 Riau Islands, r5449460 North Kalimantan.
- Examples: Aceh, Bengkulu, Central Sulawesi (Sulawesi Tengah), Gorontalo, Maluku, North Sumatra (Sumatera Utara), South Kalimantan (Kalimantan Selatan), Southeast Sulawesi (Sulawesi Tenggara), West Java (Jawa Barat), West Sumatra (Sumatera Barat).

### Level 5

- 597 relations in the extract, 502 in scope (country). Left out: 0 with another country's ISO 3166-2, 95 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `ID` 1; with this country's prefix: 1 of 502.
- Names: name 502, name:en 115, int_name 0, Latin name 502, any English-capable name 502 (of 502).
- Script of `name`: `Latin` 502.
- `name:*` tags: `name:ar` 397, `name:ko` 173, `name:en` 115, `name:id` 73, `name:ja` 54, `name:jv` 49, `name:uk` 40, `name:zh` 36.
- Geometry: the relations add up to 1,968,194 km² (39.7 % of the outline); 3,509 land plots lie in none of them, 47 in two or more; 0 relations hold no land plot; land plots per relation: 11 to 55,525, median 2,555.
- Rings not closed in the extract (a member way is missing): 0 in scope; 14 of the level's relations could not be assembled at all.
- Examples: Aceh Barat, Bengkulu, City of Tebing Tinggi (Kota Tebing Tinggi), Kabupaten Karo, Kubu Raya, Manokwari, Palu, Salatiga, Sumenep, Yapen Islands Regency (Kepulauan Yapen).

### Level 6

- 2,194 relations in the extract, 2,153 in scope (country). Left out: 0 with another country's ISO 3166-2, 41 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2,153.
- Names: name 2,152, name:en 97, int_name 0, Latin name 2,152, any English-capable name 2,152 (of 2,153).
- Script of `name`: `Latin` 2,152.
- `name:*` tags: `name:ms-Arab` 181, `name:ms` 169, `name:id` 136, `name:ar` 106, `name:en` 97, `name:ko` 78, `name:zh-Hant` 76, `name:zh` 68.
- Geometry: the relations add up to 566,577 km² (11.4 % of the outline); 1,644,046 land plots lie in none of them, 104 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 8,797, median 126.
- Rings not closed in the extract (a member way is missing): 0 in scope; 16 of the level's relations could not be assembled at all.
- Examples: Abab, Bintang Bayu, Dusun Nefo, Kecamatan Haharu, Limbangan, Ngancar, Pontianak Utara, Serang, Tambakrejo, Yalengga.

### Level 7

- 6,538 relations in the extract, 6,460 in scope (country). Left out: 0 with another country's ISO 3166-2, 68 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 6,460.
- Names: name 6,459, name:en 71, int_name 0, Latin name 6,458, any English-capable name 6,458 (of 6,460).
- Script of `name`: `Latin` 6,459.
- `name:*` tags: `name:ar` 739, `name:zh-Hant` 723, `name:ja` 718, `name:ko` 717, `name:or` 717, `name:th` 717, `name:arz` 716, `name:ban` 716.
- Geometry: the relations add up to 51,619 km² (1.0 % of the outline); 2,247,923 land plots lie in none of them, 1,688 in two or more; 379 relations hold no land plot; land plots per relation: 0 to 885, median 5.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Aan, Brayublandong, Desa Kadumbul, Jatiluwih, Kendangsari, Mekarasih, Pattapang, Selong, Tajun, Yudha Karya Jitu.

### Level 8

- 1,179 relations in the extract, 1,163 in scope (country). Left out: 0 with another country's ISO 3166-2, 7 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1,163.
- Names: name 1,161, name:en 10, int_name 0, Latin name 1,161, any English-capable name 1,161 (of 1,163).
- Script of `name`: `Latin` 1,161.
- `name:*` tags: `name:id` 17, `name:en` 10, `name:jv` 6, `name:su-Latn` 6, `name:su` 4, `name:ace` 1, `name:zh-Hans` 1.
- Geometry: the relations add up to 2,304 km² (0.0 % of the outline); 2,313,640 land plots lie in none of them, 66 in two or more; 531 relations hold no land plot; land plots per relation: 0 to 584, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 7 of the level's relations could not be assembled at all.
- Examples: Adikarto, Desa Mandalawangi Kecamatan Salopa, Dusun Cidugalen Desa Cidugalen, Dusun Muara 2, Dusun Tenga, Kampungbaru, Lindongan 1, Pampalaraeng, Ngringit, RW 08 Desa Sukamenak, Yapak Lor.

### Level 9

- 6,742 relations in the extract, 6,719 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 6,719.
- Names: name 6,664, name:en 0, int_name 0, Latin name 6,664, any English-capable name 6,664 (of 6,719).
- Script of `name`: `Latin` 6,664.
- `name:*` tags: none.
- Geometry: the relations add up to 1,899 km² (0.0 % of the outline); 2,314,033 land plots lie in none of them, 42 in two or more; 4811 relations hold no land plot; land plots per relation: 0 to 24, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Area Pengembangan Waduk, RW 01, RW 02, RW 03, RW 05, RW 06, RW 07 Desa Kertamukti, RW 09, RW 12 Kelurahan Donan, Wadul Pluit.

### Level 10

- 6,621 relations in the extract, 4,796 in scope (country). Left out: 0 with another country's ISO 3166-2, 8 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 4,796.
- Names: name 4,757, name:en 0, int_name 0, Latin name 4,756, any English-capable name 4,756 (of 4,796).
- Script of `name`: `Latin` 4,756.
- `name:*` tags: `name:id` 3, `name:ja` 3.
- Geometry: the relations add up to 374 km² (0.0 % of the outline); 2,316,146 land plots lie in none of them, 4 in two or more; 4440 relations hold no land plot; land plots per relation: 0 to 16, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 7 of the level's relations could not be assembled at all.
- Examples: 001, RT 01, RT 02, RT 04, RT 05, RT 07, RT 09, RT 11, RT 18, Wilayah KKP.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 38 relations, all with ISO 3166-2 ID-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
- Districts: level 5: 502 relations, 1 with ISO 3166-2 ID-xx, covering 99.8 % of the country's land plots (0.00 % overlap, 3,509 plots in none).

Levels that qualify: 4, 5. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:id-Latn`, `name` (when written in Latin letters). Local name: `name:id`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 36 relations)

- English name from: `name:en` 36; none for 0.
- Local name from: `name:id` 36; none for 0. English equals local for 10.
- English prefixes: “West” 6, “Central” 4, “South” 4, “North” 3, “East” 3. Suffixes: “Sulawesi” 5, “Papua” 5, “Kalimantan” 4, “Sumatra” 3, “Java” 3.
- Local prefixes: “Sulawesi” 5, “Papua” 5, “Kalimantan” 4, “Sumatera” 3, “Jawa” 3. Suffixes: “Barat” 6, “Selatan” 4, “Tengah” 4, “Timur” 3, “Utara” 3.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 5, 502 relations)

- English name from: `name` 387, `name:en` 115; none for 0.
- Local name from: `name` 429, `name:id` 73; none for 0. English equals local for 448.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 13 slugs are used more than once: 1/bima (2), 19/solok (2), 22/kupang (2), 25/blitar (2), 27/jayapura (2), 3/pekalongan (2), 3/tegal (2), 35/sorong (2), 4/bogor (2), 4/cirebon (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-indonesia`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 113,143. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 74,706 | 74,341 | 74,340 |
| hamlet | 27,294 | 26,737 | 26,659 |
| neighbourhood | 3,010 | 2,890 | 2,890 |
| town | 1,877 | 1,866 | 1,866 |
| locality | 1,654 | 1,651 | 1,651 |
| suburb | 1,596 | 1,592 | 1,590 |
| island | 1,202 | 1,198 | 1,198 |
| isolated_dwelling | 425 | 414 | 414 |
| city | 269 | 269 | 269 |
| county | 211 | 211 | 211 |
| city_block | 190 | 190 | 187 |
| quarter | 157 | 156 | 156 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Jakarta (population 10,467,629): village 132 (132 English-capable), locality 36 (36 English-capable), municipality 19 (19 English-capable), neighbourhood 10 (10 English-capable).
- Surabaya (population 2,765,487): village 102 (102 English-capable), municipality 19 (19 English-capable), hamlet 1 (1 English-capable).
- Medan (population 2,551,006): suburb 121 (121 English-capable), village 9 (9 English-capable), neighbourhood 6 (6 English-capable), farm 1 (1 English-capable), locality 1 (0 English-capable).
- Bandung (population 2,390,120): suburb 128 (128 English-capable), neighbourhood 34 (33 English-capable), district 24 (24 English-capable), village 22 (22 English-capable), quarter 6 (6 English-capable), town 5 (5 English-capable).
- Bekasi (population 2,334,871): neighbourhood 78 (78 English-capable), village 37 (37 English-capable), suburb 33 (33 English-capable), quarter 9 (9 English-capable), locality 2 (2 English-capable).

## Size

- Land plots: 2,316,624. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 96 s and 2,129 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Indonesia: from indonesia-latest.osm.pbf
{Code: "id", Extract: "indonesia",
	Province: Rule{AdminLevel: 4, ISOPrefix: "ID", Count: 38, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 5},
	EnglishName: []string{"name:en", "int_name", "name:id-Latn", "name"}, LocalLang: "id",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('id', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
