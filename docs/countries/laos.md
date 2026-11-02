# Country scan: Laos

Written by `go run ./cmd/countryscan -country la` from `laos-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 18 relations, all with ISO 3166-2 LA-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 7 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 148 relations, none with an ISO 3166-2 code, covering 100.0 % of the country's land plots (0.01 % overlap, 13 plots in none). Per province: 4 to 15, median 8; 0 districts have their centre in no province.
- **Land plots:** 275,740.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 1 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r49903 (admin_level 2); 230,916 km² in 1 polygons.
- Cells at resolution 8 with their centre inside the outline: 275,740; 275,740 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **275,740**, about 203,220 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with LA-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 5 | 1 | – | – | – | – |
| 3 | 1 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 44 | 18 | country | 18 | 18 | 0 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 2 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 310 | 148 | country | 0 | 148 | 0 | 1 | 100.0 % | 0.0 % | 0.01 % | 0 |
| 7 | 6 | 1 | country | 0 | 1 | 0 | 0 | 0.2 % | 99.8 % | 0.00 % | 0 |
| 8 | 5 | 2 | country | 0 | 2 | 0 | 0 | 0.2 % | 99.8 % | 0.00 % | 0 |
| 10 | 4 | 4 | country | 0 | 2 | 0 | 2 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Cambodia (ປະເທດກຳປູເຈຍ), China (ປະເທດຈີນ), Laos (ປະເທດລາວ), Myanmar (ປະເທດມຽນມາ), Thailand (ປະເທດໄທ), Vietnam (ປະເທດຫວຽດນາມ).

### Level 3

- 1 relation in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 44 relations in the extract, 18 in scope (country). Left out: 26 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `LA` 18; with this country's prefix: 18 of 18.
- Names: name 18, name:en 18, int_name 0, Latin name 0, any English-capable name 18 (of 18).
- Script of `name`: `Lao` 18.
- `name:*` tags: `name:de` 18, `name:en` 18, `name:es` 18, `name:fr` 18, `name:hi` 18, `name:it` 18, `name:ja` 18, `name:ko` 18.
- Geometry: the relations add up to 230,916 km² (100.0 % of the outline); 7 land plots lie in none of them, 4 in two or more; 0 relations hold no land plot; land plots per relation: 4,229 to 25,587, median 15,334.
- Rings not closed in the extract (a member way is missing): 0 in scope; 26 of the level's relations could not be assembled at all.
- Examples: Attapeu (ອັດຕະປື), Bokeo Province (ບໍ່ແກ້ວ), Champasak Province (ຈໍາປາສັກ), Khammouane (ຄໍາມ່ວນ), Luang Prabang (ຫລວງພະບາງ), Phongsaly (ຜົ້ງສາລີ), Salavan Province (ສາລະວັນ), Sekong Province (ເຊກອງ), Vientiane Province (ວຽງຈັນ), Xiangkhouang Province (ຊຽງຂວາງ).

### Level 5

- 2 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 2 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 310 relations in the extract, 148 in scope (country). Left out: 0 with another country's ISO 3166-2, 162 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 148.
- Names: name 148, name:en 148, int_name 0, Latin name 1, any English-capable name 148 (of 148).
- Script of `name`: `Lao` 147, `Latin` 1.
- `name:*` tags: `name:en` 148, `name:lo` 142, `name:vi` 113, `name:it` 106, `name:uk` 97, `name:th` 56, `name:ur` 18, `name:fr` 15.
- Geometry: the relations add up to 230,916 km² (100.0 % of the outline); 13 land plots lie in none of them, 15 in two or more; 0 relations hold no land plot; land plots per relation: 32 to 5,217, median 1,794.
- Rings not closed in the extract (a member way is missing): 0 in scope; 148 of the level's relations could not be assembled at all.
- Examples: Anouvong District (ອະນຸວົງ), Feuang District (ເຟືອງ), Khanthaboury District (ເມືອງຊົນນະບູລ), May District (ເມືອງໃໝ່), Nong Bok District (ເມືອງໜອງບົກ), Pha Oudom District (ເມືອງຜາອຸດົມ), Sanasomboun District (ເມືອງຊະນະສົມບູນ), Thakhek District (ເມືອງທ່າແຂກ), Viraboury District (ເມືອງວິລະບູລີ), Yot Ou District (ເມືອງຍອດອ).

### Level 7

- 6 relations in the extract, 1 in scope (country). Left out: 0 with another country's ISO 3166-2, 5 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1.
- Names: name 1, name:en 1, int_name 0, Latin name 0, any English-capable name 1 (of 1).
- Script of `name`: `Lao` 1.
- `name:*` tags: `name:af` 1, `name:am` 1, `name:ar` 1, `name:az` 1, `name:be` 1, `name:bg` 1, `name:bn` 1, `name:bo` 1.
- Geometry: the relations add up to 554 km² (0.2 % of the outline); 275,077 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 663 to 663, median 663.
- Rings not closed in the extract (a member way is missing): 0 in scope; 4 of the level's relations could not be assembled at all.
- Examples: Vientiane Capital (ວຽງຈັນ).

### Level 8

- 5 relations in the extract, 2 in scope (country). Left out: 0 with another country's ISO 3166-2, 3 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2.
- Names: name 2, name:en 2, int_name 0, Latin name 0, any English-capable name 2 (of 2).
- Script of `name`: `Lao` 2.
- `name:*` tags: `name:en` 2, `name:lo-Latn` 1.
- Geometry: the relations add up to 436 km² (0.2 % of the outline); 275,208 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 64 to 468, median 468.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Simmano (ສີມມະໂນ), Sophoun (ສົບຮຸນ).

### Level 10

- 4 relations in the extract, 4 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 4.
- Names: name 4, name:en 2, int_name 0, Latin name 2, any English-capable name 4 (of 4).
- Script of `name`: `Lao` 2, `Latin` 2.
- `name:*` tags: `name:en` 2, `name:lo` 2, `name:zh-Hans` 2.
- Geometry: the relations add up to 30 km² (0.0 % of the outline); 275,703 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 20, median 14.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Inner Special Zone (ເຂດຊັ້ນໃນ), Namcheng, Outer Special Zone (ເຂດ​ຊັ້ນນອກ), Phonesitay.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 18 relations, all with ISO 3166-2 LA-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 7 plots in none).
- Districts: level 6: 148 relations, none with an ISO 3166-2 code, covering 100.0 % of the country's land plots (0.01 % overlap, 13 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:lo-Latn`, `name` (when written in Latin letters). Local name: `name:lo`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 18 relations)

- English name from: `name:en` 18; none for 0.
- Local name from: `name:lo` 18; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “Province” 9.
- Local prefixes: none. Suffixes: “ນ” 6, “ັນ” 4.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 148 relations)

- English name from: `name:en` 148; none for 0.
- Local name from: `name:lo` 142, `name` 6; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “District” 148.
- Local prefixes: “ເ” 112, “ເມືອງ” 111. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 1 names, e.g. Pak Sé District: pak-s-district → pak-se-district. Stage 2 should fold accents.

The slug of a new country's province carries the country as a suffix (`-laos`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 7,492. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 6,792 | 6,162 | 6,057 |
| hamlet | 457 | 94 | 93 |
| town | 110 | 108 | 108 |
| isolated_dwelling | 41 | 10 | 10 |
| locality | 21 | 20 | 20 |
| city | 16 | 16 | 15 |
| state | 16 | 16 | 16 |
| quarter | 13 | 13 | 13 |
| neighbourhood | 11 | 10 | 10 |
| farm | 5 | 5 | 5 |
| settlement | 3 | 0 | 0 |
| suburb | 3 | 3 | 3 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Vientiane (population 783,000): village 129 (115 English-capable), locality 1 (1 English-capable).
- Pakse (population 120,000): village 57 (57 English-capable), hamlet 1 (1 English-capable).
- Savannakhet (population 120,000): village 10 (7 English-capable).
- Thakhek (population 70,000): village 25 (25 English-capable).
- Luang Prabang (population 53,792): village 47 (31 English-capable), hamlet 3 (0 English-capable), farm 2 (2 English-capable), neighbourhood 2 (2 English-capable), town 1 (1 English-capable).

## Size

- Land plots: 275,740. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 11 s and 253 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Laos: from laos-latest.osm.pbf
{Code: "la", Extract: "laos",
	Province: Rule{AdminLevel: 4, ISOPrefix: "LA", Count: 18, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:lo-Latn", "name"}, LocalLang: "lo",
	StripProvince: Strip{English: [2][]string{nil, []string{"Province"}}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, []string{"District"}}, Local: [2][]string{[]string{"ເ", "ເມືອງ"}, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('la', '{}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
