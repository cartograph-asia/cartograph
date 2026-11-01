# Gamification – design reference

Status: design, not yet for implementation. Agents building other parts (Home, accounts, data model) read this so that what they build does not block it. Numbers marked *(proposal)* are starting values to be tuned with real data; the structure is settled.

## The core loop

1. A Surveyor looks at the map and sees blank plots near them.
2. They go there, take a photo, upload it. The plot turns surveyed.
3. They see the change on the map, their name on the plot, their progress in the region.
4. The next blank plot is a little further away.

Everything below exists to make step 4 attractive. Nothing may make step 2 (going out and taking a real photo) replaceable by sitting at home.

## Actions and points

| Action | Points *(proposal)* | Notes |
|---|---|---|
| First survey of a blank plot | 10 | The core reward. Permanent "first surveyor" credit on the plot. |
| Additional photo in a surveyed plot | 2 | Capped at 3 scoring photos per Surveyor per plot; more are welcome but score nothing. |
| First survey in the other season | 5 | Dry/wet, defined per region by calendar months. Once per plot per Surveyor. |
| Remoteness bonus | ×1.5 / ×2 / ×3 | Multiplier on the first-survey points, by distance of the plot from the nearest road or settlement (see Remoteness). |
| Completing a territory | 50 | For the Surveyor whose photo brought a territory to 100 %. |

Points are never deducted. Removed photos (moderation) remove their points.

## Remoteness *(proposal)*

Computed once per plot from OSM: distance from the plot centre to the nearest road of class tertiary or above and to the nearest named place.
- Tier 0: within 1 km of both → ×1
- Tier 1: more than 1 km from a road → ×1.5
- Tier 2: more than 3 km from a road → ×2
- Tier 3: more than 10 km from any road, or island without road access → ×3

Shown on the map as a subtle marker so Surveyors can plan. Recomputed when the OSM extract is rebuilt.

## Ranks

Global, by lifetime points, same thresholds everywhere *(proposal)*:

| Rank | Points | Roughly |
|---|---|---|
| Apprentice | 0 | joined |
| Surveyor | 50 | five first surveys |
| Senior Surveyor | 500 | a region of effort |
| Master Surveyor | 2 500 | years, or an expedition habit |
| Cartographer | 10 000 | shaped the map |

Rank shows next to the name everywhere. Rank thresholds never move downward; raising them later only affects new promotions.

## Leaderboards

- Per **province** and per **country**: last 30 days and all time. Never a single Asia-wide list on the front page.
- Ranked by points earned *in that area*, so a Master Surveyor from Japan starts at zero in Laos.
- Shown on the region page and in a short form on Home ("most active in Chonburi this month").

## Titles

- **Surveyor of {country}**: awarded on reaching 500 points in that country *(proposal)*. Collected titles show on the profile; a count of countries is a sortable stat.
- **Territory badges**: one per completed territory, with date and the Surveyor who finished it.

## Territories

Closable areas with a coverage percentage: provinces (from OSM admin boundaries), named islands, and cities (admin level 6–8 where it exists). A territory is complete at 100 % of its land plots. Coverage counts only plots whose centre is on land. Small territories (under 20 plots) are not territories; they fold into their parent.

## Seasons

Per region a wet and a dry season defined by months (Thailand: wet May–October). A plot's season status is "dry", "wet" or "both". Seasonal completeness of a territory is a second, later statistic.

## Mode two: "Where was this?" (later)

Opens per region once it holds 500+ photos. A photo is shown without location; the player places a pin; score by distance (full points within 2 km, zero beyond 300 km, logarithmic in between). Photographers earn when their photos are guessed well (recognisable landmarks) and when they are guessed badly (hidden places) – both are achievements, listed separately. Same account, same rank ladder; mode-two points count toward rank at half weight so that going out stays the main road.

## What counts as a valid survey

A photo scores only if all of the following hold:
- Taken inside the plot (EXIF GPS, or manual pin confirmed by the Surveyor; a manual pin more than 2 km from the device location at upload time is flagged).
- Taken by the uploader. Claimed in the upload form; suspected copies (perceptual hash match with an existing photo or a known web image) are held for review.
- Shows the place. Not a screen, not a document, not a selfie filling the frame, not the inside of a vehicle. Automated check later; human review first.
- Not a duplicate of the same Surveyor's photo in the same plot (perceptual hash within threshold).
- Taken in the last 10 years. Older photos are archival: welcome, shown with their date, but they do not claim a plot or score.

## Anti-abuse

- Rate limit: 60 scoring uploads per Surveyor per day *(proposal)*; beyond that, photos are accepted but do not score.
- Drive-by bursts: more than 8 first surveys within 15 minutes along a road trigger review of that batch.
- Review queue with a single rule: when in doubt, the photo stays visible and the points are held.
- No points for anything a script could do.

## What the data model must allow (for agents building now)

- A `points` ledger table: one row per scoring event (surveyor, plot, photo, action, points, multiplier, created_at). Ranks and leaderboards are derived from it; never store a mutable score on the Surveyor row.
- `plots` keeps `first_surveyor` and `claimed_at` forever; a remoteness tier column can be added later.
- `photos` keep `taken_at` and will get a perceptual hash column.
- Regions are a hierarchy (country → province → city) with a type, so territories of any level can be queried the same way.

## Open decisions

- Exact point values and rank thresholds (tune after three months of real uploads).
- Whether archival photos (older than 10 years) get their own small reward.
- Whether mode-two points count at half weight or not at all toward rank.
