# Brief: Admin – editorial work, moderation and invitations for alpha

Read `CLAUDE.md`, `docs/PRINCIPLES.md` and the Home, Security and Design entries in `docs/decisions.md` first. This brief adds an admin area under `/admin`: the pages and the few schema changes that let Adrian do the editorial and moderation work of the alpha in the browser instead of in `psql`. It also closes sign-up: during alpha a new account needs an invitation from an admin (Adrian, 2026-11-01). Apart from that it adds no public feature.

The alpha has real Surveyors and real photos, and today sign-up is open to any e-mail address that receives a magic link (decisions.md: "an unknown address becomes an account only when a name is chosen"). What is missing today:

- **Done in SQL by hand:** setting the plot of the week (`featured`, migration 0009; the seed writes the only row there is); claiming a placeholder account by setting its e-mail (decisions.md, Home); ending a session (`delete from sessions`); reading who signed in when.
- **Recorded but without effect:** `place_overrides` (migration 0075) holds a decision per odd place name, but nothing reads it.
- **Not possible at all:** taking a photo off the site. There is no hidden state. Deleting a `photos` row by hand leaves `plots.first_photo` pointing at nothing (it has no foreign key, `migrations/0001_init.sql`), leaves `photo_count`, `first_surveyor` and the region's `claimed_cells` wrong, silently drops a `featured` row (`on delete cascade`), and leaves the three image objects public, because `internal/storage.Store` has only `Put` and `URL`. There is also no way to stop a Surveyor from signing in or uploading.

## What this brief changes in CLAUDE.md

The "no roles, no admin UI" line was written for the prototype. For alpha it becomes (lead replaces it when this brief is merged):

> - **Accounts:** e-mail + magic link, no passwords, one Surveyor name chosen once. No OAuth. While `SIGNUP=invite` (the default) a new account needs an invitation from an admin; `SIGNUP=open` lets any address join. One role besides Surveyor: **admin**, given to the accounts whose e-mail is listed in `ADMIN_EMAILS`. Admins work in the admin area under `/admin` (`docs/ADMIN_BRIEF.md`): invitations, plot of the week, place-name overrides, hiding photos, blocking Surveyors. Every admin action is written to `admin_actions`. SQL by hand is for what the admin area does not cover and gets a line in `docs/decisions.md`.

Plus three smaller edits: the repository layout gets `internal/admin/  admin pages and actions`; the data model paragraph names `admin_actions`, `invitations` and hidden photos; the vocabulary line becomes "**Claimed / surveyed** – a plot with at least one visible photo", and "**First surveyor**" gets the addition "it changes only when an admin hides the claiming photo, and back when it is unhidden (see Moderation)" (decision 1).

## Who is an admin

**Decision: an environment variable, `ADMIN_EMAILS`**, comma-separated, trimmed and lower-cased at start-up; an entry that is not an address stops the api with an error (like `MAIL_FROM` in `internal/auth/auth.go`). Empty means nobody is an admin and `/admin` is a 404 for everyone. The api logs the number of admins at start, not the addresses.

Why not a column (`surveyors.admin`) or a roles table:

- *Configuration only via environment variables* (CLAUDE.md): who may moderate is deployment configuration, like `SMTP_USER`. Whoever can edit `.env` on the host is the admin anyway.
- No path inside the application can grant the role: no UI for it, no SQL that a bug could run, nothing to escalate.
- The magic link already proves control of the address; the session carries the e-mail (`views.Viewer.Email`, `sessions.email` in `migrations/0006_auth.sql`), so the check is a set lookup in the middleware, no query.
- Cost: adding or removing an admin is an `.env` edit and a restart. For one to three admins that is fine.

An admin must have chosen a Surveyor name: `admin_actions.admin_id` references `surveyors(id)`. `internal/auth` sets `Viewer.Admin bool` in `Middleware` when the session's e-mail is in the set; nothing else decides it.

**Recent sign-in.** Sessions slide for 30 days (decisions.md). Admin pages additionally require that the session's magic link was redeemed within `ADMIN_SESSION_MAX_AGE` (default 24 h, from `sessions.created_at`); an older admin session sees a page "Sign in again to use the admin area" that links to `/signin?next=/admin`. A forgotten laptop with a month-old session then reads the public site, not the admin area. (Decision 3.)

## The jobs

Ranked. **Alpha** is this brief; **later** is listed so the schema does not block it, and is not built.

### Alpha

1. **Moderation: hide and unhide a photo.** The one job that is impossible today and urgent the day a stranger uploads a screenshot, a face or someone else's picture.
2. **Invitations.** Sign-up by invitation only; an admin invites by e-mail address.
3. **Plot of the week.** Today an `insert` in SQL.
4. **Overview.** What happened since the last visit.
5. **Surveyors: block, unblock, end sessions, clear bio.** The counterpart to job 1 for an account that keeps uploading.
6. **Place-name overrides.** Edit `place_overrides` and make new claims honour `replace`.

### Later (not built)

- Removing the image files of a hidden photo (decision 2: the next small package), invitations sent by Surveyors, account deletion or anonymisation on request (needs a privacy page first), renaming a Surveyor (decision 5), claiming a placeholder from the UI (only if prototype data moves to production; SQL until then, as decided), a "report this photo" link for visitors, the review queue and perceptual-hash checks of `docs/GAMIFICATION.md`, points corrections (nothing writes `points` yet), moving a photo to another plot, editing static pages, running imports from the browser.

Out entirely: OAuth, more roles, a JSON admin API, charts and server monitoring, bulk editing.

## Moderation

**Decision: hide, never hard-delete, in alpha.** "Data is kept" (CLAUDE.md) and the CC BY-SA licence both argue for keeping the record; a hidden photo can be restored with one click, a deleted one cannot.

Schema (migration 0080): `photos.hidden_at timestamptz` and `photos.hidden_reason text` (required when hiding, ≤ 200 characters, shown only in the admin area). A SQL view `visible_photos` (`select * from photos where hidden_at is null`).

**A hidden photo is gone from every public read:** pages, feeds, the counters, `GET /api/plots`, `GET /api/plots/{cell}`, the map layer, coverage. Every public query in `internal/db` (`views.go`, `queries.go`, `regions.go`: about 17 places that read `photos` today) reads `visible_photos`; only admin queries and the hide/unhide code read `photos`. A test fails when a public read in `internal/db` names `photos` directly. `/photo/{id}` of a hidden photo answers **410** with the error page ("This photo was removed."), noindex.

**What hiding does to the plot.** `plots.first_photo`, `first_surveyor`, `first_surveyor_id`, `claimed_at` and `photo_count` are derived from the photos (`internal/db/queries.go`, `ClaimPhoto`). Rule: **the claim belongs to the earliest visible photo of the plot**, by `(uploaded_at, id)`, as on upload. Hiding and unhiding recompute these five columns for that plot in the same transaction:

- Hiding a non-claiming photo: `photo_count` − 1, nothing else.
- Hiding the claiming photo of a plot with other visible photos: the claim passes to the next visible photo, its Surveyor becomes first surveyor, its `uploaded_at` becomes `claimed_at`. The old values go into the audit row. (Decision 1.)
- Hiding the only visible photo: the plot is blank again: `photo_count` 0, the four `first_*`/`claimed_at` columns null. The row stays (`photos.cell` references it), and so do its number (`plot_numbers`), place name and region.
- Unhiding recomputes by the same rule, so an unhidden older photo takes the claim back. Hide and unhide are exact inverses.
- The region's `claimed_cells` goes −1/+1 when the plot turns blank/surveyed, `last_surveyed_at` is recomputed for that region from visible photos.

**Bug to fix with this:** `ClaimPhoto` only sets the `first_*` columns when the plot row is new (`on conflict (cell) do update set photo_count = plots.photo_count + 1`). After a plot was blanked by hiding, the next upload would not claim it. The upsert must claim when the existing row has `photo_count = 0`, and report `claimed` accordingly.

**Side effects the confirm page states before the admin confirms:** the plot becomes blank or passes to *name*; the photo is the current plot of the week (the front page then falls back by itself, `internal/db/views.go`, `featured()` already requires the photo to be found); the image files stay reachable at their URLs (Caddy serves `/img/photos/*` with `max-age=31536000, immutable`, `deploy/Caddyfile`) until the files can be removed (decision 2).

Surveyors are not notified automatically; in alpha Adrian writes to them personally (decision 4).

## Surveyors

Schema (migration 0080): `surveyors.blocked_at timestamptz`, `surveyors.blocked_reason text`.

- **Block:** deletes the Surveyor's sessions; `Store.Session` and `CreateSession` (`internal/auth/store.go`) treat a blocked account as having none, so a magic link to that address ends on an error page "This account is blocked. Write to {address}." with the address of `MAIL_FROM` (sign-in mails are still sent; the throttle stays as it is). Uploads need a session, so they stop too. The profile and the photos stay public unless hidden. The confirm page offers "also hide all *n* photos", which writes one audit row per photo with the same reason.
- **Unblock:** clears `blocked_at`; photos hidden with the block stay hidden until unhidden one by one or with "unhide all photos hidden by this block".
- **End sessions:** deletes the Surveyor's sessions without blocking (lost phone).
- **Clear bio:** sets `bio` to `''`, the old text goes into the audit row. Editing a bio is the Surveyor's job; the admin only removes.
- **No rename.** "One Surveyor name chosen once" stays; the name is the URL and is copied into `photos.surveyor` and `plots.first_surveyor` (migration 0051's triggers fill ids from it). An offensive name is a block (decision 5).

## Invitations

**Decision (Adrian): during alpha a new account needs an invitation.** Sign-up is open in code today; an account comes into being when a signed-in address chooses a name (`ChooseName`, `internal/auth/store.go`). The gate sits there and, so that strangers get an answer, at `POST /signin`.

**Switch.** `SIGNUP` in `.env`: `invite` (default) or `open`; anything else stops the api at start-up with an error. `open` is today's behaviour, so the beta can open sign-up without a code change. Existing accounts and the addresses in `ADMIN_EMAILS` sign in in both modes.

**Who may create an account while `SIGNUP=invite`:** an address with an open invitation (not revoked, not yet accepted), or an address in `ADMIN_EMAILS`. Nobody else.

**Schema** (migration 0080): `invitations (email text primary key, invited_by uuid not null references surveyors(id), note text not null default '', invited_at timestamptz not null, accepted_at timestamptz, accepted_by uuid references surveyors(id), revoked_at timestamptz)`. One row per address; inviting an address again after a revoke reopens the row (clears `revoked_at`, new `invited_at`). Addresses are stored lower-cased, normalised by `normalizeEmail` like everywhere in `internal/auth`.

**Inviting.** `/admin/invites`: a form (address, optional personal note ≤ 300 characters, plain text) and the list of invitations, newest first, as pending, accepted (with the Surveyor name it became) or revoked. Submitting the form writes the row and sends the **invitation mail**:

- Subject "You're invited to Cartograph", from `MAIL_FROM`, plain text like the sign-in mail (`internal/auth/mail.go`).
- Body: who invited ("{admin's Surveyor name} invited you to survey with Cartograph"), the note if any, one paragraph on what Cartograph is, the sign-in link, "The link works for 7 days. After that, sign in at {BASE_URL}/signin with this address", the signature ("cartograph.asia – a photo for every plot of land").
- The link is an ordinary magic link (`login_tokens`, same confirm page and flow), only with the longer lifetime `INVITE_TTL` (default `168h`) instead of `LinkTTL`. It counts against the address's sign-in throttle like any link.

An address that already has an account cannot be invited (the form says "{address} is already a Surveyor: {name}"). "Resend" on a pending invitation sends a fresh link, within the throttle; "Revoke" sets `revoked_at` and deletes the address's unused login tokens. Each is an admin action with an audit row (`invite.create`, `invite.resend`, `invite.revoke`). When the invited address chooses its name, `ChooseName` sets `accepted_at` and `accepted_by` in the same transaction.

**Sign-in while `SIGNUP=invite`.** `POST /signin` answers every valid address with the same "Check your e-mail" page as today, so the page never tells who has an account or an invitation. What it sends differs:

- An existing account, an open invitation, or an admin address: the sign-in link, as today.
- Any other address: the **invitation-only mail** instead of a link. Subject "Cartograph is invitation-only for now"; body: Cartograph is in an early test with a few invited Surveyors; reply to this e-mail if you would like an invitation; nothing was created for this address. No link in it.
- At most one invitation-only mail per address per 24 h, so the form cannot be used to flood a stranger's inbox. Table `signup_refusals (email_hash bytea primary key, sent_at timestamptz not null)` (migration 0080) stores only the SHA-256 of the normalised address, not the address. The IP limit stays in front of all of this.

`ChooseName` checks the same rule again in the same transaction (a session from a link sent before the switch, or an invitation revoked after the link was redeemed): without an open invitation or an admin address it answers the choose-name page with "Cartograph is invitation-only for now. Ask the person who told you about it for an invitation.", status 403, and creates nothing.

**Pages.** While `SIGNUP=invite` the sign-in page says, above the form, "Cartograph is invitation-only for now. Already a Surveyor, or invited? Enter your address." The same sentence goes wherever the public pages invite people to join (front page lead, `/about`, `/signin`): the implementer lists the places in the PR. With `SIGNUP=open` the pages read as today.

## Plot of the week

On `featured` as it is (migration 0009, decisions.md: the current entry is the latest started, not ended).

- `/admin/featured` lists the current entry, scheduled ones and the last 20 past ones, with photo, text and dates.
- New entry: photo (chosen from the admin photo page with "Feature this photo…", or by pasting a photo URL or id), text (plain, ≤ 500 characters, enforced in the handler), start (default now), end (empty = until replaced). Only visible photos.
- "End now" sets `ends_at`; a scheduled entry that has not started can be deleted (editorial data, not user content). Started entries are never deleted, so the history of who was featured stays.

## Place-name overrides

On `place_overrides` as it is (migration 0075). `/admin/places` lists every row with its action, replacement, note and the number of plots whose stored `place_name` it currently matches; each row has a form for action (`undecided` / `keep` / `replace`), replacement (required for `replace`) and note. New rows are added from the same page (country, name as imported, kind, note).

Naming (`internal/db/naming.go`, used by `locatePlot` in `regions.go`) reads the overrides: `replace` substitutes the replacement for plots claimed from then on; `keep` and `undecided` change nothing. Plots already named keep their names (decisions.md: "names copied into plots on claim are never replaced"; decision 6). The import tests run against the Pattaya fixture (CLAUDE.md).

## Overview

`/admin`, one page, newest first everywhere (principle 2):

- A counters line: Surveyors (new in 7 days), pending invitations, photos (last 24 h, 7 days), plots surveyed, hidden photos, blocked Surveyors, invitation-only mails sent in the last 7 days.
- The 24 newest uploads as a thumb grid, hidden ones included and marked, each linking to its admin photo page.
- The 20 newest sign-ins (from `sessions`): Surveyor or "no name yet", e-mail, signed in, last seen; and the number of magic links sent and redeemed in the last 24 h (`login_tokens`), which shows mail trouble or abuse at a glance.
- The 10 newest admin actions, linking to `/admin/log`.

## Where it lives and how it looks

**Routes** (package `internal/admin`, mounted in `cmd/api/main.go` behind one guard):

```
GET  /admin                              overview
GET  /admin/photos?page=&hidden=1&surveyor=   all photos, chronological, paginated like /latest
GET  /admin/photo/{id}                   photo, plot, Surveyor, position, dates, featured history, its log
GET  /admin/photo/{id}/hide              confirm page; POST does it (field reason)
POST /admin/photo/{id}/unhide
GET  /admin/featured                     list + form; POST creates
POST /admin/featured/{id}/end            POST /admin/featured/{id}/delete (not started only)
GET  /admin/surveyors?page=              GET /admin/surveyor/{name}
GET  /admin/surveyor/{name}/block        confirm page; POST does it (fields reason, hide_photos)
POST /admin/surveyor/{name}/unblock      POST /admin/surveyor/{name}/end-sessions
GET  /admin/surveyor/{name}/clear-bio    confirm page; POST does it
GET  /admin/places                       list + forms; POST /admin/places (new), POST /admin/places/{country}/{name}
GET  /admin/invites                      list + form; POST creates and sends
POST /admin/invites/resend               POST /admin/invites/revoke (field email)
GET  /admin/log?page=                    the audit trail
```

E-mail addresses never appear in URLs (Surveyors are addressed by name; invitations take the address as a POST field).

**Guard.** One middleware on `/admin/*`: anonymous, nameless or non-admin viewers get the ordinary **404** page (the area does not announce itself); an admin with an old session gets the sign-in-again page. Every response carries `Cache-Control: no-store`, `Referrer-Policy: no-referrer` (like the account pages, `privateHeaders` in `internal/auth/auth.go`) and `X-Robots-Tag: noindex`; every page sets `Page.NoIndex`. `/admin` is not named in any `robots.txt`, which would only advertise it.

**Forms.** Every state change is a POST with the CSRF field and `CrossOriginProtection`, checked by the same code as the account forms: `internal/auth` exports its form check (today the unexported `formOK`) as `(*Auth).CheckForm(w, r) bool`, and `internal/admin` gets it injected. A GET never changes anything. Destructive actions (hide, block, clear bio, delete a scheduled entry) go through a confirm page that states the consequences in words and needs a reason; no JavaScript `confirm()`. After a POST the handler redirects back with `?done=…` and the page shows one line saying what happened.

**Audit.** Table `admin_actions` (migration 0080): `id bigint identity`, `at timestamptz`, `admin_id uuid not null references surveyors(id)`, `action text` (`photo.hide`, `photo.unhide`, `featured.create`, `featured.end`, `featured.delete`, `surveyor.block`, `surveyor.unblock`, `surveyor.end_sessions`, `surveyor.clear_bio`, `place_override.create`, `place_override.update`, `invite.create`, `invite.resend`, `invite.revoke`), `target text` (`photo:<uuid>`, `surveyor:<uuid>`, `featured:<id>`, `place_override:th/<name>`, `invite:<address>`), `reason text`, `before jsonb`, `after jsonb`. Written in the same transaction as the change; a trigger refuses `update` and `delete` on the table. Each action also logs one `slog` info line (admin name, action, target), no e-mail addresses.

**Look.** Server-rendered with `html/template` through the existing renderer (`internal/web`), new templates `admin-*.html` on `layout.html`, the public masthead and footer, plus one line of admin navigation under the masthead. Lists reuse the table-like list of `/regions` and the form styles of `/settings`; photo grids use the thumb variant and the existing grid. The page tokens only; no inline styles, no one-off CSS, no framework, works without JavaScript (DESIGN_BRIEF.md). Admin rules go into `style.css` only while the front page stays within its 100 KB budget (`TestFrontPageWeight`); if they would push it over, they move to an `admin.css` on the same tokens, linked from admin pages only. Navigation line: Overview · Photos · Invitations · Plot of the week · Surveyors · Place names · Log. Masthead: for an admin, "Admin" appears as plain text next to "Sign out"; nobody else sees it. The public photo page gets one admin line for admins only ("Admin: details · feature · hide"), so moderation starts from where the photo was seen.

## Safety

- No hard delete of user content (photos, Surveyors, bios beyond clearing) in this brief. The only delete is a scheduled `featured` row that never went live.
- Every action reversible from the UI except "clear bio" (the old text is in the audit row) and "end sessions".
- Every state change: admin guard, recent sign-in, CSRF + same origin, POST only, audit row in the same transaction.
- Admin pages show e-mail addresses; nothing else does, and nothing logs them. Addresses that were refused an account are stored only as hashes.
- While `SIGNUP=invite` no page or status code tells a visitor whether an address has an account or an invitation. Every path sends at most one mail, synchronously as today, and a failed send is the same 503 for both kinds; a throttled invitation-only mail answers without sending, like a throttled link.

## Tests

Following the repo's conventions (database tests skip without `TEST_DATABASE_URL`; template changes run the vocabulary and page tests; naming changes run against the Pattaya fixture):

- **db:** hiding a non-claiming photo, the claiming photo with others left, and the only photo: plot columns, `photo_count`, `claimed_cells`, `last_surveyed_at`; unhide restores the exact previous state; a blanked plot is claimed again by the next upload (the `ClaimPhoto` fix); a hidden photo is absent from every `views.Source` method, every feed and the API queries (`Plots`, `Plot`, coverage); the featured fallback; the audit row is written in the same transaction and cannot be updated or deleted; a test that fails when a public read in `internal/db` names `photos` instead of `visible_photos`; `replace` overrides name new plots, existing plots keep theirs.
- **auth:** `ADMIN_EMAILS` parsing (case, spaces, empty, invalid → error); `SIGNUP` parsing (`invite`, `open`, empty → `invite`, anything else → error); `Viewer.Admin` set only for listed addresses with a name; a blocked account gets no session and sees the blocked page; block deletes sessions. With `SIGNUP=invite`: an existing account, an invited address and an admin address get a link; any other address gets the invitation-only mail, at most once in 24 h, and the same page and status as the others; `ChooseName` without an invitation is 403 and creates nothing; accepting sets `accepted_at`/`accepted_by`; a revoked invitation's unused links stop working; the invitation link lives `INVITE_TTL`, other links `LinkTTL`. With `SIGNUP=open` everything behaves as today.
- **admin handlers** (fake store, like `internal/auth/memstore_test.go`): a table over every registered `/admin` route (walked from the chi router, so a new route cannot skip the guard): anonymous, nameless and non-admin get 404; old admin session gets the sign-in-again page; POST without CSRF or cross-origin gets 403; GET changes nothing; each action writes its audit row; the headers above.
- **web:** every admin template in `TestEveryPageRenders`, `TestNoInlineStyles` and the vocabulary test; `noindex` on every admin page; the masthead "Admin" link and the photo page's admin line appear for an admin only; `/photo/{id}` of a hidden photo is 410; front page weight still within budget.

## Migrations

The highest migration is `0075_place_naming.sql`; this brief takes the next tens block, **0080+** (the comments work package, which also proposed 0080, moves to 0090). Task 1 owns `0080_admin.sql` (photos hidden columns and `visible_photos`, surveyors blocked columns, `admin_actions` and its trigger, `invitations`, `signup_refusals`). Further numbers (0081…) only from the lead. No migration rewrites existing rows; existing Surveyors need no invitation row.

## Configuration

In `deploy/.env.example` under "Accounts", each with a comment, and passed to the api in `deploy/docker-compose.yml` like `BASE_URL`:

- `ADMIN_EMAILS` (default empty),
- `ADMIN_SESSION_MAX_AGE` (default `24h`),
- `SIGNUP` (default `invite`),
- `INVITE_TTL` (default `168h`).

Nothing environment-specific in code. The local `.env` sets `ADMIN_EMAILS` to Adrian's address; the Pattaya fixtures' Surveyors are existing accounts and sign in as before.

## Sub-agent split

- **Task 1 – Data:** migration 0080; `visible_photos` in every public read; hide/unhide with the plot and region recompute; the `ClaimPhoto` fix; queries for the overview, admin photo and Surveyor pages, the log; featured create/end/delete; place overrides read/write and the naming change; block/unblock/end sessions/clear bio in the store; tests above. Publishes its result types first (in `internal/views`, as the Home brief did) so tasks 2 and 3 build against them.
- **Task 2 – Access:** `ADMIN_EMAILS`, `ADMIN_SESSION_MAX_AGE`, `SIGNUP` and `INVITE_TTL` in `auth.ConfigFromEnv`, `Viewer.Admin`, session age in `auth.Session`, blocked accounts in `Store`, the exported form check, the `/admin` guard middleware and the blocked page; invitations in the store (create, resend, revoke, accept in `ChooseName`), the sign-in gate, `signup_refusals`, the invitation and invitation-only mails in `internal/auth/mail.go`; tests.
- **Task 3 – Pages:** `internal/admin` handlers, `admin-*.html` templates (including `/admin/invites`), CSS on the tokens, the masthead link and the photo page's admin line, the 410 page, the invitation-only sentence on the public pages; fixtures in `viewstest` (one hidden photo, one blocked Surveyor, an admin viewer, a pending and an accepted invitation); admin pages in `cmd/webpreview` (admin viewer via `?as=admin`, webpreview only) and an `admin` set in `tools/screenshots/pages.json`.
- **Task 4 – Integration (lead):** wiring in `cmd/api/main.go`, `.env.example` and compose, the full stack with the Pattaya fixtures, walking every job below, screenshots, the CLAUDE.md wording above, `docs/decisions.md`.

Tasks 1–3 run in parallel after task 1's types are in; task 1 owns the only migration.

## Done means

From a clean `docker compose up` with the Pattaya fixtures and `ADMIN_EMAILS=noi-pattaya@example.test` (link from the api log): `/admin` shows the overview with real numbers; hiding the claiming photo of a two-photo plot passes the plot to the other Surveyor everywhere (plot page, grids, `/map` panel, feeds), hiding the last photo makes the plot blank and lowers Chon Buri's coverage, unhiding restores both exactly, and the hidden photo's page is a 410; a new plot of the week appears on the front page and "End now" brings back the fallback; blocking `ploy88` signs them out and their next magic link ends on the blocked page; an invitation to a new address arrives (link in the api log), its link leads to choosing a name and the invitation shows as accepted, while a sign-in from an uninvited address shows the same page and logs the invitation-only mail, once per day, and creates no account; with `SIGNUP=open` the uninvited address can join; a `replace` override names the next plot claimed near that place; every action is in `/admin/log`; signed in as `mali`, every `/admin` URL is a 404. Screenshots of every admin page at 400 and 1400 px from `tools/screenshots`; no inline styles; front page within budget; `go test ./...` green with `TEST_DATABASE_URL` set.

## Decisions (Adrian, 2026-11-01)

1. **The first-surveyor credit passes on** when the claiming photo is hidden: to the earliest visible photo, and back on unhide. A hidden photo is one that should not have counted; the original credit stays in `admin_actions`. CLAUDE.md's vocabulary line gets the addition named above.
2. **Removing the image files of a hidden photo is the next small package, not this one.** Hiding leaves the three objects public at their URLs. The follow-up adds `Delete` to `internal/storage` and a "remove files" action for photos hidden at least 24 h, which deletes the objects and sets `photos.files_removed_at`, keeping the row and the audit. Until then a takedown is done on the host by hand and logged in `docs/decisions.md`.
3. **Recent sign-in for the admin area: yes, 24 h** (`ADMIN_SESSION_MAX_AGE`).
4. **Surveyors are not told automatically** when a photo is hidden; Adrian writes personally. An automatic mail can come with the review queue.
5. **No renaming of Surveyors.** An offensive name is a block. If ever needed, a later brief adds a rename with a redirect from the old name.
6. **`replace` overrides apply to new claims only.** Plots that already carry a name keep it; renaming them (Ban Bang Krabueang, if it is to be replaced) is a separate, logged follow-up, because it changes names Surveyors have already seen and shared.
7. **Sign-up by invitation during alpha**, switchable with `SIGNUP` so the beta can open it without a code change. An admin enters an address and the site sends the invitation mail with a sign-in link; an uninvited address gets the same page as everyone and a short invitation-only mail, at most once a day (section Invitations).

## Open questions

None at the time of writing. Questions that come up during implementation go into the PR description.
