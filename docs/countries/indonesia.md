# Country scan: Indonesia

Written by `go run ./cmd/countryscan -country id` from `indonesia-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Boundaries completed with the ways and nodes of the neighbours' cuts `malaysia-singapore-brunei-latest.osm.pbf.cut.osm.pbf` (relations and places from the own file only). Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 38 relations, all with ISO 3166-2 ID-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 5 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 5.
- **Districts:** level 5: 514 relations, 1 with ISO 3166-2 ID-xx, covering 99.9 % of the country's land plots (0.00 % overlap, 3,587 plots in none). Per province: 4 to 38, median 11; 0 districts have their centre in no province.
- **Land plots:** 2,423,959.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 13 slugs used twice, 0 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r304751 (admin_level 2); 5,299,981 km² in 1 polygons.
- Cells at resolution 8 with their centre inside the outline: 6,767,023; 2,423,959 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **2,423,959**, about 1,786,457 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with ID-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 5 | 5 | extract | 0 | 5 | 1 | 5 | – | – | – | – |
| 3 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 60 | 38 | country | 38 | 38 | 0 | 38 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 597 | 514 | country | 1 | 115 | 0 | 514 | 99.9 % | 0.1 % | 0.00 % | 0 |
| 6 | 2,194 | 2,173 | country | 0 | 97 | 0 | 2,172 | 27.8 % | 72.2 % | 0.00 % | 0 |
| 7 | 6,538 | 6,525 | country | 0 | 71 | 0 | 6,523 | 2.9 % | 97.1 % | 0.07 % | 0 |
| 8 | 1,179 | 1,163 | country | 0 | 10 | 0 | 1,161 | 0.1 % | 99.9 % | 0.00 % | 0 |
| 9 | 6,742 | 6,719 | country | 0 | 0 | 0 | 6,664 | 0.1 % | 99.9 % | 0.00 % | 0 |
| 10 | 6,621 | 4,797 | country | 0 | 0 | 0 | 4,757 | 0.0 % | 100.0 % | 0.00 % | 0 |

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
- Geometry: the relations add up to 5,299,981 km² (100.0 % of the outline); 5 land plots lie in none of them, 5 in two or more; 0 relations hold no land plot; land plots per relation: 896 to 211,809, median 56,189.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Aceh, Bengkulu, Central Sulawesi (Sulawesi Tengah), Gorontalo, Maluku, North Sumatra (Sumatera Utara), South Kalimantan (Kalimantan Selatan), Southeast Sulawesi (Sulawesi Tenggara), West Java (Jawa Barat), West Sumatra (Sumatera Barat).

### Level 5

- 597 relations in the extract, 514 in scope (country). Left out: 0 with another country's ISO 3166-2, 83 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `ID` 1; with this country's prefix: 1 of 514.
- Names: name 514, name:en 115, int_name 0, Latin name 514, any English-capable name 514 (of 514).
- Script of `name`: `Latin` 514.
- `name:*` tags: `name:ar` 409, `name:ko` 174, `name:en` 115, `name:id` 73, `name:ja` 55, `name:jv` 49, `name:uk` 41, `name:zh` 39.
- Geometry: the relations add up to 2,047,334 km² (38.6 % of the outline); 3,587 land plots lie in none of them, 48 in two or more; 0 relations hold no land plot; land plots per relation: 11 to 55,525, median 2,554.
- Rings not closed in the extract (a member way is missing): 0 in scope; 5 of the level's relations could not be assembled at all.
- Examples: Aceh Barat, Bengkulu Selatan, City of Tebing Tinggi (Kota Tebing Tinggi), Kabupaten Langkat, Kudus, Manokwari, Palopo, Samarinda, Supiori, Yapen Islands Regency (Kepulauan Yapen).

### Level 6

- 2,194 relations in the extract, 2,173 in scope (country). Left out: 0 with another country's ISO 3166-2, 21 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2,173.
- Names: name 2,172, name:en 97, int_name 0, Latin name 2,172, any English-capable name 2,172 (of 2,173).
- Script of `name`: `Latin` 2,172.
- `name:*` tags: `name:ms-Arab` 182, `name:ms` 170, `name:id` 137, `name:ar` 106, `name:en` 97, `name:ja` 79, `name:ko` 78, `name:zh-Hant` 76.
- Geometry: the relations add up to 568,068 km² (10.7 % of the outline); 1,749,554 land plots lie in none of them, 105 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 8,797, median 125.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Abab, Binjai Selatan, Dumai Kota, Kecamatan Bongas, Libarek, Ngambon, Pontianak Timur, Sepauk, Tamansari, Yalengga.

### Level 7

- 6,538 relations in the extract, 6,525 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 6,525.
- Names: name 6,524, name:en 71, int_name 0, Latin name 6,523, any English-capable name 6,523 (of 6,525).
- Script of `name`: `Latin` 6,524.
- `name:*` tags: `name:ar` 739, `name:zh-Hant` 723, `name:ja` 718, `name:ko` 717, `name:or` 717, `name:th` 717, `name:arz` 716, `name:ban` 716.
- Geometry: the relations add up to 52,628 km² (1.0 % of the outline); 2,354,060 land plots lie in none of them, 1,689 in two or more; 380 relations hold no land plot; land plots per relation: 0 to 885, median 5.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Aan, Brangkal, Desa Kakaha, Jatinegara Kaum, Keponggok, Mekarjaya, Pedurenan, Semangga Jaya, Talaga, Yudha Karya Jitu.

### Level 8

- 1,179 relations in the extract, 1,163 in scope (country). Left out: 0 with another country's ISO 3166-2, 7 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1,163.
- Names: name 1,161, name:en 10, int_name 0, Latin name 1,161, any English-capable name 1,161 (of 1,163).
- Script of `name`: `Latin` 1,161.
- `name:*` tags: `name:id` 17, `name:en` 10, `name:jv` 6, `name:su-Latn` 6, `name:su` 4, `name:ace` 1, `name:zh-Hans` 1.
- Geometry: the relations add up to 2,304 km² (0.0 % of the outline); 2,420,975 land plots lie in none of them, 66 in two or more; 531 relations hold no land plot; land plots per relation: 0 to 584, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 7 of the level's relations could not be assembled at all.
- Examples: Adikarto, Desa Mandalawangi Kecamatan Salopa, Dusun Cidugalen Desa Cidugalen, Dusun Muara 2, Dusun Tenga, Kampungbaru, Lindongan 1, Pampalaraeng, Ngringit, RW 08 Desa Sukamenak, Yapak Lor.

### Level 9

- 6,742 relations in the extract, 6,719 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 6,719.
- Names: name 6,664, name:en 0, int_name 0, Latin name 6,664, any English-capable name 6,664 (of 6,719).
- Script of `name`: `Latin` 6,664.
- `name:*` tags: none.
- Geometry: the relations add up to 1,899 km² (0.0 % of the outline); 2,421,368 land plots lie in none of them, 42 in two or more; 4811 relations hold no land plot; land plots per relation: 0 to 24, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Area Pengembangan Waduk, RW 01, RW 02, RW 03, RW 05, RW 06, RW 07 Desa Kertamukti, RW 09, RW 12 Kelurahan Donan, Wadul Pluit.

### Level 10

- 6,621 relations in the extract, 4,797 in scope (country). Left out: 0 with another country's ISO 3166-2, 7 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 4,797.
- Names: name 4,758, name:en 0, int_name 0, Latin name 4,757, any English-capable name 4,757 (of 4,797).
- Script of `name`: `Latin` 4,757.
- `name:*` tags: `name:id` 3, `name:ja` 3.
- Geometry: the relations add up to 374 km² (0.0 % of the outline); 2,423,481 land plots lie in none of them, 4 in two or more; 4441 relations hold no land plot; land plots per relation: 0 to 16, median 0.
- Rings not closed in the extract (a member way is missing): 0 in scope; 7 of the level's relations could not be assembled at all.
- Examples: 001, RT 01, RT 02, RT 04, RT 05, RT 07, RT 09, RT 11, RT 18, Wilayah KKP.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 38 relations, all with ISO 3166-2 ID-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 5 plots in none).
- Districts: level 5: 514 relations, 1 with ISO 3166-2 ID-xx, covering 99.9 % of the country's land plots (0.00 % overlap, 3,587 plots in none).

Levels that qualify: 4, 5. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:id-Latn`, `name` (when written in Latin letters). Local name: `name:id`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 38 relations)

- English name from: `name:en` 38; none for 0.
- Local name from: `name:id` 38; none for 0. English equals local for 10.
- English prefixes: “West” 6, “Central” 4, “North” 4, “South” 4, “East” 3. Suffixes: “Kalimantan” 5, “Sulawesi” 5, “Papua” 5, “Sumatra” 3, “Java” 3.
- Local prefixes: “Kalimantan” 5, “Sulawesi” 5, “Papua” 5, “Sumatera” 3, “Jawa” 3. Suffixes: “Barat” 6, “Selatan” 4, “Tengah” 4, “Utara” 4, “Timur” 3.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 5, 514 relations)

- English name from: `name` 399, `name:en` 115; none for 0.
- Local name from: `name` 441, `name:id` 73; none for 0. English equals local for 460.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 13 slugs are used more than once: 1/bima (2), 19/solok (2), 22/kupang (2), 25/blitar (2), 28/jayapura (2), 3/pekalongan (2), 3/tegal (2), 37/sorong (2), 4/bogor (2), 4/cirebon (2).
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-indonesia`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 114,673. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 75,552 | 75,131 | 75,130 |
| hamlet | 27,345 | 26,783 | 26,705 |
| neighbourhood | 3,060 | 2,940 | 2,940 |
| town | 1,907 | 1,896 | 1,896 |
| locality | 1,906 | 1,903 | 1,903 |
| suburb | 1,707 | 1,699 | 1,697 |
| island | 1,320 | 1,316 | 1,316 |
| isolated_dwelling | 425 | 414 | 414 |
| city | 280 | 280 | 280 |
| county | 214 | 214 | 214 |
| quarter | 201 | 200 | 200 |
| city_block | 192 | 192 | 189 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Jakarta (population 10,467,629): village 132 (132 English-capable), locality 36 (36 English-capable), municipality 19 (19 English-capable), neighbourhood 10 (10 English-capable).
- Surabaya (population 2,765,487): village 102 (102 English-capable), municipality 19 (19 English-capable), hamlet 1 (1 English-capable).
- Medan (population 2,551,006): suburb 121 (121 English-capable), village 9 (9 English-capable), neighbourhood 6 (6 English-capable), farm 1 (1 English-capable), locality 1 (0 English-capable).
- Bandung (population 2,390,120): suburb 128 (128 English-capable), neighbourhood 34 (33 English-capable), district 24 (24 English-capable), village 22 (22 English-capable), quarter 6 (6 English-capable), town 5 (5 English-capable).
- Bekasi (population 2,334,871): neighbourhood 78 (78 English-capable), village 37 (37 English-capable), suburb 33 (33 English-capable), quarter 9 (9 English-capable), locality 2 (2 English-capable).

## Size

- Land plots: 2,423,959. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 101 s and 2,227 MiB. A rough figure; it was not measured for this country.

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
