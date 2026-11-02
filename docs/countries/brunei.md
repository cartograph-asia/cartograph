# Country scan: Brunei

Written by `go run ./cmd/countryscan -country bn` from `malaysia-singapore-brunei-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 4 relations, all with ISO 3166-2 BN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 4.
- **Districts:** none: no level below 4 covers 90 % of the land plots with more relations than the provinces.
- **Land plots:** 7,864.

## Outline and land

- Outline: the country relation r2103120 (admin_level 2); 8,952 km² in 1 polygons.
- Cells at resolution 8 with their centre inside the outline: 11,901; 7,864 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **7,864**, about 5,795 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with BN-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 0 | 4 | – | – | – | – |
| 3 | 3 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 4 | 32 | 4 | country | 4 | 4 | 0 | 4 | 100.0 % | 0.0 % | 0.00 % | 0 |
| 5 | 37 | 0 | country | 0 | 0 | 0 | 0 | 0.0 % | 100.0 % | 0.00 % | 0 |
| 6 | 284 | 28 | country | 0 | 0 | 0 | 28 | 24.8 % | 75.2 % | 0.00 % | 0 |
| 7 | 356 | 1 | country | 0 | 0 | 0 | 1 | 1.7 % | 98.3 % | 0.00 % | 0 |
| 8 | 86 | 55 | country | 0 | 1 | 0 | 55 | 5.2 % | 94.8 % | 0.00 % | 0 |
| 10 | 380 | 235 | country | 0 | 36 | 0 | 235 | 16.1 % | 83.9 % | 1.04 % | 0 |

### Level 2

Countries in the extract: Brunei, Indonesia, Malaysia, Philippines (Filipina), Singapore (Singapura), Thailand.

### Level 3

- 3 relations in the extract, 0 in scope (country). Left out: 2 with another country's ISO 3166-2, 1 with their centre outside the country (or not assembled and without ISO code).

### Level 4

- 32 relations in the extract, 4 in scope (country). Left out: 28 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `BN` 4; with this country's prefix: 4 of 4.
- Names: name 4, name:en 4, int_name 0, Latin name 4, any English-capable name 4 (of 4).
- Script of `name`: `Latin` 4.
- `name:*` tags: `name:ace` 4, `name:ar` 4, `name:ast` 4, `name:bg` 4, `name:ceb` 4, `name:de` 4, `name:el` 4, `name:en` 4.
- Geometry: the relations add up to 8,952 km² (100.0 % of the outline); 0 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 777 to 3,569, median 1,790.
- Rings not closed in the extract (a member way is missing): 0 in scope; 13 of the level's relations could not be assembled at all.
- Examples: Belait District (Daerah Belait), Brunei-Muara District (Daerah Brunei-Muara), Temburong District (Daerah Temburong), Tutong District (Daerah Tutong).

### Level 5

- 37 relations in the extract, 0 in scope (country). Left out: 0 with another country's ISO 3166-2, 37 with their centre outside the country (or not assembled and without ISO code).

### Level 6

- 284 relations in the extract, 28 in scope (country). Left out: 0 with another country's ISO 3166-2, 256 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 28.
- Names: name 28, name:en 0, int_name 0, Latin name 28, any English-capable name 28 (of 28).
- Script of `name`: `Latin` 28.
- `name:*` tags: `name:zh` 2, `name:ar` 1, `name:ja` 1.
- Geometry: the relations add up to 6,678 km² (74.6 % of the outline); 5,915 land plots lie in none of them, 0 in two or more; 4 relations hold no land plot; land plots per relation: 0 to 222, median 60.
- Rings not closed in the extract (a member way is missing): 0 in scope; 38 of the level's relations could not be assembled at all.
- Examples: Miri, Mukim Burong Pingai Ayer, Mukim Keriam, Mukim Kiudang, Mukim Lamunin, Mukim Mentiri, Mukim Peramu, Mukim Serasa, Mukim Sungai Kedayan, Mukim Telisai.

### Level 7

- 356 relations in the extract, 1 in scope (country). Left out: 0 with another country's ISO 3166-2, 355 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1.
- Names: name 1, name:en 0, int_name 0, Latin name 1, any English-capable name 1 (of 1).
- Script of `name`: `Latin` 1.
- `name:*` tags: `name:uk` 1.
- Geometry: the relations add up to 99 km² (1.1 % of the outline); 7,731 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 133 to 133, median 133.
- Rings not closed in the extract (a member way is missing): 0 in scope; 2 of the level's relations could not be assembled at all.
- Examples: Bandar Seri Begawan.

### Level 8

- 86 relations in the extract, 55 in scope (country). Left out: 0 with another country's ISO 3166-2, 31 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 55.
- Names: name 55, name:en 1, int_name 0, Latin name 55, any English-capable name 55 (of 55).
- Script of `name`: `Latin` 55.
- `name:*` tags: `name:en` 1, `name:ms` 1, `name:uk` 1.
- Geometry: the relations add up to 305 km² (3.4 % of the outline); 7,454 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 1 to 36, median 7.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Baong, Buang Tawar, Kapok, Kulapis, Masin, Pangkalan Sibabau, Riong, Seria, Sungai Tilong, Wasan.

### Level 10

- 380 relations in the extract, 235 in scope (country). Left out: 0 with another country's ISO 3166-2, 144 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 235.
- Names: name 235, name:en 36, int_name 0, Latin name 235, any English-capable name 235 (of 235).
- Script of `name`: `Latin` 235.
- `name:*` tags: `name:ms` 53, `name:en` 36.
- Geometry: the relations add up to 1,034 km² (11.5 % of the outline); 6,597 land plots lie in none of them, 82 in two or more; 48 relations hold no land plot; land plots per relation: 0 to 148, median 2.
- Rings not closed in the extract (a member way is missing): 0 in scope; 3 of the level's relations could not be assembled at all.
- Examples: Anduki, Junjongan, Kampong Kuala Tutong, Kampong Panchor Papan, Kampong Sungai Kang, Lapangan Terbang Antarabangsa Brunei, Pekan Seria Kawasan 1, Perumahan Negara Rimba Kawasan 3, Sumbiling Lama (Kampung Sumbiling Lama), Ujong Tanjong (Kampung Ujong Tanjong).

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 4 relations, all with ISO 3166-2 BN-xx, covering 100.0 % of the country's land plots (0.00 % overlap, 0 plots in none).
- Districts: none: no level below 4 covers 90 % of the land plots with more relations than the provinces.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:ms-Latn`, `name` (when written in Latin letters). Local name: `name:ms`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 4 relations)

- English name from: `name:en` 4; none for 0.
- Local name from: `name:ms` 4; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “District” 4.
- Local prefixes: “Daerah” 4. Suffixes: none.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

The slug of a new country's province carries the country as a suffix (`-brunei`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 206. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 116 | 113 | 113 |
| neighbourhood | 38 | 38 | 38 |
| hamlet | 23 | 18 | 18 |
| locality | 16 | 16 | 16 |
| town | 9 | 8 | 8 |
| quarter | 2 | 2 | 2 |
| city | 1 | 1 | 1 |
| country | 1 | 1 | 1 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Bandar Seri Begawan (population 50,000): village 31 (31 English-capable), neighbourhood 12 (12 English-capable), hamlet 1 (1 English-capable).

## Size

- Land plots: 7,864. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 1 s and 100 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Brunei: from malaysia-singapore-brunei-latest.osm.pbf
{Code: "bn", Extract: "malaysia-singapore-brunei",
	Province: Rule{AdminLevel: 4, ISOPrefix: "BN", Count: 4, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{}, // none
	EnglishName: []string{"name:en", "int_name", "name:ms-Latn", "name"}, LocalLang: "ms",
	StripProvince: Strip{English: [2][]string{nil, []string{"District"}}, Local: [2][]string{[]string{"Daerah"}, nil}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('bn', '{neighbourhood}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 1 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
