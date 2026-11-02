# Country scan: Mongolia

Written by `go run ./cmd/countryscan -country mn` from `mongolia-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 22 relations, all with ISO 3166-2 MN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 4 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4, 6.
- **Districts:** level 6: 340 relations, none with an ISO 3166-2 code, covering 100.0 % of the country's land plots (0.00 % overlap, 13 plots in none). Per province: 2 to 27, median 17; 0 districts have their centre in no province.
- **Land plots:** 2,225,456.
- **Names (provinces):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 5 names whose slug loses letters without accent folding.
- **Names (districts):** 0 without an English-capable name, 0 with an empty slug, 0 slugs used twice, 44 names whose slug loses letters without accent folding.

## Outline and land

- Outline: the country relation r161033 (admin_level 2); 1,564,516 km² in 1 polygons.
- Cells at resolution 8 with their centre inside the outline: 2,225,456; 2,225,456 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **2,225,456**, about 1,640,161 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with MN-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 3 | 3 | extract | 0 | 3 | 3 | 0 | – | – | – | – |
| 3 | 2 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 29 | 22 | country | 22 | 22 | 3 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 12 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 386 | 340 | country | 0 | 340 | 3 | 0 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 7 | 2 | 1 | country | 0 | 1 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 8 | 124 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 9 | 2 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |

### Level 2

Countries in the extract: China (Хятад), Mongolia (Монгол улс ᠮᠤᠩᠭᠤᠯ ᠤᠯᠤᠰ), Russia (Оросын Холбооны Улс).

### Level 3

- 2 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 2 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 29 relations in the extract, 22 in scope (country). Left out: 7 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `MN` 22; with this country's prefix: 22 of 22.
- Names: name 22, name:en 22, int_name 3, Latin name 0, any English-capable name 22 (of 22).
- Script of `name`: `other` 14, `Cyrillic` 8.
- `name:*` tags: `name:ar` 22, `name:azb` 22, `name:de` 22, `name:en` 22, `name:es` 22, `name:fa` 22, `name:fr` 22, `name:he` 22.
- Geometry: the relations add up to 1,564,516 km² (100.0 % of the outline); 4 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 1,189 to 237,540, median 105,500.
- Rings not closed in the extract (a member way is missing): 0 in scope; 7 of the level's relations could not be assembled at all.
- Examples: Arkhangai (Архангай), Bayankhongor (Баянхонгор), Darkhan-Uul (Дархан-Уул), Dundgovi (Дундговь), Govisumber (Говьсүмбэр), Khentii (Хэнтий), Selenge (Сэлэнгэ), Töv (Төв), Uvs (Увс), Övörkhangai (Өвөрхангай).

### Level 5

- 12 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 12 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 386 relations in the extract, 340 in scope (country). Left out: 0 with another country's ISO 3166-2, 46 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 340.
- Names: name 340, name:en 340, int_name 3, Latin name 0, any English-capable name 340 (of 340).
- Script of `name`: `other` 218, `Cyrillic` 122.
- `name:*` tags: `name:en` 340, `name:de` 339, `name:mn` 339, `name:mn-Mong` 339, `name:mn-Cyrl` 338, `name:fa` 336, `name:zh` 318, `name:ru` 148.
- Geometry: the relations add up to 1,564,516 km² (100.0 % of the outline); 13 land plots lie in none of them, 1 in two or more; 0 relations hold no land plot; land plots per relation: 29 to 48,380, median 4,821.
- Rings not closed in the extract (a member way is missing): 0 in scope; 46 of the level's relations could not be assembled at all.
- Examples: Adaatsag (Адаацаг сум), Bayan-Unjuul (Баян-Өнжүүл сум), Bugat (Бугат сум), Delgerkhaan (Дэлгэрхаан сум), Jargalant (Жаргалант сум), Khüder (Хүдэр сум), Renchinlkhumbe (Рэнчинлхүмбэ сум), Tes (Тэс сум), Tüvshinshiree (Тувшинширээ сум), Üyench (Үенч сум).

### Level 7

- 2 relations in the extract, 1 in scope (country). Left out: 0 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1.
- Names: name 1, name:en 1, int_name 0, Latin name 0, any English-capable name 1 (of 1).
- Script of `name`: `Cyrillic` 1.
- `name:*` tags: `name:de` 1, `name:en` 1, `name:fa` 1, `name:mn` 1, `name:mn-Cyrl` 1, `name:mn-Mong` 1, `name:uk` 1, `name:zh` 1.
- Geometry: the relations add up to 87 km² (0.0 % of the outline); 2,225,338 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 118 to 118, median 118.
- Rings not closed in the extract (a member way is missing): 0 in scope; 1 of the level's relations could not be assembled at all.
- Examples: Khatgal (Хатгал тосгон).

### Level 8

- 124 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 124 with their centre outside the country (or not assembled and without ISO code).

### Level 9

- 2 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 2 with their centre outside the country (or not assembled and without ISO code).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 22 relations, all with ISO 3166-2 MN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 4 plots in none).
- Districts: level 6: 340 relations, none with an ISO 3166-2 code, covering 100.0 % of the country's land plots (0.00 % overlap, 13 plots in none).

Levels that qualify: 4, 6. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:mn-Latn`, `name` (when written in Latin letters). Local name: `name:mn`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 22 relations)

- English name from: `name:en` 22; none for 0.
- Local name from: `name:mn` 22; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: “й” 5.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 5 names, e.g. Sükhbaatar: s-khbaatar → sukhbaatar, Ömnögovi: mn-govi → omnogovi, Övörkhangai: v-rkhangai → ovorkhangai. Stage 2 should fold accents.

### Districts (level 6, 340 relations)

- English name from: `name:en` 340; none for 0.
- Local name from: `name:mn` 339, `name` 1; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: none.
- Local prefixes: none. Suffixes: “сум” 322.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)
- **Accents:** `osmfile.Slug` drops accented letters; with accents folded (NFD, đ → d) the slug is different for 44 names, e.g. Khürmen: kh-rmen → khurmen, Ölziit: lziit → olziit, Öndörshil: nd-rshil → ondorshil. Stage 2 should fold accents.

The slug of a new country's province carries the country as a suffix (`-mongolia`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 795. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| town | 297 | 297 | 297 |
| hamlet | 173 | 111 | 101 |
| village | 77 | 65 | 53 |
| suburb | 72 | 72 | 55 |
| isolated_dwelling | 64 | 43 | 41 |
| locality | 53 | 35 | 29 |
| city | 25 | 25 | 25 |
| state | 21 | 21 | 21 |
| region | 3 | 3 | 1 |
| district | 2 | 2 | 1 |
| farm | 2 | 0 | 0 |
| quarter | 2 | 2 | 0 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Ulaanbaatar (population 1,672,627): suburb 12 (11 English-capable), hamlet 1 (0 English-capable).
- Erdenet (population 86,866): suburb 10 (10 English-capable), neighbourhood 1 (1 English-capable), square 1 (0 English-capable).
- Darkhan (population 74,738): no other place nodes.
- Choibalsan (population 38,537): suburb 38 (26 English-capable), locality 1 (0 English-capable).
- Murun (population 35,789): no other place nodes.

## Size

- Land plots: 2,225,456. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 92 s and 2,045 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Mongolia: from mongolia-latest.osm.pbf
{Code: "mn", Extract: "mongolia",
	Province: Rule{AdminLevel: 4, ISOPrefix: "MN", Count: 22, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:mn-Latn", "name"}, LocalLang: "mn",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, []string{"сум"}}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('mn', '{suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
