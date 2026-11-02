# Province relations that do not assemble

Nine province relations do not assemble from their own Geofabrik extract (`docs/COUNTRY_IMPORT_STAGE2_BRIEF.md`, Risks; WP F). This note says, for each of them, which member ways are missing, where those ways are, and what the importers need to build the province. Checked on 2026-11-02 against the extracts in `map/data/` (all with the replication timestamp 2026-10-31T21:23:06Z) and their osmium cuts in `map/data/scan/`.

## Summary

- **Eight of the nine assemble whole** when the country's cut is read together with one or two neighbours' cuts that are already in `map/data/`. None of them needs `Partial`.
- **Xizang (r153292) needs an extract we do not have:** ten ways of the Bhutan–China border, which are in no extract in `map/data/`. `Partial` does not rescue it (its one outer ring stays open).
- **The Philippines extract holds the Spratly lines** of four other countries: the territorial-sea ways around the features that Vietnam (Khánh Hòa), Malaysia (Sabah), Taiwan (Kaohsiung) and China (Hainan) hold or claim lie inside Geofabrik's Philippines polygon (the Kalayaan Island Group) and in no other extract.
- **What is missing matters very differently.** For the four Spratly cases a `Partial` build would lose sea and 0–14 land plots. For Riau Islands, North Kalimantan, North Pyongan, Fujian and Xizang a single missing way leaves the province's outer ring open, so without the neighbour's ways the whole province is lost (9,980 to 1.57 million land plots); for North Kalimantan, North Pyongan and Xizang the missing ways are a land border.
- **Every affected country's land plot figure was too low**, because the scan's outline lacked the province: Malaysia by 32 %, North Korea by 11.5 %, Taiwan by 8.9 %, Indonesia by 4.6 %, Vietnam by 2.5 % and China by about 15 %. Malaysia's and North Korea's `LandPlots` would fail the dry run's 10 % check against a whole import; each country PR takes the corrected figure.

## The nine relations

*Own cut* is `map/data/scan/<extract>-latest.osm.pbf.cut.osm.pbf`. *Partial* is what `osmfile.Assemble` with the relation in `partial` builds from the own cut alone. Land plots are resolution-8 cells whose centre lies in the boundary (simplified with `geo.SimplifyTolerance`) and not in an OSM water polygon, as the scan counts them.

| Relation | Country (own extract) | Missing from the own cut | Where they are | Partial from the own cut | Verdict | Setting |
|---|---|---|---|---|---|---|
| Khánh Hòa r1887959 | Vietnam (`vietnam`) | 16 of 62 ways (15 outer, 1 inner) | Territorial-sea lines around Vietnam's Spratly features (Trường Sa), 8.9–11.3° N, 114.0–114.7° E, `admin_level=2 maritime=yes`; in `philippines` | 3 rings, 1 fragment dropped: 27,820 of 40,545 km², 9,812 of 9,816 land plots (4 islets lost) | **a** | neighbour `philippines`; no `Partial` |
| Sabah r3879783 | Malaysia (`malaysia-singapore-brunei`) | 2 of 41 (w1120240919, w1120240920, outer) | Territorial-sea ring around Swallow Reef (Layang-Layang), 7.5–8.4° N, 114.2–114.6° E; in `philippines` | 2 rings: 120,525 of 130,598 km², all 101,512 land plots (nothing on land lost) | **a** | neighbour `philippines`; no `Partial` |
| Riau Islands r3797244 | Indonesia (`indonesia`) | 1 of 29 (w1147865283, outer) | "Singapore-Indonesia" territorial-sea line east of Singapore, 1.33–1.44° N, 104.41–104.57° E; in `malaysia-singapore-brunei` | not closed: the province (9,980 land plots) is lost | **a** | neighbour `malaysia-singapore-brunei` |
| North Kalimantan r5449460 | Indonesia (`indonesia`) | 1 of 35 (w1181737260, outer) | "Border Malaysia - Indonesia", a land border of about 3.5 km at 4.38–4.39° N, 115.87–115.90° E; in `malaysia-singapore-brunei` | not closed: the province (97,350 land plots) is lost | **a** | neighbour `malaysia-singapore-brunei` |
| Kaohsiung r2127079 | Taiwan (`taiwan`) | 2 of 247 (w284788564, w654482084, outer) | w284788564: territorial sea around Pratas (Dongsha), 20.94° N 116.65° E, in `china`; w654482084: restricted waters around Taiping Island (Itu Aba), 10.42° N 114.39° E, in `philippines` | 2 rings: 2,996 of 6,607 km², 4,501 of 4,504 land plots (Pratas and Taiping lost) | **a** | neighbours `china`, `philippines`; no `Partial` |
| North Pyongan r356540 | North Korea (`north-korea`) | 2 of 64 (w218452189, w1032769161, outer) | The Yalu river border with China, about 2.6 km at 40.22–40.23° N, 124.51–124.53° E; in `china` | not closed: the province (24,641 land plots) is lost | **a** | neighbour `china` |
| Xizang r153292 | China (`china`) | 10 of 464 (w298158655, w589204000, w589426237, w888419443, w1008972051, w1337681673, w1452581708, w1452581709, w1453332523, w1454184810, outer) | The Bhutan–China border on the east side of the Chumbi valley (Doklam), between n12373408574 (27.253° N 89.072° E) and n8259769049 (27.495° N 89.098° E), a gap of 27 km; in **no extract we have** (not in the 15 cuts, and `osmium getid` finds them in neither `china-latest.osm.pbf` nor `myanmar-latest.osm.pbf`) | not closed: the province (about 1,573,305 land plots) is lost | **c** | neighbour `bhutan` (Geofabrik `asia/bhutan`, not downloaded) |
| Fujian r553303 | China (`china`) | 27 of 166 (9 outer, 18 inner) | The territorial-sea line between the mainland and Taiwan-held Kinmen and Matsu (outer, 24.0–26.3° N, 118.3–120.7° E) and the Kinmen, Matsu, Dongyin and Wuqiu restricted waters and islets cut out of Fujian as holes (inner); all in `taiwan` | not closed: the province (191,254 land plots) is lost | **a** | neighbour `taiwan` |
| Hainan r2128285 | China (`china`) | 16 of 25 (outer) | Territorial-sea lines of the Spratly features in Sansha, 8.7–11.1° N, 112.7–115.5° E (one tagged `disputed=yes`); in `philippines`, two of them also in `vietnam` | 5 rings: 100,798 of 110,796 km², 43,387 of 43,401 land plots (14 islets lost) | **a** | neighbour `philippines`; no `Partial` |

Verdicts: **a** whole from the cuts of extracts in `map/data/`; **b** usable as `Partial`; **c** needs an extract we do not have. Khánh Hòa, Sabah, Kaohsiung and Hainan could also be imported as **b** (`Partial`) at the cost shown (sea and 0–14 land plots); since their neighbours' cuts are at hand, **a** is the better choice and `Partial` stays empty for all nine.

The North Pyongan, North Kalimantan, Riau Islands and Fujian relations are present in the neighbour's extract too, but the neighbour's copy lacks other ways; only the union of both cuts is whole. Every one of the nine relations is in its own cut, so the relation is always read from there and only ways and nodes come from the neighbours.

## Settings for the importers

Read the provinces and districts from the country's own cut, completed with ways and nodes from the neighbours' cuts; read places from the own cut only. Two ways to do that, both in this branch:

- **`osmfile.ReadBoundaryDataWith(cut, neighbourCuts, keep)`** reads the relations from the own cut only and their ways and nodes from all files, keeping the highest version of an object found twice. This is what the importer should call: no extra file, and a neighbour's relations are never read (a district level read with China's cut as a merged file would otherwise pull in China's 19,789 level-8 relations).
- **`map/merge-cut.sh <extract> <neighbour> …`** writes `map/data/scan/<extract>+<neighbour>-merged.cut.osm.pbf` (host osmium, else a Debian container; skipped when newer than its cuts; refuses cuts of different days). It is for the scan (`cmd/countryscan -pbf <merged> -no-cut`) and for a single `-pbf` override of `regions`. It holds the neighbours' place nodes and relations, so `places` must not read it: a place outside every province is given the run's country, and the merged file would hand it the whole neighbour.

The settings type has no field for the neighbours yet. Proposed for WP C or the country PRs (the lead decides): `Import.Neighbours []string`, Geofabrik names whose cuts complete the boundaries, never read for places and never added to the base map's `EXTRACTS`.

| Country | `Neighbours` | `Partial` | Merged cut for the scan |
|---|---|---|---|
| Vietnam | `philippines` | – | `map/merge-cut.sh vietnam philippines` |
| Malaysia (and Brunei, Singapore, same extract) | `philippines` | – | `map/merge-cut.sh malaysia-singapore-brunei philippines` |
| Indonesia | `malaysia-singapore-brunei` | – | `map/merge-cut.sh indonesia malaysia-singapore-brunei` |
| Taiwan | `china`, `philippines` | – | `map/merge-cut.sh taiwan china philippines` |
| North Korea | `china` | – | `map/merge-cut.sh north-korea china` |
| China | `taiwan`, `philippines`, `bhutan` | – | `map/merge-cut.sh china taiwan philippines bhutan` (after `bhutan` is downloaded and cut) |

Until such a field exists, `regions -country vn -pbf map/data/scan/vietnam+philippines-merged.cut.osm.pbf` with `places -country vn` reading its own cut should give the same provinces (the regions run then also assembles the neighbour's relations of the district level and drops them by their centre). Production needs the neighbours' cuts beside the country's (the cuts are a few MB, China's 66 MB), not their full extracts.

## Effect on land plots and the reports

Each country scanned again from its merged cut (`cmd/countryscan -country xx -pbf <merged> -no-cut`, reports in a scratch directory, not committed). With the neighbours' ways the country relation itself becomes whole for Vietnam, Malaysia, Indonesia, Taiwan and North Korea, so the outline is the admin_level-2 relation instead of the union of the provinces.

| Country | Land plots, report | From the merged cut | Change | Districts in scope, report → merged |
|---|---:|---:|---:|---|
| Vietnam | 390,593 | 400,417 | +9,824 (+2.5 %) | 3,254 → 3,319 (Khánh Hòa's communes) |
| Malaysia | 318,281 | 419,805 | +101,524 (+31.9 %) | 130 → 157 (Sabah's districts) |
| Indonesia | 2,316,624 | 2,423,959 | +107,335 (+4.6 %) | 502 → 514 |
| Taiwan | 51,028 | 55,564 | +4,536 (+8.9 %) | see below |
| North Korea | 214,569 | 239,213 | +24,644 (+11.5 %) | 171 → 195 (North Pyongan's counties) |
| China | 11,976,594 | 12,211,248 (Xizang still missing) | see China | |

The differences to the sum of the provinces' own land plots (Vietnam +9,816, Malaysia +101,512, Indonesia +107,330, Taiwan +4,504, North Korea +24,641) are 3 to 32 cells, from the country relation's outline against the union of the provinces.

- **`LandPlots` must be updated** by each country PR from a scan of the merged cut. Against the reports' figures a whole import is off by more than 10 % for Malaysia and North Korea, so the dry run's check 4 would fail.
- **Reading the merged reports:** the level tables count the neighbours' relations in *In extract* (Vietnam's level 4: 145 instead of 59), the *in scope* columns are right. Taiwan's merged cut contains China's 19,789 level-8 relations, more than the scan's geometry cap, so level 8 is not built and no district level is proposed; take only the land plot figure from it and the rest from the own-extract report.
- **Spratly overlaps:** whole builds of Khánh Hòa, Kaohsiung and Hainan (and of the Philippines' own Kalayaan) put a few Spratly islets into several countries; Sabah's ring holds no land plot. A plot in two countries belongs to the country imported first (`docs/COUNTRY_IMPORT_BRIEF.md`, decisions 4 and 5); `overlaps.md` should be written again from the merged scans before those imports, and Adrian decides each overlap.

## China

China's scan used the union of 28 provinces; Xizang, Fujian and Hainan were missing from its outline.

- Fujian and Hainan assemble whole from `china+taiwan+philippines`: 191,254 and 43,401 land plots.
- Xizang needs `asia/bhutan`. As a bounded estimate it was built with the gap closed by a straight line between the two open ends: 1,192,981 km², **1,573,305 land plots** (all inland). The true border between the two ends is not in our data; if it stays within 10 km of that 27 km line, the error is a sliver of at most about 270 km², some 360 plots (0.02 % of the province), either way.
- China's country relation stays not whole even in the merged cut, so the outline is still the union of provinces.

| | Land plots |
|---|---:|
| Scan report (28 provinces) | 11,976,594 |
| Scan of `china+taiwan+philippines` (30 provinces, Xizang missing) | 12,211,248 |
| Union of the report's cells with Fujian, Hainan and Xizang (closed by the straight line); the same as the merged scan plus Xizang's 1,573,305 | 13,784,553 |

**Corrected estimate: about 13.78 million land plots** (13,784,553, ± about 400 for the Doklam gap), 15 % more than the report. The exact figure needs the Bhutan extract: `map/download.sh bhutan`, cut it (`go run ./cmd/countryscan` cuts on the way, or `map/cut.sh bhutan`), `map/merge-cut.sh china taiwan philippines bhutan`, and scan China from that merged cut. Bhutan's extract is only read for boundaries and never joins the base map's `EXTRACTS` (the brief keeps South Asia out). Its size was not checked here (no downloads in this package); Bhutan is one of Geofabrik's small Asian extracts, check `download.geofabrik.de/asia/bhutan.html` before fetching. If its cut does not close the gap, the next candidate is `asia/india` (well over 1 GB), read the same way.

## How this was checked

- A temporary test (not committed) read the nine relations from every cut in `map/data/scan/` with `ReadBoundaryDataWith`, listed each relation's member ways per cut, the ways missing from the own cut with their tags and end points, the open ends of the rings, and ran `Assemble` on the own cut (with and without `partial`) and on the union of the cuts.
- `osmium getid` looked for Xizang's ten ways in the full `china` and `myanmar` extracts; none is there.
- The six merged cuts were made with `map/merge-cut.sh`; on each, `ReadBoundaryData` + `Assemble` (no `partial`) built every level-4 relation with the country's ISO code: Vietnam 34 of 34, Malaysia 16 of 16, Indonesia 38 of 38, Taiwan 22 of 22, North Korea 13 of 13, China 30 of 31 (Xizang not whole).
- Land plots per province from the built geometry as the scan counts them; the country figures from `cmd/countryscan` on the merged cuts.
