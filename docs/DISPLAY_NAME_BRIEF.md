# Brief: Display name – a name to show beside the Surveyor name

Read `CLAUDE.md`, `docs/PRINCIPLES.md`, the Home, Design and Admin entries in `docs/decisions.md` and `docs/ADMIN_BRIEF.md` first. This brief adds an optional **display name** to every Surveyor: "Adrian Altner" beside the Surveyor name `adrian`. It changes how names are shown, not how Surveyors are identified.

## Why

The Surveyor name is a handle: 3–30 characters of A–Z, digits and hyphens, chosen once, unique regardless of case (`views.NamePattern`, decisions.md, Home). It is the profile URL (`/surveyor/{name}`), the feed URL, and it is copied into `photos.surveyor` and `plots.first_surveyor`. Those rules stay. People want to appear with their real name, with spaces, umlauts or in Thai, and to change it later; the handle cannot do that.

## Decisions (Adrian, 2026-11-01)

1. **Shown everywhere instead of the handle; the handle stands beside it on the profile and the photo page.** Grids, lists, "most active", feeds, the masthead and the `/map` panel show the display name. The profile and the photo page show both ("Adrian Altner" and "@adrian" in muted ink), so that nobody can pass as someone else by display name alone.
2. **Optional.** Without one, the handle is shown, as today. The name form at first sign-in gets a second, optional field; existing Surveyors set it in `/settings`.
3. **Changeable at any time** in `/settings`, like the bio. Admins can clear it, like the bio (`surveyor.clear_display_name` in `admin_actions`).
4. **The licence line uses the current display name:** "© Adrian Altner (@adrian), CC BY-SA 4.0", linked to the profile. The handle in it is the permanent identifier; the display name follows changes.

## Vocabulary

- **Surveyor name** stays the word for the handle in code, docs and the forms' labels ("Surveyor name", "chosen once").
- **Display name** is the new field ("Display name", hint: "How your name is shown. Optional; you can change it any time.").
- In CLAUDE.md, the Vocabulary gets: "**Display name** – a Surveyor's optional, changeable name for display (`surveyors.display_name`); shown instead of the Surveyor name wherever names are shown, beside it on the profile and photo page. URLs, feeds' links and the database's copies always use the Surveyor name." The Accounts line gets "one Surveyor name chosen once, an optional display name changed at will".

## Rules for a display name

Validated in `internal/auth/validate.go` (like the bio), stored as given after normalising:

- Unicode NFC (`golang.org/x/text/unicode/norm`; `x/text` is already in `go.mod` as an indirect dependency, so it becomes a direct one, not a new one), leading and trailing space trimmed, runs of white space collapsed to one space.
- 1 to 50 characters after that (`views.DisplayNameMax`); empty means "none".
- Allowed: letters and marks of any script (`\p{L}`, `\p{M}`, so Thai with its vowel and tone marks works), digits, space, and `.` `'` `’` `-`. At least one letter. Nothing else: no control or format characters (this excludes bidi overrides and zero-width characters), no emoji, no `@`, no `/`.
- It must not equal another Surveyor's **handle**, ignoring case ("adrian" as someone else's display name is refused; your own handle is fine). Display names need not be unique among themselves: two Surveyors may both be "Somchai"; the handle beside it on profile and photo page tells them apart.
- Error messages in the form, like the bio's: "A display name has 1 to 50 characters: letters in any script, digits, spaces and . ' -", and "That is another Surveyor's name."

## Data

Migration `0090_display_name.sql` (the next tens block after 0080; the comments work package moves on to 0100): `alter table surveyors add column display_name text check (display_name is null or char_length(display_name) between 1 and 50)`. Nothing else in the schema changes; no existing row is rewritten, and every Surveyor starts without a display name.

`photos.surveyor` and `plots.first_surveyor` keep copying the handle. The display name is always read live from `surveyors`, so a change shows everywhere at once; photos without an account (prototype free-text names) have no display name and show their free-text name as today.

## Code

- `views.SurveyorRef` gets `Display string` and a method `Label() string` (display name, else `Name`). Templates use `.Label` for text and `.Name` for every URL, `title` and the "@handle". `views.Viewer` gets `Display` too, so the masthead shows it.
- `internal/db/views.go`: `photoCols`, the activity lists, new Surveyors, the Surveyors list, the profile and the plot's first surveyor select `s.display_name` next to the name. Grouping and sorting ("most active", ties by name) keep using the handle.
- `internal/web/feed.go`: the item author (`dc:creator`) is the label.
- JSON API: `first_surveyor` and `surveyor` stay the handle (the map links with them); new fields `first_surveyor_label` and `surveyor_label` carry the label. `web/src/panel.ts` shows the label and links with the handle. `/api/me` adds `"label"`.
- Profile (`surveyor.html`): `<h1>` is the label; under it "@adrian" in muted ink when a display name is set. `<title>` and `og:title` use the label; the canonical URL stays `/surveyor/{name}`.
- Photo page: "by Adrian Altner @adrian" and the licence line "© Adrian Altner (@adrian)". Alt texts ("Photo at … by …") use the label.
- Forms: `/signin/name` gets the optional display name field below the Surveyor name; `/settings` gets it above the bio. `internal/auth` stores it with the name (`ChooseName`) and in `UpdateProfile`.
- Admin: the Surveyors list and page show both; the Surveyor page gets "Clear the display name" with a confirm page and reason, like "Clear the bio" (`surveyor.clear_display_name`, old value in the audit row; migration 0090 extends the `admin_actions.action` check).

## Look

No new components: the label takes the place of the name in every existing style; "@handle" uses the existing muted meta style (`.note`). Grid captions keep their two-line rule; a long display name is cut with the ellipsis like a long plot name (`.lt`). Thai display names render in the system serif like Thai place names. No inline styles, no one-off CSS (DESIGN_BRIEF.md); screenshots of profile, photo page, a grid, `/surveyors`, `/settings` and the name form at 400 and 1400 px, with a Latin, a long and a Thai display name, in `docs/design/display-name/`.

## Tests

- **auth:** the rules above, one case each (NFC, collapsing, length, Thai with marks, refused characters including a bidi override and a zero-width joiner, someone else's handle refused in any case, your own allowed, empty clears); the name form with and without a display name; settings sets, changes and clears it.
- **db:** every `views.Source` method returns the display name where a Surveyor is named, and the handle where none is set; feeds and the API carry the label fields; grouping by handle is unchanged.
- **web:** vocabulary and page tests; every template that shows a Surveyor uses `.Label` in text and `.Name` in URLs (a test renders a fixture Surveyor whose display name contains `"` `<` and a space, and checks that no URL contains the display name and the text is escaped); the profile and photo page show "@handle" only when a display name is set; front page weight within budget.
- **admin:** clearing writes the audit row with the old value; the guard covers the new routes.

## Done means

From a clean `docker compose up` with the Pattaya fixtures: setting "Noi Pattaya" in `/settings` changes her name in the masthead, on the front page grid, `/surveyors`, her profile (with "@noi-pattaya" under it), her photos' pages and licence lines, the RSS feed and the `/map` panel, while every link still goes to `/surveyor/noi-pattaya`; a Thai display name ("น้อย พัทยา") renders correctly in all of these; trying "somchai-k" as her display name is refused; clearing it brings the handle back everywhere; an admin can clear it and the log shows the old value. `go test ./...` green with `TEST_DATABASE_URL` set; `/security-review` run.

## Open questions

None at the time of writing. Questions that come up during implementation go into the PR description.
