# Design round 2 – every page

Brief: [docs/DESIGN_BRIEF.md](../../DESIGN_BRIEF.md). Direction a (the system serif) from [round 1](../round-1/README.md) rolled out to every server-rendered page. The map (`/map`, the base map style and the small maps' look) is round 3 and unchanged here.

## Screenshots

Taken with `cd web && npm run screenshots -- round-2` against `cmd/webpreview`. The state before this round is in [before/](before/).

| Page | 400 px | 1400 px |
|---|---|---|
| Front page | [home-400](home-400.jpg) | [home-1400](home-1400.jpg) |
| Latest surveys | [latest-400](latest-400.jpg) | [latest-1400](latest-1400.jpg) |
| Plot | [plot-400](plot-400.jpg) | [plot-1400](plot-1400.jpg) |
| Blank plot | [plot-blank-400](plot-blank-400.jpg) | [plot-blank-1400](plot-blank-1400.jpg) |
| Photo | [photo-400](photo-400.jpg) | [photo-1400](photo-1400.jpg) |
| Surveyor | [surveyor-400](surveyor-400.jpg) | [surveyor-1400](surveyor-1400.jpg) |
| Surveyors | [surveyors-400](surveyors-400.jpg) | [surveyors-1400](surveyors-1400.jpg) |
| Region | [region-400](region-400.jpg) | [region-1400](region-1400.jpg) |
| Regions | [regions-400](regions-400.jpg) | [regions-1400](regions-1400.jpg) |
| About | [about-400](about-400.jpg) | [about-1400](about-1400.jpg) |
| About the plots | [about-plots-400](about-plots-400.jpg) | [about-plots-1400](about-plots-1400.jpg) |
| Principles | [principles-400](principles-400.jpg) | [principles-1400](principles-1400.jpg) |
| Data & licence | [licence-400](licence-400.jpg) | [licence-1400](licence-1400.jpg) |
| Sign in | [signin-400](signin-400.jpg) | [signin-1400](signin-1400.jpg) |
| Check your e-mail | [check-email-400](check-email-400.jpg) | [check-email-1400](check-email-1400.jpg) |
| Choose your name (with an error) | [choose-name-400](choose-name-400.jpg) | [choose-name-1400](choose-name-1400.jpg) |
| Edit profile | [settings-400](settings-400.jpg) | [settings-1400](settings-1400.jpg) |
| Not found | [not-found-400](not-found-400.jpg) | [not-found-1400](not-found-1400.jpg) |

Tall pages at 1400 px are rendered at a device scale below 1 so that they fit 300 KB; the layout is the 1400 px layout.

## What was decided, and why

- **One page head.** Every page starts the same way: on plot, Surveyor, region and error pages a small-caps kicker that says what the page is ("Plot", "Surveyor", "Region", "Error 404"), then the title at display size, a meta line, and the page's figures. Notes and ledes in a head keep the 38 rem measure. Visitors always know where they are (principle 2) without a breadcrumb.
- **Sections like the front page.** Every section below a head starts with the 2 px ink rule and a large head; the note or the "Open on the map →" link sits on the head's line. Before, inner pages had plain `h2`s with equal gaps.
- **Plot names as headings.** A plot name is long. In a heading the dot stays with the word before it and the province with its number is one inline block, so the name breaks as "Pattaya, Bang Lamung ·" / "Chon Buri 2417" instead of starting a line with the dot (`heading` in `funcs.go`, built from `PlotRef.Name()`). On phones the plot title is set at the section-head size so that the place and the province each keep a line. Thai names in a title are set at 60 % in muted ink, like the wordmark's suffix.
- **Figures.** The Surveyor's and the region's counters are the front page's statement line at the section-head size: plots claimed, photos, regions; coverage (with "12 of 5,921 plots"), photos, Surveyors. The region's coverage bar runs under them.
- **Captions leave out what a grid shares.** On a plot page every photo shows the same plot, so the caption is the Surveyor, then the time and the "first" tag. On a Surveyor's page it is the plot, then the time. Everywhere else the two-line caption from round 1 stays.
- **Surveyors list.** The same caption type as the grids: the name in ink, then photos and plots claimed, then "joined …" in muted ink (three short lines; these are people, not photos).
- **Most active Surveyors.** One section with two columns, "Last 30 days" and "All time" as small-caps sub-heads; each row has the rank in muted figures, the name without underline and the counts right-aligned.
- **Regions list.** The coverage column has a fixed width, so every row's bar ends at the same place (before, each bar ended where its own figures began).
- **Photo page.** The photo is as large as the screen allows and centred, without the grey letterbox. The caption is two columns on wide screens: the plot name and the byline on the left, dates, licence and "Full size" in muted ink on the right.
- **Forms and sign-in.** One narrow column: title, the intro as a lede, a hairline, then the fields. Labels in small caps, square fields on a lighter paper (`--paper-light`, now a token), a green filled submit with square corners: the page's one action. Errors carry a red rule on the left and turn the field's border and focus ring red; "Saved." carries a green one.
- **Error page.** "Error 404" as the kicker, the title, the message as a lede, and three ways on: front page, latest surveys, map.
- **Static pages** were already in direction a after round 1; only the table cells now use the spacing tokens.

## Weight

Front page from the fixture: 24.2 KB HTML + 19.0 KB CSS, against a budget of 100 KB.

## Open points for Adrian

1. **Kickers.** "Plot", "Surveyor", "Region" above the titles. They help on the Surveyor page (a bare name otherwise) and are redundant on the others. Keep all, keep only the Surveyor's, or drop them?
2. **Region kicker** says "Region", not "Province": regions are provinces today, but the page does not know a region's kind. Say if it should read "Province".
3. **Small maps** are still the round-1 look (big hexagons on paper-deep, an empty box for the region's coverage until the map loads). They are round 3.
4. **Surveyors list** captions have three lines. Two would mean cutting "joined …" or the counts; say which matters more.
