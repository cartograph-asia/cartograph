# Country scan: overlaps between countries

Written by `go run ./cmd/countryscan -overlaps` from the land plots (resolution 8) each country scan left behind. A plot is listed when its centre lies inside the outline of two countries and is not in water. OSM's de facto boundaries apply; each overlap is decided by Adrian and the decision gets a line in `docs/decisions.md` (`docs/COUNTRY_IMPORT_BRIEF.md`, question 5). A plot in both countries belongs to the country imported first and keeps its number.

| Countries | Plots | About km² | Between (lat, lng) | Examples |
|---|---:|---:|---|---|
| Cambodia / Laos | 1 | 0 | 14.26, 105.25 to 14.26, 105.25 | 14.2621, 105.2457 |
| Cambodia / Vietnam | 1 | 0 | 12.90, 107.49 to 12.90, 107.49 | 12.8978, 107.4915 |
| China / Mongolia | 1 | 0 | 42.76, 110.40 to 42.76, 110.40 | 42.7624, 110.3998 |
| China / Myanmar | 1 | 0 | 21.60, 101.17 to 21.60, 101.17 | 21.6042, 101.1700 |
| China / North Korea | 1 | 0 | 42.57, 130.52 to 42.57, 130.52 | 42.5665, 130.5212 |
| Indonesia / Timor-Leste | 1 | 0 | -9.35, 124.07 to -9.35, 124.07 | -9.3508, 124.0737 |
| Laos / Thailand | 2 | 1 | 14.39, 100.23 to 20.33, 105.33 | 20.3319, 100.2285; 14.3926, 105.3332 |
| Laos / Vietnam | 3 | 2 | 18.02, 102.49 to 22.00, 105.58 | 22.0049, 102.4887; 20.0087, 104.9195; 18.0173, 105.5827 |
| Malaysia / Thailand | 1 | 0 | 5.99, 101.95 to 5.99, 101.95 | 5.9868, 101.9457 |
| Myanmar / Thailand | 1 | 0 | 19.77, 98.93 to 19.77, 98.93 | 19.7726, 98.9339 |
