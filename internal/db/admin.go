package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Errors of the admin actions.
var (
	// ErrUnchanged: the action has nothing to do (the photo is hidden
	// already, the Surveyor is not blocked, the entry has ended).
	ErrUnchanged = errors.New("db: nothing to change")
	// ErrInvalid: the input breaks a rule; the message says which, in
	// words for the admin.
	ErrInvalid = errors.New("db: invalid")
)

// invalid wraps ErrInvalid with a message for the admin.
func invalid(msg string) error { return fmt.Errorf("%w: %s", ErrInvalid, msg) }

// InvalidMessage returns the admin-facing message of an ErrInvalid error.
func InvalidMessage(err error) string {
	return strings.TrimPrefix(err.Error(), ErrInvalid.Error()+": ")
}

// Admin is the admin area's side of the database (docs/ADMIN_BRIEF.md). It
// is the only code that reads photos directly instead of visible_photos.
// Every write records an admin_actions row in its own transaction.
type Admin struct{ d *DB }

// Admin returns the admin side of d.
func (d *DB) Admin() *Admin { return &Admin{d: d} }

// adminPhotoFrom is photoFrom on every photo, hidden ones included.
var adminPhotoFrom = strings.Replace(photoFrom, "visible_photos", "photos", 1)

const adminPhotoCols = photoCols + `, ph.hidden_at, coalesce(ph.hidden_reason, ''), ph.files_removed_at`

func scanAdminPhoto(row pgx.CollectableRow) (views.AdminPhoto, error) {
	var p views.AdminPhoto
	var id uuid.UUID
	err := row.Scan(append(photoDest(&p.PhotoItem, &id), &p.HiddenAt, &p.HiddenReason, &p.FilesRemovedAt)...)
	finishPhoto(&p.PhotoItem, id)
	return p, err
}

func (a *Admin) photos(ctx context.Context, where, order string, args ...any) ([]views.AdminPhoto, error) {
	rows, err := a.d.Pool.Query(ctx, "select "+adminPhotoCols+adminPhotoFrom+where+" "+order, args...)
	if err != nil {
		return nil, fmt.Errorf("db: admin photos: %w", err)
	}
	out, err := pgx.CollectRows(rows, scanAdminPhoto)
	if err != nil {
		return nil, fmt.Errorf("db: admin photos: %w", err)
	}
	return out, nil
}

// audit writes one admin_actions row in tx and logs one line without
// e-mail addresses.
func audit(ctx context.Context, tx pgx.Tx, adminID, action, target, reason string, before, after any) error {
	b, err := jsonOrNil(before)
	if err != nil {
		return err
	}
	f, err := jsonOrNil(after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into admin_actions (admin_id, action, target, reason, before, after)
		values ($1::uuid, $2, $3, $4, $5, $6)`, adminID, action, target, reason, b, f)
	if err != nil {
		return fmt.Errorf("audit %s: %w", action, err)
	}
	if strings.HasPrefix(target, "invite:") {
		target = "invite"
	}
	slog.Info("admin action", "action", action, "target", target)
	return nil
}

func jsonOrNil(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// plotClaim is the part of a plot row that follows from its visible photos.
type plotClaim struct {
	FirstPhoto      *uuid.UUID `json:"first_photo"`
	FirstSurveyor   *string    `json:"first_surveyor"`
	FirstSurveyorID *uuid.UUID `json:"first_surveyor_id"`
	ClaimedAt       *time.Time `json:"claimed_at"`
	PhotoCount      int        `json:"photo_count"`
}

// recomputePlot sets the plot's claim and count from its visible photos: the
// claim belongs to the earliest visible photo by (uploaded_at, id), as on
// upload; without one the plot is blank. The region's claimed_cells follows
// when the plot turns blank or surveyed, and its last_surveyed_at is
// recomputed. It returns the claim before and after.
func recomputePlot(ctx context.Context, tx pgx.Tx, cell string) (before, after plotClaim, err error) {
	var region *int
	err = tx.QueryRow(ctx, `
		select first_photo, first_surveyor, first_surveyor_id, claimed_at, photo_count, region_id
		from plots where cell = $1 for update`, cell,
	).Scan(&before.FirstPhoto, &before.FirstSurveyor, &before.FirstSurveyorID, &before.ClaimedAt, &before.PhotoCount, &region)
	if err != nil {
		return before, after, fmt.Errorf("read plot: %w", err)
	}
	if err = tx.QueryRow(ctx, `select count(*) from visible_photos where cell = $1`, cell).Scan(&after.PhotoCount); err != nil {
		return before, after, fmt.Errorf("count photos: %w", err)
	}
	if after.PhotoCount > 0 {
		var id uuid.UUID
		var name string
		var sid *uuid.UUID
		var at time.Time
		err = tx.QueryRow(ctx, `
			select id, surveyor, surveyor_id, uploaded_at from visible_photos
			where cell = $1 order by uploaded_at, id limit 1`, cell).Scan(&id, &name, &sid, &at)
		if err != nil {
			return before, after, fmt.Errorf("first photo: %w", err)
		}
		after.FirstPhoto, after.FirstSurveyor, after.FirstSurveyorID, after.ClaimedAt = &id, &name, sid, &at
	}
	_, err = tx.Exec(ctx, `
		update plots set first_photo = $2, first_surveyor = $3, first_surveyor_id = $4, claimed_at = $5, photo_count = $6
		where cell = $1`, cell, after.FirstPhoto, after.FirstSurveyor, after.FirstSurveyorID, after.ClaimedAt, after.PhotoCount)
	if err != nil {
		return before, after, fmt.Errorf("update plot: %w", err)
	}
	if region != nil {
		delta := 0
		switch {
		case before.PhotoCount > 0 && after.PhotoCount == 0:
			delta = -1
		case before.PhotoCount == 0 && after.PhotoCount > 0:
			delta = 1
		}
		_, err = tx.Exec(ctx, `
			update regions set claimed_cells = claimed_cells + $2,
			  last_surveyed_at = (select max(ph.uploaded_at) from visible_photos ph join plots p on p.cell = ph.cell where p.region_id = $1)
			where id = $1`, *region, delta)
		if err != nil {
			return before, after, fmt.Errorf("update region: %w", err)
		}
	}
	return before, after, nil
}

// hideInTx hides one visible photo and recomputes its plot.
func hideInTx(ctx context.Context, tx pgx.Tx, adminID string, photoID uuid.UUID, reason string, now time.Time) error {
	var cell string
	err := tx.QueryRow(ctx, `
		update photos set hidden_at = $2, hidden_reason = $3
		where id = $1 and hidden_at is null returning cell`, photoID, now, reason).Scan(&cell)
	if errors.Is(err, pgx.ErrNoRows) {
		return photoMissingOr(ctx, tx, photoID)
	}
	if err != nil {
		return err
	}
	before, after, err := recomputePlot(ctx, tx, cell)
	if err != nil {
		return err
	}
	return audit(ctx, tx, adminID, "photo.hide", "photo:"+photoID.String(), reason,
		map[string]any{"plot": cell, "claim": before}, map[string]any{"plot": cell, "claim": after})
}

// unhideInTx unhides one hidden photo and recomputes its plot.
func unhideInTx(ctx context.Context, tx pgx.Tx, adminID string, photoID uuid.UUID) error {
	var cell, reason string
	var hiddenAt time.Time
	err := tx.QueryRow(ctx, `
		update photos p set hidden_at = null, hidden_reason = null
		from photos old where old.id = p.id and p.id = $1 and p.hidden_at is not null and p.files_removed_at is null
		returning p.cell, old.hidden_at, old.hidden_reason`, photoID).Scan(&cell, &hiddenAt, &reason)
	if errors.Is(err, pgx.ErrNoRows) {
		var removed bool
		if err := tx.QueryRow(ctx, `select files_removed_at is not null from photos where id = $1`, photoID).Scan(&removed); err == nil && removed {
			return invalid("Its image files were removed; it cannot be shown again.")
		}
		return photoMissingOr(ctx, tx, photoID)
	}
	if err != nil {
		return err
	}
	before, after, err := recomputePlot(ctx, tx, cell)
	if err != nil {
		return err
	}
	return audit(ctx, tx, adminID, "photo.unhide", "photo:"+photoID.String(), "",
		map[string]any{"plot": cell, "claim": before, "hidden_at": hiddenAt, "hidden_reason": reason},
		map[string]any{"plot": cell, "claim": after})
}

// photoMissingOr is views.ErrNotFound for an unknown photo, else ErrUnchanged.
func photoMissingOr(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from photos where id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return views.ErrNotFound
	}
	return ErrUnchanged
}

func parsePhotoID(id string) (uuid.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil, views.ErrNotFound
	}
	return u, nil
}

// finish maps the sentinel errors of a transaction through and wraps the rest.
func finish(what string, err error) error {
	if err == nil || errors.Is(err, views.ErrNotFound) || errors.Is(err, ErrUnchanged) || errors.Is(err, ErrInvalid) {
		return err
	}
	return fmt.Errorf("db: %s: %w", what, err)
}

// checkReason validates a hide or block reason.
func checkReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	switch {
	case reason == "":
		return "", invalid("Give a reason; only admins see it.")
	case len([]rune(reason)) > views.ReasonMax:
		return "", invalid(fmt.Sprintf("The reason is longer than %d characters.", views.ReasonMax))
	}
	return reason, nil
}

// HidePhoto hides a photo: it is gone from every public read, and its plot's
// claim and counts are recomputed. ErrUnchanged if it is hidden already.
func (a *Admin) HidePhoto(ctx context.Context, adminID, photoID, reason string, now time.Time) error {
	id, err := parsePhotoID(photoID)
	if err != nil {
		return err
	}
	if reason, err = checkReason(reason); err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error { return hideInTx(ctx, tx, adminID, id, reason, now) })
	return finish("hide photo", err)
}

// UnhidePhoto shows a hidden photo again; an older photo takes the plot's
// claim back. ErrUnchanged if it is not hidden.
func (a *Admin) UnhidePhoto(ctx context.Context, adminID, photoID string) error {
	id, err := parsePhotoID(photoID)
	if err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error { return unhideInTx(ctx, tx, adminID, id) })
	return finish("unhide photo", err)
}

// RemovePhotoFiles deletes the image files of a photo hidden for at least
// views.RemoveFilesAfter, through del (the store's Delete), and records it.
// The row, its hidden state and the audit trail stay. The objects are
// deleted inside the transaction, before the commit: if a delete fails,
// nothing is recorded and the action can be repeated (deleting twice is
// fine).
func (a *Admin) RemovePhotoFiles(ctx context.Context, adminID, photoID, reason string, now time.Time,
	del func(ctx context.Context, key string) error) error {
	id, err := parsePhotoID(photoID)
	if err != nil {
		return err
	}
	if reason, err = checkReason(reason); err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		var hiddenAt, removed *time.Time
		var large, thumb string
		var medium *string
		err := tx.QueryRow(ctx, `
			select hidden_at, files_removed_at, key_large, key_medium, key_thumb
			from photos where id = $1 for update`, id).Scan(&hiddenAt, &removed, &large, &medium, &thumb)
		if errors.Is(err, pgx.ErrNoRows) {
			return views.ErrNotFound
		}
		if err != nil {
			return err
		}
		switch {
		case removed != nil:
			return ErrUnchanged
		case hiddenAt == nil:
			return invalid("Hide the photo first.")
		case now.Before(hiddenAt.Add(views.RemoveFilesAfter)):
			return invalid("The files can be removed 24 hours after the photo was hidden.")
		}
		keys := []string{large, thumb}
		if medium != nil && *medium != "" {
			keys = append(keys, *medium)
		}
		for _, k := range keys {
			if err := del(ctx, k); err != nil {
				return fmt.Errorf("delete %s: %w", k, err)
			}
		}
		if _, err := tx.Exec(ctx, `update photos set files_removed_at = $2 where id = $1`, id, now); err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "photo.remove_files", "photo:"+id.String(), reason, nil,
			map[string]any{"keys": keys, "files_removed_at": now})
	})
	return finish("remove photo files", err)
}

// HideEffect says what hiding the photo would do to its plot and the front page.
func (a *Admin) HideEffect(ctx context.Context, photoID string, now time.Time) (*views.HideEffect, error) {
	id, err := parsePhotoID(photoID)
	if err != nil {
		return nil, err
	}
	var e views.HideEffect
	var next *string
	err = a.d.Pool.QueryRow(ctx, `
		select pl.first_photo is not distinct from ph.id,
		  (select coalesce(s.name, o.surveyor) from visible_photos o left join surveyors s on s.id = o.surveyor_id
		   where o.cell = ph.cell and o.id <> ph.id order by o.uploaded_at, o.id limit 1),
		  exists (select 1 from featured f where f.photo_id = ph.id and f.id = (
		    select id from featured where starts_at <= $2 and (ends_at is null or ends_at > $2)
		    order by starts_at desc, id desc limit 1))
		from photos ph join plots pl on pl.cell = ph.cell where ph.id = $1`, id, now,
	).Scan(&e.Claims, &next, &e.Featured)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: hide effect: %w", err)
	}
	e.PassesTo = deref(next)
	return &e, nil
}

// Photos returns one page of every photo, newest first; hiddenOnly and
// surveyor (a name, any case) narrow it.
func (a *Admin) Photos(ctx context.Context, page int, hiddenOnly bool, surveyor string) (*views.AdminPhotos, error) {
	where := `where ($1 = false or ph.hidden_at is not null) and ($2 = '' or lower(coalesce(s.name, ph.surveyor)) = lower($2))`
	var total int
	if err := a.d.Pool.QueryRow(ctx, `select count(*)`+adminPhotoFrom+where, hiddenOnly, surveyor).Scan(&total); err != nil {
		return nil, fmt.Errorf("db: admin photos: %w", err)
	}
	pg, err := pagination(page, views.AdminPerPage, total)
	if err != nil {
		return nil, err
	}
	ps, err := a.photos(ctx, where, newestFirst+" limit $3 offset $4", hiddenOnly, surveyor, pg.PerPage, offset(pg))
	if err != nil {
		return nil, err
	}
	return &views.AdminPhotos{Photos: ps, Pagination: pg, HiddenOnly: hiddenOnly, Surveyor: surveyor}, nil
}

// Photo returns the admin page of one photo, hidden or not.
func (a *Admin) Photo(ctx context.Context, photoID string) (*views.AdminPhotoDetail, error) {
	id, err := parsePhotoID(photoID)
	if err != nil {
		return nil, err
	}
	ps, err := a.photos(ctx, "where ph.id = $1", "", id)
	if err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		return nil, views.ErrNotFound
	}
	out := &views.AdminPhotoDetail{Photo: ps[0]}
	var first *string
	err = a.d.Pool.QueryRow(ctx, `
		select ph.lat, ph.lng, coalesce(s.email, ''), pl.photo_count, coalesce(fs.name, pl.first_surveyor)
		from photos ph join plots pl on pl.cell = ph.cell
		left join surveyors s on s.id = ph.surveyor_id
		left join surveyors fs on fs.id = pl.first_surveyor_id
		where ph.id = $1`, id).Scan(&out.Lat, &out.Lng, &out.Email, &out.PlotPhotos, &first)
	if err != nil {
		return nil, fmt.Errorf("db: admin photo: %w", err)
	}
	if out.PlotPhotos > 0 {
		out.FirstSurveyor = deref(first)
	}
	if out.Featured, err = a.featuredWhere(ctx, "where f.photo_id = $1 order by f.starts_at desc", id); err != nil {
		return nil, err
	}
	if out.Actions, err = a.actions(ctx, "where a.target = $1", "limit 50", "photo:"+id.String()); err != nil {
		return nil, err
	}
	return out, nil
}

// Overview returns /admin.
func (a *Admin) Overview(ctx context.Context, now time.Time) (*views.AdminOverview, error) {
	out := &views.AdminOverview{}
	c := &out.Counters
	day, week := now.Add(-24*time.Hour), now.Add(-7*24*time.Hour)
	err := a.d.Pool.QueryRow(ctx, `select
		(select count(*) from surveyors),
		(select count(*) from surveyors where joined_at > $2),
		(select count(*) from invitations where accepted_at is null and revoked_at is null),
		(select count(*) from photos where uploaded_at > $1),
		(select count(*) from photos where uploaded_at > $2),
		(select count(*) from plots where photo_count > 0),
		(select count(*) from photos where hidden_at is not null),
		(select count(*) from surveyors where blocked_at is not null),
		(select count(*) from signup_refusals where sent_at > $2),
		(select count(*) from login_tokens where created_at > $1),
		(select count(*) from login_tokens where used_at > $1)`, day, week,
	).Scan(&c.Surveyors, &c.NewSurveyors, &c.PendingInvites, &c.Photos24h, &c.Photos7d, &c.Plots, &c.Hidden,
		&c.Blocked, &c.Refusals7d, &out.LinksSent, &out.LinksRedeemed)
	if err != nil {
		return nil, fmt.Errorf("db: admin overview: %w", err)
	}
	if out.Uploads, err = a.photos(ctx, "", newestFirst+" limit $1", views.AdminLatest); err != nil {
		return nil, err
	}
	if out.SignIns, err = a.signIns(ctx, "", views.AdminSignIns); err != nil {
		return nil, err
	}
	if out.Actions, err = a.actions(ctx, "", fmt.Sprintf("limit %d", views.AdminActions)); err != nil {
		return nil, err
	}
	return out, nil
}

// signIns returns the newest sessions, of one Surveyor if surveyorID is set.
func (a *Admin) signIns(ctx context.Context, surveyorID string, limit int) ([]views.AdminSignIn, error) {
	rows, err := a.d.Pool.Query(ctx, `
		select coalesce(u.name, ''), s.email, s.created_at, s.last_seen_at
		from sessions s left join surveyors u on u.id = s.surveyor_id
		where ($1 = '' or s.surveyor_id::text = $1)
		order by s.created_at desc limit $2`, surveyorID, limit)
	if err != nil {
		return nil, fmt.Errorf("db: sign-ins: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (views.AdminSignIn, error) {
		var s views.AdminSignIn
		err := r.Scan(&s.Surveyor, &s.Email, &s.SignedInAt, &s.LastSeen)
		s.SignedInAt, s.LastSeen = s.SignedInAt.UTC(), s.LastSeen.UTC()
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: sign-ins: %w", err)
	}
	return out, nil
}

// actions reads the audit trail with the target's name and admin page.
func (a *Admin) actions(ctx context.Context, where, limit string, args ...any) ([]views.AdminAction, error) {
	rows, err := a.d.Pool.Query(ctx, `
		select a.id, a.at, coalesce(s.name, ''), a.action, a.target,
		  coalesce((select name from surveyors t where a.target = 'surveyor:' || t.id::text), ''),
		  a.reason, coalesce(a.before::text, ''), coalesce(a.after::text, '')
		from admin_actions a left join surveyors s on s.id = a.admin_id `+where+`
		order by a.at desc, a.id desc `+limit, args...)
	if err != nil {
		return nil, fmt.Errorf("db: admin actions: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (views.AdminAction, error) {
		var x views.AdminAction
		var target, surveyor string
		err := r.Scan(&x.ID, &x.At, &x.Admin, &x.Action, &target, &surveyor, &x.Reason, &x.Before, &x.After)
		x.At = x.At.UTC()
		x.Target, x.Link = describeTarget(target, surveyor)
		return x, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: admin actions: %w", err)
	}
	return out, nil
}

// describeTarget turns an admin_actions target into words and a link.
func describeTarget(target, surveyor string) (string, string) {
	kind, id, _ := strings.Cut(target, ":")
	switch kind {
	case "photo":
		short := id
		if len(short) > 8 {
			short = short[:8]
		}
		return "photo " + short, "/admin/photo/" + id
	case "surveyor":
		if surveyor == "" {
			return "a Surveyor", ""
		}
		return surveyor, "/admin/surveyor/" + surveyor
	case "featured":
		return "plot of the week #" + id, "/admin/featured"
	case "place_override":
		return id, "/admin/places"
	case "invite":
		return id, "/admin/invites"
	}
	return target, ""
}

// Log returns one page of the audit trail.
func (a *Admin) Log(ctx context.Context, page int) (*views.AdminLog, error) {
	var total int
	if err := a.d.Pool.QueryRow(ctx, `select count(*) from admin_actions`).Scan(&total); err != nil {
		return nil, fmt.Errorf("db: admin log: %w", err)
	}
	pg, err := pagination(page, views.AdminLogPerPage, total)
	if err != nil {
		return nil, err
	}
	acts, err := a.actions(ctx, "", "limit $1 offset $2", pg.PerPage, offset(pg))
	if err != nil {
		return nil, err
	}
	return &views.AdminLog{Actions: acts, Pagination: pg}, nil
}

// Plot of the week.

func (a *Admin) featuredWhere(ctx context.Context, where string, args ...any) ([]views.AdminFeatured, error) {
	rows, err := a.d.Pool.Query(ctx, `
		select f.id, f.body, f.starts_at, f.ends_at, f.photo_id, `+photoCols+adminPhotoFrom+`
		join featured f on f.photo_id = ph.id `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("db: featured: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (views.AdminFeatured, error) {
		var f views.AdminFeatured
		var fid, id uuid.UUID
		err := r.Scan(append([]any{&f.ID, &f.Text, &f.StartsAt, &f.EndsAt, &fid}, photoDest(&f.Photo, &id)...)...)
		finishPhoto(&f.Photo, id)
		f.StartsAt = f.StartsAt.UTC()
		if f.EndsAt != nil {
			t := f.EndsAt.UTC()
			f.EndsAt = &t
		}
		return f, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: featured: %w", err)
	}
	return out, nil
}

// Featured returns /admin/featured: the current entry (the latest started,
// not ended), the scheduled ones and the last FeaturedPast others.
func (a *Admin) Featured(ctx context.Context, now time.Time) (*views.AdminFeaturedList, error) {
	all, err := a.featuredWhere(ctx, "order by f.starts_at desc, f.id desc")
	if err != nil {
		return nil, err
	}
	out := &views.AdminFeaturedList{}
	for i := range all {
		f := all[i]
		switch {
		case f.StartsAt.After(now):
			out.Scheduled = append([]views.AdminFeatured{f}, out.Scheduled...)
		case out.Current == nil && (f.EndsAt == nil || f.EndsAt.After(now)):
			out.Current = &f
		case len(out.Past) < views.FeaturedPast:
			out.Past = append(out.Past, f)
		}
	}
	return out, nil
}

// CreateFeatured adds a plot of the week. The photo must be visible; ends
// may be nil (until replaced).
func (a *Admin) CreateFeatured(ctx context.Context, adminID, photoID, text string, starts time.Time, ends *time.Time) (int, error) {
	id, err := parsePhotoID(photoID)
	if err != nil {
		return 0, invalid("There is no photo with this id.")
	}
	text = strings.TrimSpace(text)
	if len([]rune(text)) > views.FeaturedTextMax {
		return 0, invalid(fmt.Sprintf("The text is longer than %d characters.", views.FeaturedTextMax))
	}
	if ends != nil && !ends.After(starts) {
		return 0, invalid("The end must be after the start.")
	}
	var fid int
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		var ok bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from visible_photos where id = $1)`, id).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return invalid("There is no visible photo with this id.")
		}
		err := tx.QueryRow(ctx, `insert into featured (photo_id, body, starts_at, ends_at) values ($1, $2, $3, $4) returning id`,
			id, text, starts, ends).Scan(&fid)
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "featured.create", fmt.Sprintf("featured:%d", fid), "", nil,
			map[string]any{"photo_id": id, "body": text, "starts_at": starts, "ends_at": ends})
	})
	return fid, finish("create featured", err)
}

// EndFeatured ends a started entry now. ErrUnchanged if it has ended or not started.
func (a *Admin) EndFeatured(ctx context.Context, adminID string, id int, now time.Time) error {
	err := pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		var old *time.Time
		err := tx.QueryRow(ctx, `
			update featured f set ends_at = $2 from featured o
			where o.id = f.id and f.id = $1 and f.starts_at <= $2 and (f.ends_at is null or f.ends_at > $2)
			returning o.ends_at`, id, now).Scan(&old)
		if errors.Is(err, pgx.ErrNoRows) {
			return a.featuredMissingOr(ctx, tx, id)
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "featured.end", fmt.Sprintf("featured:%d", id), "",
			map[string]any{"ends_at": old}, map[string]any{"ends_at": now})
	})
	return finish("end featured", err)
}

// DeleteFeatured deletes an entry that has not started. Started entries are
// never deleted, so the history of who was featured stays.
func (a *Admin) DeleteFeatured(ctx context.Context, adminID string, id int, now time.Time) error {
	err := pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		var photo uuid.UUID
		var body string
		var starts time.Time
		err := tx.QueryRow(ctx, `delete from featured where id = $1 and starts_at > $2 returning photo_id, body, starts_at`,
			id, now).Scan(&photo, &body, &starts)
		if errors.Is(err, pgx.ErrNoRows) {
			return a.featuredMissingOr(ctx, tx, id)
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "featured.delete", fmt.Sprintf("featured:%d", id), "",
			map[string]any{"photo_id": photo, "body": body, "starts_at": starts}, nil)
	})
	return finish("delete featured", err)
}

func (a *Admin) featuredMissingOr(ctx context.Context, tx pgx.Tx, id int) error {
	var exists bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from featured where id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return views.ErrNotFound
	}
	return ErrUnchanged
}

// Surveyors.

const adminSurveyorCols = `s.id, s.name, coalesce(s.display_name, ''), coalesce(s.email, ''), s.joined_at,
	(select count(*) from photos where surveyor_id = s.id),
	(select count(*) from photos where surveyor_id = s.id and hidden_at is not null),
	(select count(*) from sessions where surveyor_id = s.id and expires_at > $1),
	s.blocked_at`

func scanAdminSurveyor(r pgx.CollectableRow) (views.AdminSurveyorItem, error) {
	var s views.AdminSurveyorItem
	var id uuid.UUID
	err := r.Scan(&id, &s.Name, &s.Display, &s.Email, &s.JoinedAt, &s.Photos, &s.Hidden, &s.Sessions, &s.BlockedAt)
	s.JoinedAt = s.JoinedAt.UTC()
	return s, err
}

// Surveyors returns one page of every Surveyor, most recently joined first.
func (a *Admin) Surveyors(ctx context.Context, page int, now time.Time) (*views.AdminSurveyors, error) {
	var total int
	if err := a.d.Pool.QueryRow(ctx, `select count(*) from surveyors`).Scan(&total); err != nil {
		return nil, fmt.Errorf("db: admin surveyors: %w", err)
	}
	pg, err := pagination(page, views.SurveyorsPerPage, total)
	if err != nil {
		return nil, err
	}
	rows, err := a.d.Pool.Query(ctx, `select `+adminSurveyorCols+` from surveyors s
		order by s.joined_at desc, s.id limit $2 offset $3`, now, pg.PerPage, offset(pg))
	if err != nil {
		return nil, fmt.Errorf("db: admin surveyors: %w", err)
	}
	list, err := pgx.CollectRows(rows, scanAdminSurveyor)
	if err != nil {
		return nil, fmt.Errorf("db: admin surveyors: %w", err)
	}
	return &views.AdminSurveyors{Surveyors: list, Pagination: pg}, nil
}

// surveyorID looks a Surveyor up by name, any case.
func surveyorID(ctx context.Context, q querier, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `select id from surveyors where lower(name) = lower($1)`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, views.ErrNotFound
	}
	return id, err
}

// Surveyor returns the admin page of one Surveyor.
func (a *Admin) Surveyor(ctx context.Context, name string, now time.Time) (*views.AdminSurveyor, error) {
	rows, err := a.d.Pool.Query(ctx, `select `+adminSurveyorCols+`, s.bio, coalesce(s.blocked_reason, ''),
		(select count(*) from photos where surveyor_id = s.id and hidden_at = s.blocked_at)
		from surveyors s where lower(s.name) = lower($2)`, now, name)
	if err != nil {
		return nil, fmt.Errorf("db: admin surveyor: %w", err)
	}
	var id uuid.UUID
	out, err := pgx.CollectExactlyOneRow(rows, func(r pgx.CollectableRow) (*views.AdminSurveyor, error) {
		x := &views.AdminSurveyor{}
		s := &x.Surveyor
		err := r.Scan(&id, &s.Name, &s.Display, &s.Email, &s.JoinedAt, &s.Photos, &s.Hidden, &s.Sessions, &s.BlockedAt,
			&x.Bio, &x.BlockedReason, &x.HiddenByBlock)
		s.JoinedAt = s.JoinedAt.UTC()
		return x, err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: admin surveyor: %w", err)
	}
	if out.Photos, err = a.photos(ctx, "where ph.surveyor_id = $1", newestFirst+" limit $2", id, views.AdminPerPage); err != nil {
		return nil, err
	}
	if out.Sessions, err = a.signIns(ctx, id.String(), 20); err != nil {
		return nil, err
	}
	if out.Actions, err = a.actions(ctx, "where a.target = $1", "limit 50", "surveyor:"+id.String()); err != nil {
		return nil, err
	}
	return out, nil
}

// BlockSurveyor blocks a Surveyor: their sessions end and they get no new
// ones. With hidePhotos every visible photo of theirs is hidden with the
// same reason and at the block's time, so that unblocking can find them.
// ErrUnchanged if they are blocked already.
func (a *Admin) BlockSurveyor(ctx context.Context, adminID, name, reason string, hidePhotos bool, now time.Time) error {
	reason, err := checkReason(reason)
	if err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		id, err := surveyorID(ctx, tx, name)
		if err != nil {
			return err
		}
		if id.String() == adminID {
			return invalid("You cannot block yourself.")
		}
		tag, err := tx.Exec(ctx, `update surveyors set blocked_at = $2, blocked_reason = $3 where id = $1 and blocked_at is null`,
			id, now, reason)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrUnchanged
		}
		ended, err := endSessions(ctx, tx, id)
		if err != nil {
			return err
		}
		var hidden []uuid.UUID
		if hidePhotos {
			rows, err := tx.Query(ctx, `select id from visible_photos where surveyor_id = $1 order by uploaded_at, id`, id)
			if err != nil {
				return err
			}
			if hidden, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
				return err
			}
		}
		if err := audit(ctx, tx, adminID, "surveyor.block", "surveyor:"+id.String(), reason, nil,
			map[string]any{"sessions_ended": ended, "photos_hidden": len(hidden)}); err != nil {
			return err
		}
		for _, p := range hidden {
			if err := hideInTx(ctx, tx, adminID, p, reason, now); err != nil {
				return err
			}
		}
		return nil
	})
	return finish("block surveyor", err)
}

// UnblockSurveyor lifts a block. With unhidePhotos the photos hidden
// together with the block are shown again; others stay hidden.
func (a *Admin) UnblockSurveyor(ctx context.Context, adminID, name string, unhidePhotos bool) error {
	err := pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		id, err := surveyorID(ctx, tx, name)
		if err != nil {
			return err
		}
		var at time.Time
		var reason string
		err = tx.QueryRow(ctx, `
			update surveyors s set blocked_at = null, blocked_reason = null from surveyors o
			where o.id = s.id and s.id = $1 and s.blocked_at is not null
			returning o.blocked_at, o.blocked_reason`, id).Scan(&at, &reason)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnchanged
		}
		if err != nil {
			return err
		}
		var shown []uuid.UUID
		if unhidePhotos {
			rows, err := tx.Query(ctx, `select id from photos where surveyor_id = $1 and hidden_at = $2 order by uploaded_at, id`, id, at)
			if err != nil {
				return err
			}
			if shown, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
				return err
			}
		}
		if err := audit(ctx, tx, adminID, "surveyor.unblock", "surveyor:"+id.String(), "",
			map[string]any{"blocked_at": at, "blocked_reason": reason}, map[string]any{"photos_unhidden": len(shown)}); err != nil {
			return err
		}
		for _, p := range shown {
			if err := unhideInTx(ctx, tx, adminID, p); err != nil {
				return err
			}
		}
		return nil
	})
	return finish("unblock surveyor", err)
}

// endSessions deletes every session of a Surveyor, also nameless ones of
// their address.
func endSessions(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	tag, err := tx.Exec(ctx, `
		delete from sessions where surveyor_id = $1
		   or lower(email) = (select lower(email) from surveyors where id = $1)`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// EndSessions signs a Surveyor out everywhere without blocking them.
func (a *Admin) EndSessions(ctx context.Context, adminID, name string) error {
	err := pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		id, err := surveyorID(ctx, tx, name)
		if err != nil {
			return err
		}
		n, err := endSessions(ctx, tx, id)
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "surveyor.end_sessions", "surveyor:"+id.String(), "", nil,
			map[string]any{"sessions_ended": n})
	})
	return finish("end sessions", err)
}

// ClearBio empties a Surveyor's bio; the old text is kept in the audit row.
func (a *Admin) ClearBio(ctx context.Context, adminID, name, reason string) error {
	reason, err := checkReason(reason)
	if err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		id, err := surveyorID(ctx, tx, name)
		if err != nil {
			return err
		}
		var old string
		err = tx.QueryRow(ctx, `
			update surveyors s set bio = '' from surveyors o
			where o.id = s.id and s.id = $1 and s.bio <> '' returning o.bio`, id).Scan(&old)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnchanged
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "surveyor.clear_bio", "surveyor:"+id.String(), reason,
			map[string]any{"bio": old}, map[string]any{"bio": ""})
	})
	return finish("clear bio", err)
}

// ClearDisplayName removes a Surveyor's display name; the old one is kept
// in the audit row.
func (a *Admin) ClearDisplayName(ctx context.Context, adminID, name, reason string) error {
	reason, err := checkReason(reason)
	if err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		id, err := surveyorID(ctx, tx, name)
		if err != nil {
			return err
		}
		var old string
		err = tx.QueryRow(ctx, `
			update surveyors s set display_name = null from surveyors o
			where o.id = s.id and s.id = $1 and s.display_name is not null returning o.display_name`, id).Scan(&old)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnchanged
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "surveyor.clear_display_name", "surveyor:"+id.String(), reason,
			map[string]any{"display_name": old}, map[string]any{"display_name": nil})
	})
	return finish("clear display name", err)
}

// Place-name overrides.

// Places returns /admin/places: every override with the number of plots
// whose stored place name it matches.
func (a *Admin) Places(ctx context.Context) (*views.AdminPlaces, error) {
	rows, err := a.d.Pool.Query(ctx, `
		select o.country, o.name, o.kind, o.action, coalesce(o.replacement, ''), o.note,
		  (select count(*) from plots p join regions r on r.id = p.region_id
		   where p.place_name = o.name and r.country = o.country)
		from place_overrides o order by o.country, o.name`)
	if err != nil {
		return nil, fmt.Errorf("db: place overrides: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (views.PlaceOverride, error) {
		var o views.PlaceOverride
		return o, r.Scan(&o.Country, &o.Name, &o.Kind, &o.Action, &o.Replacement, &o.Note, &o.Plots)
	})
	if err != nil {
		return nil, fmt.Errorf("db: place overrides: %w", err)
	}
	rows, err = a.d.Pool.Query(ctx, `select country from place_naming order by country`)
	if err != nil {
		return nil, fmt.Errorf("db: place naming countries: %w", err)
	}
	countries, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("db: place naming countries: %w", err)
	}
	return &views.AdminPlaces{Overrides: list, Countries: countries}, nil
}

// checkOverride validates and trims an override's fields.
func checkOverride(o *views.PlaceOverride) error {
	o.Country, o.Name, o.Kind = strings.ToLower(strings.TrimSpace(o.Country)), strings.TrimSpace(o.Name), strings.TrimSpace(o.Kind)
	o.Replacement, o.Note = strings.TrimSpace(o.Replacement), strings.TrimSpace(o.Note)
	if o.Action == "" {
		o.Action = views.OverrideUndecided
	}
	switch {
	case o.Action != views.OverrideUndecided && o.Action != views.OverrideKeep && o.Action != views.OverrideReplace:
		return invalid("Choose undecided, keep or replace.")
	case o.Action == views.OverrideReplace && o.Replacement == "":
		return invalid("A replacement is needed for replace.")
	case o.Note == "":
		return invalid("Write a note: why this name is listed.")
	case len([]rune(o.Name)) > 200 || len([]rune(o.Replacement)) > 200 || len([]rune(o.Note)) > 2000:
		return invalid("A field is too long.")
	}
	return nil
}

// CreatePlaceOverride adds an override for a place name as imported.
func (a *Admin) CreatePlaceOverride(ctx context.Context, adminID string, o views.PlaceOverride) error {
	if err := checkOverride(&o); err != nil {
		return err
	}
	if o.Name == "" || o.Kind == "" {
		return invalid("Give the place name as imported and its OSM kind.")
	}
	err := pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		var known bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from place_naming where country = $1)`, o.Country).Scan(&known); err != nil {
			return err
		}
		if !known {
			return invalid("There are no naming rules for this country.")
		}
		tag, err := tx.Exec(ctx, `
			insert into place_overrides (country, name, kind, action, replacement, note)
			values ($1, $2, $3, $4, nullif($5, ''), $6) on conflict do nothing`,
			o.Country, o.Name, o.Kind, o.Action, o.Replacement, o.Note)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return invalid("This place name is listed already.")
		}
		o.Plots = 0
		return audit(ctx, tx, adminID, "place_override.create", "place_override:"+o.Country+"/"+o.Name, "", nil, o)
	})
	return finish("create place override", err)
}

// UpdatePlaceOverride sets the action, replacement and note of an override.
// A replace applies to plots claimed from now on; plots already named keep
// their names.
func (a *Admin) UpdatePlaceOverride(ctx context.Context, adminID string, o views.PlaceOverride) error {
	if err := checkOverride(&o); err != nil {
		return err
	}
	err := pgx.BeginFunc(ctx, a.d.Pool, func(tx pgx.Tx) error {
		var old views.PlaceOverride
		err := tx.QueryRow(ctx, `
			select country, name, kind, action, coalesce(replacement, ''), note from place_overrides
			where country = $1 and name = $2 for update`, o.Country, o.Name,
		).Scan(&old.Country, &old.Name, &old.Kind, &old.Action, &old.Replacement, &old.Note)
		if errors.Is(err, pgx.ErrNoRows) {
			return views.ErrNotFound
		}
		if err != nil {
			return err
		}
		if old.Action == o.Action && old.Replacement == o.Replacement && old.Note == o.Note {
			return ErrUnchanged
		}
		_, err = tx.Exec(ctx, `
			update place_overrides set action = $3, replacement = nullif($4, ''), note = $5
			where country = $1 and name = $2`, o.Country, o.Name, o.Action, o.Replacement, o.Note)
		if err != nil {
			return err
		}
		o.Kind = old.Kind
		return audit(ctx, tx, adminID, "place_override.update", "place_override:"+o.Country+"/"+o.Name, "", old, o)
	})
	return finish("update place override", err)
}
