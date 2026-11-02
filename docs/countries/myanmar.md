# Country scan: Myanmar

Written by `go run ./cmd/countryscan -country mm` from `myanmar-latest.osm.pbf`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.

## Summary

- **Provinces:** level 4: 17 relations, 15 with ISO 3166-2 MM-xx, covering 99.8 % of the country's land plots (2.88 % overlap, 1,641 plots in none).
  Levels that tile the land well enough (cover ≥ 90 %, overlap ≤ 2 %): 6, 7.
- **Districts:** level 6: 121 relations, none with an ISO 3166-2 code, covering 97.5 % of the country's land plots (0.14 % overlap, 20,017 plots in none). Per province: 0 to 28, median 6; 0 districts have their centre in no province.
- **Land plots:** 804,966.

## Outline and land

- Outline: the country relation r50371 (admin_level 2); 822,567 km² in 3 polygons.
- Cells at resolution 8 with their centre inside the outline: 975,940; 804,966 of them are land (not in an OSM water polygon).
- Land plots (resolution 8): **804,966**, about 593,259 km².

## Administrative levels

`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. *In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with MM-, or they have none and their centre lies in the outline); for a level with more than 8,000 relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. Cover, gaps and overlap are shares of the country's land plots (resolution 8) that lie in one, in no and in two or more relations of the level.

| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |
|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 6 | 6 | extract | 0 | 6 | 5 | 1 | – | – | – | – |
| 3 | 4 | 4 | country | 0 | 4 | 0 | 0 | 7.8 % | 92.2 % | 0.00 % | 0 |
| 4 | 36 | 17 | country | 15 | 16 | 14 | 0 | 99.8 % | 0.2 % | 2.88 % | 0 |
| 5 | 31 | 2 | country | 0 | 2 | 1 | 0 | 0.3 % | 99.7 % | 0.00 % | 0 |
| 6 | 235 | 121 | country | 0 | 121 | 0 | 0 | 97.5 % | 2.5 % | 0.14 % | 0 |
| 7 | 364 | 358 | country | 0 | 357 | 0 | 1 | 97.8 % | 2.2 % | 0.23 % | 0 |
| 8 | 46 | 6 | country | 0 | 4 | 0 | 0 | 0.2 % | 99.8 % | 0.00 % | 0 |
| 9 | 23 | 23 | country | 0 | 23 | 0 | 0 | 0.3 % | 99.7 % | 0.00 % | 0 |
| 10 | 2,040 | 1,999 | country | 0 | 1,963 | 0 | 0 | 0.3 % | 99.7 % | 0.00 % | 0 |

### Level 2

Countries in the extract: Bangladesh (ဘင်္ဂလားဒေ့ရှ်နိုင်ငံ), China (တရုတ်), India (အိန္ဒိယ), Laos (လာအိုနိုင်ငံ), Myanmar (မြန်မာ), Thailand (ထိုင်းနိုင်ငံ).

### Level 3

- 4 relations in the extract, 4 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 4.
- Names: name 4, name:en 4, int_name 0, Latin name 0, any English-capable name 4 (of 4).
- Script of `name`: `Myanmar` 3, `Han` 1.
- `name:*` tags: `name:en` 4, `name:my` 4, `name:zh` 4, `name:zh-Hans` 2, `name:zh-Hant` 2, `name:br` 1, `name:de` 1, `name:fr` 1.
- Geometry: the relations add up to 51,709 km² (6.3 % of the outline); 742,340 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 7,772 to 23,167, median 17,726.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Eastern Shan State Special Region 4 (ရှမ်းပြည်နယ်အရှေ့ပိုင်း အထူးဒေသ(၄)), Special Region 1 (အထူးဒေသ (၁)), Ta'ang State (တအာင်းပြည်နယ်), Wa State ('ဝ' ပြည်နယ်).

### Level 4

- 36 relations in the extract, 17 in scope (country). Left out: 19 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: `MM` 15; with this country's prefix: 15 of 17.
- Names: name 17, name:en 16, int_name 14, Latin name 0, any English-capable name 17 (of 17).
- Script of `name`: `Myanmar` 17.
- `name:*` tags: `name:zh` 17, `name:en` 16, `name:my` 16, `name:de` 15, `name:es` 15, `name:fr` 15, `name:ja` 15, `name:ko` 15.
- Geometry: the relations add up to 758,788 km² (92.2 % of the outline); 1,641 land plots lie in none of them, 23,171 in two or more; 0 relations hold no land plot; land plots per relation: 8,024 to 187,664, median 38,666.
- Rings not closed in the extract (a member way is missing): 0 in scope; 19 of the level's relations could not be assembled at all.
- Examples: Ayeyarwady (ဧ​ရာဝတီတိုင်း), Bago Region (ပဲခူးတိုင်းဒေသကြီး), Kachin State (ကချင်ပြည်နယ်), Kayin State (ကရင်ပြည်နယ်), Mandalay (မန္တလေးတိုင်း), Mon State (မွန်ပြည်နယ်), Northern Wa State (မြောက်ပိုင်း 'ဝ' ပြည်နယ်), Sagaing Region (စစ်ကိုင်းတိုင်းဒေသကြီး), Tanintharyi Region (တနင်္သာရီတိုင်း), Yangon (ရန်ကုန်တိုင်းဒေသကြီး).

### Level 5

- 31 relations in the extract, 2 in scope (country). Left out: 3 with another country's ISO 3166-2, 26 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 2.
- Names: name 2, name:en 2, int_name 1, Latin name 0, any English-capable name 2 (of 2).
- Script of `name`: `Myanmar` 2.
- `name:*` tags: `name:en` 2, `name:my` 2, `name:pl` 2, `name:sk` 2, `name:zh` 2, `name:bn` 1, `name:ca` 1, `name:cs` 1.
- Geometry: the relations add up to 1,917 km² (0.2 % of the outline); 802,724 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 1,097 to 1,145, median 1,145.
- Rings not closed in the extract (a member way is missing): 0 in scope; 29 of the level's relations could not be assembled at all.
- Examples: Mandalay City (မန္တလေးမြို့), Yangon City (ရန်ကုန်မြို့).

### Level 6

- 235 relations in the extract, 121 in scope (country). Left out: 0 with another country's ISO 3166-2, 114 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 121.
- Names: name 121, name:en 121, int_name 0, Latin name 0, any English-capable name 121 (of 121).
- Script of `name`: `Myanmar` 117, `Latin` 4.
- `name:*` tags: `name:en` 121, `name:zh` 121, `name:my` 118, `name:th` 40, `name:zh-Hant` 22, `name:sv` 21, `name:zh-Hans` 17, `name:pl` 14.
- Geometry: the relations add up to 719,277 km² (87.4 % of the outline); 20,017 land plots lie in none of them, 1,118 in two or more; 0 relations hold no land plot; land plots per relation: 12 to 35,962, median 4,368.
- Rings not closed in the extract (a member way is missing): 0 in scope; 107 of the level's relations could not be assembled at all.
- Examples: Ahlon District (အလုံခရိုင်), Falam District (ဖလမ်းခရိုင်), Kale District (ကလေးခရိုင်), Kyauktada District (ကျောက်တံတားခရိုင်), Maungdaw District (မောင်တောခရိုင), Mong Hsu District (မိုင်းရှူးခရိုင်), Myitkyina District (မြစ်ကြီးနားခရိုင်), Pyin Oo Lwin District (ပြင်ဦးလွင်ခရိုင်), Taungup District (တောင်ကုတ်ခရိုင်), Zeyathiri District (ဇေယျာသီရိခရိုင်).

### Level 7

- 364 relations in the extract, 358 in scope (country). Left out: 0 with another country's ISO 3166-2, 6 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 358.
- Names: name 358, name:en 357, int_name 0, Latin name 1, any English-capable name 358 (of 358).
- Script of `name`: `Myanmar` 332, `Latin` 25, `Han` 1.
- `name:*` tags: `name:zh` 358, `name:en` 357, `name:my` 287, `name:zh-Hant` 65, `name:zh-Hans` 49, `name:prk` 25, `name:uk` 21, `name:sv` 18.
- Geometry: the relations add up to 722,028 km² (87.8 % of the outline); 17,611 land plots lie in none of them, 1,846 in two or more; 1 relations hold no land plot; land plots per relation: 0 to 15,290, median 1,573.
- Rings not closed in the extract (a member way is missing): 0 in scope; 5 of the level's relations could not be assembled at all.
- Examples: Ahlone (အလုံ), Falam Township (ဖလမ်းမြို့နယ်), Kani Township (ကနီမြို့နယ်), Lahe Township (လဟယ်မြို့နယ်), Meung Phien Township (မိုင်းဖျန်မြို့နယ်), Myaungmya Township (မြောင်းမြမြို့နယ်), Pale Township (ပုလဲမြို့နယ်), Seikgyikanaungto (ဆိပ်ကြီးခနောင်တို), Thandwe Township (သံတွဲ), Zigon Township (ဇီးကုန်းမြို့နယ်).

### Level 8

- 46 relations in the extract, 6 in scope (country). Left out: 0 with another country's ISO 3166-2, 40 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 6.
- Names: name 6, name:en 4, int_name 0, Latin name 0, any English-capable name 4 (of 6).
- Script of `name`: `Myanmar` 6.
- `name:*` tags: `name:my` 6, `name:en` 4, `name:zh` 4, `name:zh-Hans` 3, `name:zh-Hant` 3.
- Geometry: the relations add up to 1,058 km² (0.1 % of the outline); 803,707 land plots lie in none of them, 0 in two or more; 0 relations hold no land plot; land plots per relation: 14 to 726, median 154.
- Rings not closed in the extract (a member way is missing): 0 in scope; 31 of the level's relations could not be assembled at all.
- Examples: Kutkai Town (ကွတ်ခိုင်မြို့), Man Au Village-tract (မန်အူကျေးရွာအုပ်စု), Ponparkyin Sub-Township (ပုံပါကျင်မြို့နယ်ခွဲ), Tangyan Town (တန့်ယန်းမြို့), နမ့်ခပ်ဒိုင်နယ်, ဝမ်ခိုးဒိုင်နယ်.

### Level 9

- 23 relations in the extract, 23 in scope (country). Left out: 0 with another country's ISO 3166-2, 0 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 23.
- Names: name 23, name:en 23, int_name 0, Latin name 0, any English-capable name 23 (of 23).
- Script of `name`: `Myanmar` 22, `Thai` 1.
- `name:*` tags: `name:en` 23, `name:my` 22, `name:zh` 10, `name:zh-Hans` 4, `name:zh-Hant` 4, `name:shn` 3, `name:th` 1.
- Geometry: the relations add up to 1,644 km² (0.2 % of the outline); 802,884 land plots lie in none of them, 0 in two or more; 3 relations hold no land plot; land plots per relation: 0 to 1,444, median 3.
- Rings not closed in the extract (a member way is missing): 0 in scope; 0 of the level's relations could not be assembled at all.
- Examples: Hka Lon Village Tract (ခလုံးကျေးရွာအုပ်စု), Hopong Town (ဟိုပုန်းမြို့), Kengtung Town (ကျိုင်းတုံမြို့), Mong Yang Town (မိုင်းယန်းမြို့), Muse Town (မူဆယ်မြို့), Shingbwiyang (ရှင်ဗွေယန်မြို့), Ward 10 (ရပ်ကွက် (၁၀)), Ward 4 (ရပ်ကွက်(၄)), Ward 6 (ရပ်ကွက် (၆)), Ward 9 (ရပ်ကွက် (၉)).

### Level 10

- 2,040 relations in the extract, 1,999 in scope (country). Left out: 0 with another country's ISO 3166-2, 40 with their centre outside the country (or not assembled and without ISO code).
- ISO 3166-2 prefixes: none; with this country's prefix: 0 of 1,999.
- Names: name 1,970, name:en 1,963, int_name 0, Latin name 0, any English-capable name 1,963 (of 1,999).
- Script of `name`: `Myanmar` 1,970.
- `name:*` tags: `name:en` 1,963, `name:my` 1,939, `name:zh` 76, `name:zh-Hans` 17, `name:zh-Hant` 17, `name:shn` 12.
- Geometry: the relations add up to 2,126 km² (0.3 % of the outline); 802,518 land plots lie in none of them, 40 in two or more; 791 relations hold no land plot; land plots per relation: 0 to 71, median 1.
- Rings not closed in the extract (a member way is missing): 0 in scope; 27 of the level's relations could not be assembled at all.
- Examples: (Ga Gyi) Ward ((ဃ) ရပ်ကွက်), Block 10 (အပိုင်း ၁၀), Kyar Kan Ward (ကြာကန်ရပ်ကွက်), Myo Ma Kwet Thit Ward (မြို့မကွက်သစ်ရပ်ကွက်), No (112) Ward (အမှတ် (၁၁၂) ရပ်ကွက်), No (3) Ward (အမှတ် (၃) ရပ်ကွက်), No (7) Ward (အမှတ် (၇) ရပ်ကွက်), Pyin Oo Lwin Ward (ပြင်ဦးလွင်ရပ်ကွက်), Thauk Taw Kwin Ward (သောက်တော်ကွင်းရပ်ကွက်), ၼႃးမႂ်ႇ.

## Proposal: province and district levels

The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 % of them with this country's ISO 3166-2 code, covering 90 % or more of the land plots with at most 5 % overlap; without ISO codes the same with at most 2 % overlap. The district level is the next level with more relations that covers 90 % or more with at most 2 % overlap. It is a proposal; the table above is what to decide by.

- Provinces: level 4: 17 relations, 15 with ISO 3166-2 MM-xx, covering 99.8 % of the country's land plots (2.88 % overlap, 1,641 plots in none).
- Districts: level 6: 121 relations, none with an ISO 3166-2 code, covering 97.5 % of the country's land plots (0.14 % overlap, 20,017 plots in none).

Levels that qualify: 6, 7. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.

## Names

English-capable name: the first of `name:en`, `int_name`, `name:my-Latn`, `name` (when written in Latin letters). Local name: `name:my`, else `name`. Prefixes and suffixes are the first and last words shared by at least 10 % of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %.

### Provinces (level 4, 17 relations)

- English name from: `name:en` 16, `int_name` 1; none for 0.
- Local name from: `name:my` 16, `name` 1; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “State” 6, “Region” 4.
- Local prefixes: none. Suffixes: “ြည်နယ်” 7, “း” 7, “ေသကြီး” 4.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.

### Districts (level 6, 121 relations)

- English name from: `name:en` 121; none for 0.
- Local name from: `name:my` 118, `name` 3; none for 0. English equals local for 0.
- English prefixes: none. Suffixes: “District” 110.
- Local prefixes: “မ” 33. Suffixes: “်” 114, “ခရိုင်” 112.
- Slugs: 0 names have no slug (no English-capable name, or nothing in it that a slug can hold); 0 slugs are used more than once.
  (A district's slug is built from its province's; the duplicates above are within one province.)

The slug of a new country's province carries the country as a suffix (`-myanmar`), decided on 2026-11-02, so it cannot collide with another country's.

## Places

`place=*` nodes inside the outline: 58,583. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.

| place | Nodes | Named | English-capable |
|---|---:|---:|---:|
| village | 54,313 | 53,976 | 53,340 |
| quarter | 2,116 | 2,093 | 2,047 |
| hamlet | 1,234 | 987 | 685 |
| town | 508 | 501 | 494 |
| locality | 113 | 111 | 72 |
| isolated_dwelling | 79 | 8 | 8 |
| suburb | 65 | 64 | 54 |
| city | 62 | 62 | 62 |
| neighbourhood | 40 | 37 | 37 |
| state | 15 | 15 | 15 |
| city_block | 14 | 14 | 6 |
| island | 10 | 6 | 6 |

Place nodes within 8 km of the largest cities (by `population`), the input for `urban_kinds`:

- Yangon (population 5,160,512): quarter 347 (347 English-capable), suburb 20 (20 English-capable), village 9 (9 English-capable), neighbourhood 5 (5 English-capable), locality 4 (3 English-capable).
- Mandalay (population 1,208,099): quarter 87 (87 English-capable), village 37 (37 English-capable), suburb 5 (5 English-capable), city_block 1 (1 English-capable), town 1 (1 English-capable).
- Nay Pyi Taw (population 1,160,242): village 20 (20 English-capable), quarter 15 (15 English-capable), town 2 (2 English-capable).
- Hpa-An (population 421,575): village 47 (44 English-capable), quarter 9 (9 English-capable).
- Mawlamyine (population 325,927): village 36 (35 English-capable), quarter 27 (27 English-capable), locality 1 (1 English-capable), suburb 1 (1 English-capable), town 1 (1 English-capable).

## Size

- Land plots: 804,966. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).
- Stage 3, scaled linearly from the Thailand measurements (598,404 land plots, 25 s, about 550 MiB): about 33 s and 739 MiB. A rough figure; it was not measured for this country.

## Proposed configuration

A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.

```go
// Myanmar: from myanmar-latest.osm.pbf
{Code: "mm", Extract: "myanmar",
	Province: Rule{AdminLevel: 4, ISOPrefix: "MM", Count: 17, Kind: "province"}, // kind word to decide: province, prefecture, state, region
	District: Rule{AdminLevel: 6},
	EnglishName: []string{"name:en", "int_name", "name:my-Latn", "name"}, LocalLang: "my",
	StripProvince: Strip{English: [2][]string{nil, nil}, Local: [2][]string{nil, nil}}, // prefixes, suffixes shared by at least half of the names
	StripDistrict: Strip{English: [2][]string{nil, []string{"District"}}, Local: [2][]string{nil, []string{"်", "ခရိုင်"}}}, // prefixes, suffixes shared by at least half of the names
},

-- place_naming (a migration of its own, numbered by the lead)
insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values
  ('mm', '{quarter,neighbourhood,suburb}', 3, '{village|town,city}', 5);
```

`place_naming`: English-named nodes within 8 km of the 5 largest cities (by population tag): kinds with at least 5 are proposed; the radii and tiers are Thailand's, not measured here.
