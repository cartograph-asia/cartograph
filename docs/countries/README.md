# Country scans

One report per country, written by `cmd/countryscan` (stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`) from the Geofabrik extracts in `map/data/`. A scan is read-only: it imports nothing and touches no database. The same extract gives the same report, so the reports are committed and compared after an extract update (`git diff docs/countries`).

```sh
go run ./cmd/countryscan -country vn     # one country -> docs/countries/vietnam.md
go run ./cmd/countryscan -all            # every country in internal/countries, one after the other
go run ./cmd/countryscan -overlaps       # compare the scans -> docs/countries/overlaps.md
```

Needs the extracts (`map/download.sh`), the water polygons Planetiler fetched into `map/data/sources/`, and osmium (`brew install osmium-tool`; without it a Debian container is used). Each extract is first cut down to its administrative boundaries and place nodes; the cut files and the land cell lists that `-overlaps` compares stay in `map/data/scan/`, which is git-ignored. `-res 7` makes a scan about seven times faster and its plot counts estimates (it leaves no cell list for `-overlaps`).

What a report holds: the administrative levels with their tags, how well each level tiles the country's land plots, a proposed province and district level, the names (tags, repeated prefixes and suffixes, slugs), the place nodes and the largest cities, the number of plots with an estimate of the stage 3 run, and a sketch of the settings and the `place_naming` row.

How to read the numbers:

- **Land plot:** a resolution-8 cell whose centre is inside the country's outline and not in an OSM water polygon. The outline is the country's `admin_level` 2 relation, or, where the extract does not hold it whole, the union of the relations with its ISO 3166-2 code.
- **Cover, gaps, overlap:** of the land plots, the share inside one, no and two or more relations of a level. Boundaries are simplified as the import does (`geo.SimplifyTolerance`).
- **In scope:** relations of the country. A relation counts when its ISO 3166-2 starts with the country's code, or has none and its centre lies in the outline. Levels with more relations than `-geometry-cap` (8,000) are counted and read for their names only, and then include the neighbours' relations.
- **Proposed levels** follow a fixed rule (see the report); they are a starting point for Adrian's decision, not the decision.
