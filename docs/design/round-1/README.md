# Design round 1 – the front page

Brief: [docs/DESIGN_BRIEF.md](../../DESIGN_BRIEF.md). This round changes the stylesheet and the front page's markup; the other pages pick up the new tokens but are not designed yet (round 2).

## Screenshots

Regenerate with `cd web && npm run screenshots -- round-1` (runs `cmd/webpreview`, which crops the photos in `testdata/` into the fixture photos).

| Direction | 400 px | 1400 px |
|---|---|---|
| a – system serif | [home-a-400.jpg](home-a-400.jpg) | [home-a-1400.jpg](home-a-1400.jpg) |
| b1 – Fraunces for display | [home-b1-400.jpg](home-b1-400.jpg) | [home-b1-1400.jpg](home-b1-1400.jpg) |
| b2 – Inter Tight for display | [home-b2-400.jpg](home-b2-400.jpg) | [home-b2-1400.jpg](home-b2-1400.jpg) |
| before this round | [before/home-a-400.jpg](before/home-a-400.jpg) | [before/home-a-1400.jpg](before/home-a-1400.jpg) |

The 1400 px shots are rendered at a device scale of 0.8 (1120 px wide images) so that a full page fits the 300 KB limit; the layout is the 1400 px layout. In the preview, add `?type=a|b1|b2` to any page to switch.

## What was decided, and why

- **Tokens.** `:root` holds colour roles, a type scale (display, head, title, lede, body, meta, small) and a spacing scale (0.25 / 0.5 / 1 / 2 / 4 / 6 rem). The rule is large gaps between sections, small ones inside them. Every value in `style.css` comes from them; a test fails on any `style=` in a template.
- **Colour roles.** Ink (slate) for text and for links in running text, underlined in a lighter ink that darkens on hover. Green only for actions (Start surveying, form submits) and data (coverage bars, plot marks, the first-survey tag). Plot and Surveyor names in grids are ink without underline, underlined on hover and focus. Muted ink only for meta.
- **Masthead.** Logo (56 px) and wordmark set as one mark, the claim as a quiet line under the wordmark, a hairline below. "Start surveying →" is green text, not a filled block; "Sign in", the Surveyor name and "Sign out" are plain text.
- **Lead block.** The intro moved from the bottom to the top and shares a block with the counters: four large numbers with small labels, in one row on desktop and two by two on phones. The counters' ragged wrap on phones is gone.
- **Sections.** Each starts with a 2 px ink rule and a large head; "More surveys →" and "All regions →" sit on the head's line.
- **Photo grid.** Four tiles per row at full width, three from 40 rem, two on phones; 4:3 crops of the thumb variant. Captions are two lines at meta size: the plot name, then Surveyor · time with a small green "first" tag.
- **Plot of the week.** The medium photo at about 70 % of the width, the caption in the side column (kicker, plot name at title size, text, byline). It stacks on phones.
- **Regions.** One row per region: name and Thai name, the coverage bar across the row, percentage and counts on the right, most active Surveyors under it. Bars are SVG with the length as an attribute instead of an inline style; a single surveyed plot stays visible.
- **New Surveyors.** Square crops of their first plots, up to eight in a row, the name and "joined …" under each.
- **Footer.** The wordmark on the left; links, contact and licence on the right, under a 2 px rule.
- **Typography.** a: the system serif (Charter and fallbacks) for everything, 0 font bytes. b1: Fraunces 600 (18 KB) for the display level only: wordmark, counters' numbers, page and section heads, plot names in the featured block and region rows. b2: Inter Tight 600 (22.6 KB) in the same places. Body text stays the system serif in all three. Fonts are Latin subsets with `font-display: swap`; Thai falls back per glyph. They are only downloaded in their direction.

## Weight

Front page from the fixture: 22.0 KB HTML + 17.4 KB CSS. That is 39 KB in direction a, 57 KB with Fraunces and 62 KB with Inter Tight, against a budget of 100 KB. A test enforces 100 KB including the larger font, and 40 KB for the font.

## Open points for Adrian

1. **Direction:** a, b1 or b2.
2. **Intro at the top.** The brief keeps the content order; the intro was not part of the Home brief's order, so I moved it to the top as the lead. Say if it should go back to the bottom.
3. **Long plot names on phones.** "Pattaya, Bang Lamung · Chon Buri 2417" is cut with an ellipsis on line 1 of a phone caption, so the number is often not visible there. The full name stays in the link text for screen readers and on the plot page. The alternative is to let the name wrap to a second line, which makes captions three lines.
4. **Featured photo.** It uses `loading="lazy"` like before. On desktop it starts below the first screen, so this is fine there.
