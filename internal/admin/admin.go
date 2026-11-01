// Package admin serves the admin area under /admin (docs/ADMIN_BRIEF.md):
// the overview, moderation of photos and Surveyors, invitations, the plot
// of the week, place-name overrides and the audit trail.
//
// Every route sits behind internal/auth's AdminGuard. A GET never changes
// anything; every change is a POST that passes the same CSRF and
// same-origin check as the account forms, and the database writes its
// admin_actions row in the same transaction. Destructive actions go
// through a confirm page that states the consequences and asks for a
// reason. After a POST the handler redirects back with ?done=<key>, and
// the page shows the fixed line for that key.
package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/auth"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// Store is the database side of the admin area; *db.Admin implements it.
type Store interface {
	Overview(ctx context.Context, now time.Time) (*views.AdminOverview, error)
	Photos(ctx context.Context, page int, hiddenOnly bool, surveyor string) (*views.AdminPhotos, error)
	Photo(ctx context.Context, id string) (*views.AdminPhotoDetail, error)
	HideEffect(ctx context.Context, id string, now time.Time) (*views.HideEffect, error)
	HidePhoto(ctx context.Context, adminID, id, reason string, now time.Time) error
	UnhidePhoto(ctx context.Context, adminID, id string) error
	Featured(ctx context.Context, now time.Time) (*views.AdminFeaturedList, error)
	CreateFeatured(ctx context.Context, adminID, photoID, text string, starts time.Time, ends *time.Time) (int, error)
	EndFeatured(ctx context.Context, adminID string, id int, now time.Time) error
	DeleteFeatured(ctx context.Context, adminID string, id int, now time.Time) error
	Surveyors(ctx context.Context, page int, now time.Time) (*views.AdminSurveyors, error)
	Surveyor(ctx context.Context, name string, now time.Time) (*views.AdminSurveyor, error)
	BlockSurveyor(ctx context.Context, adminID, name, reason string, hidePhotos bool, now time.Time) error
	UnblockSurveyor(ctx context.Context, adminID, name string, unhidePhotos bool) error
	EndSessions(ctx context.Context, adminID, name string) error
	ClearBio(ctx context.Context, adminID, name, reason string) error
	ClearDisplayName(ctx context.Context, adminID, name, reason string) error
	Places(ctx context.Context) (*views.AdminPlaces, error)
	CreatePlaceOverride(ctx context.Context, adminID string, o views.PlaceOverride) error
	UpdatePlaceOverride(ctx context.Context, adminID string, o views.PlaceOverride) error
	Log(ctx context.Context, page int) (*views.AdminLog, error)
}

// Accounts is the account side the admin area needs; *auth.Auth implements it.
type Accounts interface {
	AdminGuard(next http.Handler) http.Handler
	CheckForm(w http.ResponseWriter, r *http.Request) bool
	InviteOnly() bool
	Invite(ctx context.Context, admin *views.Viewer, email, note string) error
	ResendInvite(ctx context.Context, admin *views.Viewer, email string) error
	RevokeInvite(ctx context.Context, admin *views.Viewer, email string) error
	Invitations(ctx context.Context) ([]views.Invitation, error)
}

var _ Store = (*db.Admin)(nil)
var _ Accounts = (*auth.Auth)(nil)

// Handlers serves /admin.
type Handlers struct {
	store  Store
	acc    Accounts
	render views.RenderFunc
	now    func() time.Time
}

// New returns the admin handlers.
func New(store Store, acc Accounts, render views.RenderFunc) *Handlers {
	return &Handlers{store: store, acc: acc, render: render, now: time.Now}
}

// Routes mounts the admin area on r, every route behind the guard:
//
//	GET  /admin                              overview
//	GET  /admin/photos?page=&hidden=1&surveyor=
//	GET  /admin/photo/{id}                   POST …/unhide
//	GET  /admin/photo/{id}/hide              confirm page; POST hides (field reason)
//	GET  /admin/featured                     POST creates; POST …/{id}/end, …/{id}/delete
//	GET  /admin/surveyors?page=              GET /admin/surveyor/{name}
//	GET  /admin/surveyor/{name}/block        confirm page; POST blocks (reason, hide_photos)
//	POST /admin/surveyor/{name}/unblock      field unhide_photos
//	POST /admin/surveyor/{name}/end-sessions
//	GET  /admin/surveyor/{name}/clear-bio    confirm page; POST clears (reason)
//	GET  /admin/surveyor/{name}/clear-display-name  confirm page; POST clears (reason)
//	GET  /admin/places                       POST /admin/places, POST /admin/places/{country}/{name}
//	GET  /admin/invites                      POST creates; POST /admin/invites/resend, …/revoke (field email)
//	GET  /admin/log?page=
func (h *Handlers) Routes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(h.acc.AdminGuard)
		r.Get("/", h.overview)
		r.Get("/photos", h.photos)
		r.Get("/photo/{id}", h.photo)
		r.Get("/photo/{id}/hide", h.hideForm)
		r.Post("/photo/{id}/hide", h.hide)
		r.Post("/photo/{id}/unhide", h.unhide)
		r.Get("/featured", h.featured)
		r.Post("/featured", h.createFeatured)
		r.Post("/featured/{fid}/end", h.endFeatured)
		r.Post("/featured/{fid}/delete", h.deleteFeatured)
		r.Get("/surveyors", h.surveyors)
		r.Get("/surveyor/{name}", h.surveyor)
		r.Get("/surveyor/{name}/block", h.blockForm)
		r.Post("/surveyor/{name}/block", h.block)
		r.Post("/surveyor/{name}/unblock", h.unblock)
		r.Post("/surveyor/{name}/end-sessions", h.endSessions)
		r.Get("/surveyor/{name}/clear-bio", h.clearBioForm)
		r.Post("/surveyor/{name}/clear-bio", h.clearBio)
		r.Get("/surveyor/{name}/clear-display-name", h.clearDisplayForm)
		r.Post("/surveyor/{name}/clear-display-name", h.clearDisplay)
		r.Get("/places", h.places)
		r.Post("/places", h.createPlace)
		r.Post("/places/{country}/{name}", h.updatePlace)
		r.Get("/invites", h.invites)
		r.Post("/invites", h.invite)
		r.Post("/invites/resend", h.resendInvite)
		r.Post("/invites/revoke", h.revokeInvite)
		r.Get("/log", h.log)
	})
}

// doneLines are the only lines ?done= can show; anything else shows nothing.
var doneLines = map[string]string{
	"hidden":           "The photo is hidden.",
	"unhidden":         "The photo is visible again.",
	"featured":         "The plot of the week is saved.",
	"featured-ended":   "The plot of the week has ended.",
	"featured-deleted": "The scheduled entry is deleted.",
	"blocked":          "The Surveyor is blocked and signed out.",
	"unblocked":        "The block is lifted.",
	"sessions-ended":   "The Surveyor is signed out everywhere.",
	"bio-cleared":      "The bio is cleared.",
	"display-cleared":  "The display name is cleared.",
	"place-added":      "The place name is listed.",
	"place-saved":      "The place name is saved.",
	"invited":          "The invitation is sent.",
	"resent":           "A new link is sent.",
	"revoked":          "The invitation is revoked.",
	"unchanged":        "Nothing changed: that was done already.",
}

// base returns an admin page base with the title, section and done line.
func (h *Handlers) base(r *http.Request, title, section string) views.AdminPage {
	return views.AdminPage{
		Page:    views.Page{Title: title, Canonical: r.URL.Path, NoIndex: true, Now: h.now()},
		Section: section,
		Done:    doneLines[r.URL.Query().Get("done")],
	}
}

func viewer(r *http.Request) *views.Viewer { return views.ViewerFrom(r.Context()) }

// back redirects to path with ?done=key.
func back(w http.ResponseWriter, r *http.Request, path, key string) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	http.Redirect(w, r, path+sep+"done="+key, http.StatusSeeOther)
}

// fail renders the error page for err: 404 for views.ErrNotFound, 500 else.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := http.StatusInternalServerError, "Something went wrong on our side. Please try again."
	if errors.Is(err, views.ErrNotFound) {
		status, msg = http.StatusNotFound, "There is nothing at this address."
	} else {
		slog.Error("admin", "path", r.URL.Path, "err", err)
	}
	title := "Error"
	if status == http.StatusNotFound {
		title = "Not found"
	}
	h.render(w, r, status, views.PageError, &views.Error{
		Page: views.Page{Title: title, NoIndex: true}, Status: status, Message: msg,
	})
}

// done finishes a POST: unchanged and success redirect back, the rest fails.
func (h *Handlers) done(w http.ResponseWriter, r *http.Request, err error, path, key string) {
	switch {
	case err == nil:
		back(w, r, path, key)
	case errors.Is(err, db.ErrUnchanged), errors.Is(err, auth.ErrNotPending):
		back(w, r, path, "unchanged")
	default:
		h.fail(w, r, err)
	}
}

// pageParam reads ?page=; anything but a positive integer is page 1.
func pageParam(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func (h *Handlers) overview(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Overview(r.Context(), h.now())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d.AdminPage = h.base(r, "Admin", views.AdminOverviewSection)
	h.render(w, r, http.StatusOK, views.PageAdminOverview, d)
}

func (h *Handlers) photos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	d, err := h.store.Photos(r.Context(), pageParam(r), q.Get("hidden") == "1", strings.TrimSpace(q.Get("surveyor")))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d.AdminPage = h.base(r, "Photos", views.AdminPhotosSection)
	f := url.Values{}
	if d.HiddenOnly {
		f.Set("hidden", "1")
	}
	if d.Surveyor != "" {
		f.Set("surveyor", d.Surveyor)
	}
	if e := f.Encode(); e != "" {
		d.Canonical += "?" + e
	}
	h.render(w, r, http.StatusOK, views.PageAdminPhotos, d)
}

func (h *Handlers) photo(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Photo(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d.AdminPage = h.base(r, "Photo", views.AdminPhotosSection)
	h.render(w, r, http.StatusOK, views.PageAdminPhoto, d)
}

func photoPath(id string) string { return "/admin/photo/" + url.PathEscape(id) }

// hideConfirm builds the hide confirm page for photo id.
func (h *Handlers) hideConfirm(r *http.Request, id string) (*views.AdminConfirm, error) {
	ctx := r.Context()
	d, err := h.store.Photo(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Photo.Hidden() {
		return nil, db.ErrUnchanged
	}
	e, err := h.store.HideEffect(ctx, id, h.now())
	if err != nil {
		return nil, err
	}
	var cs []string
	switch {
	case e.Claims && e.PassesTo != "":
		cs = append(cs, "The plot passes to "+e.PassesTo+", whose photo is the next one; they become its first surveyor.")
	case e.Claims:
		cs = append(cs, "It is the plot's only photo: the plot becomes blank again, and its province's coverage goes down by one plot.")
	default:
		cs = append(cs, "The plot keeps its first surveyor; its photo count goes down by one.")
	}
	if e.Featured {
		cs = append(cs, "It is the current plot of the week: the front page falls back to the latest first survey.")
	}
	cs = append(cs,
		"The photo disappears from every page, feed and the map; its page answers “This photo was removed.”",
		"The image files stay reachable at their addresses until files can be removed.",
		"The Surveyor is not told. Write to them yourself if needed.")
	p := d.Photo
	return &views.AdminConfirm{
		AdminPage:    h.base(r, "Hide this photo", views.AdminPhotosSection),
		Heading:      "Hide this photo",
		Action:       photoPath(d.Photo.ID) + "/hide",
		Consequences: cs,
		Photo:        &p,
		NeedReason:   true,
		Button:       "Hide the photo",
		Back:         photoPath(d.Photo.ID),
	}, nil
}

func (h *Handlers) hideForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.hideConfirm(r, id)
	if errors.Is(err, db.ErrUnchanged) {
		back(w, r, photoPath(id), "unchanged")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, views.PageAdminConfirm, c)
}

func (h *Handlers) hide(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	id := chi.URLParam(r, "id")
	reason := r.PostForm.Get("reason")
	err := h.store.HidePhoto(r.Context(), viewer(r).SurveyorID, id, reason, h.now())
	if errors.Is(err, db.ErrInvalid) {
		c, cerr := h.hideConfirm(r, id)
		if cerr != nil {
			h.done(w, r, cerr, photoPath(id), "")
			return
		}
		c.Reason, c.Error = reason, db.InvalidMessage(err)
		h.render(w, r, http.StatusBadRequest, views.PageAdminConfirm, c)
		return
	}
	h.done(w, r, err, photoPath(id), "hidden")
}

func (h *Handlers) unhide(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	id := chi.URLParam(r, "id")
	h.done(w, r, h.store.UnhidePhoto(r.Context(), viewer(r).SurveyorID, id), photoPath(id), "unhidden")
}

// Plot of the week.

// zone is Bangkok time, in which the featured form's dates are entered.
var zone = time.FixedZone("ICT", 7*60*60)

const localLayout = "2006-01-02T15:04"

var uuidRE = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

func (h *Handlers) featuredPage(r *http.Request, form views.FeaturedForm) (*views.AdminFeaturedList, error) {
	d, err := h.store.Featured(r.Context(), h.now())
	if err != nil {
		return nil, err
	}
	d.AdminPage = h.base(r, "Plot of the week", views.AdminFeaturedSection)
	d.Form = form
	return d, nil
}

func (h *Handlers) featured(w http.ResponseWriter, r *http.Request) {
	d, err := h.featuredPage(r, views.FeaturedForm{Photo: r.URL.Query().Get("photo")})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, views.PageAdminFeatured, d)
}

func (h *Handlers) createFeatured(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	f := views.FeaturedForm{
		Photo: strings.TrimSpace(r.PostForm.Get("photo")), Text: r.PostForm.Get("text"),
		Starts: strings.TrimSpace(r.PostForm.Get("starts")), Ends: strings.TrimSpace(r.PostForm.Get("ends")),
	}
	again := func(msg string) {
		f.Error = msg
		d, err := h.featuredPage(r, f)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		h.render(w, r, http.StatusBadRequest, views.PageAdminFeatured, d)
	}
	id := uuidRE.FindString(f.Photo)
	if id == "" {
		again("Paste a photo's address or id.")
		return
	}
	starts := h.now()
	if f.Starts != "" {
		t, err := time.ParseInLocation(localLayout, f.Starts, zone)
		if err != nil {
			again("The start is not a date and time.")
			return
		}
		starts = t
	}
	var ends *time.Time
	if f.Ends != "" {
		t, err := time.ParseInLocation(localLayout, f.Ends, zone)
		if err != nil {
			again("The end is not a date and time.")
			return
		}
		ends = &t
	}
	_, err := h.store.CreateFeatured(r.Context(), viewer(r).SurveyorID, id, f.Text, starts, ends)
	if errors.Is(err, db.ErrInvalid) {
		again(db.InvalidMessage(err))
		return
	}
	h.done(w, r, err, "/admin/featured", "featured")
}

func featuredID(r *http.Request) (int, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, "fid"))
	return n, err == nil && n > 0
}

func (h *Handlers) endFeatured(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	id, ok := featuredID(r)
	if !ok {
		h.fail(w, r, views.ErrNotFound)
		return
	}
	h.done(w, r, h.store.EndFeatured(r.Context(), viewer(r).SurveyorID, id, h.now()), "/admin/featured", "featured-ended")
}

func (h *Handlers) deleteFeatured(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	id, ok := featuredID(r)
	if !ok {
		h.fail(w, r, views.ErrNotFound)
		return
	}
	h.done(w, r, h.store.DeleteFeatured(r.Context(), viewer(r).SurveyorID, id, h.now()), "/admin/featured", "featured-deleted")
}

// Surveyors.

func surveyorPath(name string) string { return "/admin/surveyor/" + url.PathEscape(name) }

func (h *Handlers) surveyors(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Surveyors(r.Context(), pageParam(r), h.now())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d.AdminPage = h.base(r, "Surveyors", views.AdminSurveyorsSection)
	h.render(w, r, http.StatusOK, views.PageAdminSurveyors, d)
}

func (h *Handlers) surveyor(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Surveyor(r.Context(), chi.URLParam(r, "name"), h.now())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d.AdminPage = h.base(r, d.Surveyor.Name, views.AdminSurveyorsSection)
	h.render(w, r, http.StatusOK, views.PageAdminSurveyor, d)
}

func (h *Handlers) blockConfirm(r *http.Request) (*views.AdminConfirm, error) {
	d, err := h.store.Surveyor(r.Context(), chi.URLParam(r, "name"), h.now())
	if err != nil {
		return nil, err
	}
	if d.Surveyor.BlockedAt != nil {
		return nil, db.ErrUnchanged
	}
	name := d.Surveyor.Name
	cs := []string{
		fmt.Sprintf("%s is signed out everywhere (%s now) and cannot sign in again: a sign-in link to their address ends on a page saying the account is blocked.",
			name, plural(d.Surveyor.Sessions, "session")),
		"They cannot upload any more.",
		"Their profile and photos stay public unless you also hide the photos.",
		"They are not told. Write to them yourself if needed.",
	}
	c := &views.AdminConfirm{
		AdminPage:    h.base(r, "Block "+name, views.AdminSurveyorsSection),
		Heading:      "Block " + name,
		Action:       surveyorPath(name) + "/block",
		Consequences: cs,
		NeedReason:   true,
		Button:       "Block " + name,
		Back:         surveyorPath(name),
	}
	if visible := d.Surveyor.Photos - d.Surveyor.Hidden; visible > 0 {
		c.Option = "hide_photos"
		c.OptionLabel = fmt.Sprintf("Also hide all %s, with the same reason", plural(visible, "visible photo"))
	}
	return c, nil
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// confirmed renders a confirm page from build, or redirects when there is
// nothing to do.
func (h *Handlers) confirmed(w http.ResponseWriter, r *http.Request, build func(*http.Request) (*views.AdminConfirm, error)) {
	c, err := build(r)
	if errors.Is(err, db.ErrUnchanged) {
		back(w, r, surveyorPath(chi.URLParam(r, "name")), "unchanged")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, views.PageAdminConfirm, c)
}

// againConfirm re-renders a confirm page with the reason and an error.
func (h *Handlers) againConfirm(w http.ResponseWriter, r *http.Request, build func(*http.Request) (*views.AdminConfirm, error), err error) {
	c, cerr := build(r)
	if cerr != nil {
		h.done(w, r, cerr, surveyorPath(chi.URLParam(r, "name")), "")
		return
	}
	c.Reason, c.Error = r.PostForm.Get("reason"), db.InvalidMessage(err)
	h.render(w, r, http.StatusBadRequest, views.PageAdminConfirm, c)
}

func (h *Handlers) blockForm(w http.ResponseWriter, r *http.Request) {
	h.confirmed(w, r, h.blockConfirm)
}

func (h *Handlers) block(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	name := chi.URLParam(r, "name")
	err := h.store.BlockSurveyor(r.Context(), viewer(r).SurveyorID, name, r.PostForm.Get("reason"),
		r.PostForm.Get("hide_photos") == "1", h.now())
	if errors.Is(err, db.ErrInvalid) {
		h.againConfirm(w, r, h.blockConfirm, err)
		return
	}
	h.done(w, r, err, surveyorPath(name), "blocked")
}

func (h *Handlers) unblock(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	name := chi.URLParam(r, "name")
	err := h.store.UnblockSurveyor(r.Context(), viewer(r).SurveyorID, name, r.PostForm.Get("unhide_photos") == "1")
	h.done(w, r, err, surveyorPath(name), "unblocked")
}

func (h *Handlers) endSessions(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	name := chi.URLParam(r, "name")
	h.done(w, r, h.store.EndSessions(r.Context(), viewer(r).SurveyorID, name), surveyorPath(name), "sessions-ended")
}

func (h *Handlers) clearBioConfirm(r *http.Request) (*views.AdminConfirm, error) {
	d, err := h.store.Surveyor(r.Context(), chi.URLParam(r, "name"), h.now())
	if err != nil {
		return nil, err
	}
	if d.Bio == "" {
		return nil, db.ErrUnchanged
	}
	name := d.Surveyor.Name
	return &views.AdminConfirm{
		AdminPage: h.base(r, "Clear the bio of "+name, views.AdminSurveyorsSection),
		Heading:   "Clear the bio of " + name,
		Action:    surveyorPath(name) + "/clear-bio",
		Consequences: []string{
			"The bio disappears from " + name + "’s profile. They can write a new one in their settings.",
			"The old text is kept in the admin log: “" + d.Bio + "”",
		},
		NeedReason: true,
		Button:     "Clear the bio",
		Back:       surveyorPath(name),
	}, nil
}

func (h *Handlers) clearBioForm(w http.ResponseWriter, r *http.Request) {
	h.confirmed(w, r, h.clearBioConfirm)
}

func (h *Handlers) clearBio(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	name := chi.URLParam(r, "name")
	err := h.store.ClearBio(r.Context(), viewer(r).SurveyorID, name, r.PostForm.Get("reason"))
	if errors.Is(err, db.ErrInvalid) {
		h.againConfirm(w, r, h.clearBioConfirm, err)
		return
	}
	h.done(w, r, err, surveyorPath(name), "bio-cleared")
}

func (h *Handlers) clearDisplayConfirm(r *http.Request) (*views.AdminConfirm, error) {
	d, err := h.store.Surveyor(r.Context(), chi.URLParam(r, "name"), h.now())
	if err != nil {
		return nil, err
	}
	if d.Surveyor.Display == "" {
		return nil, db.ErrUnchanged
	}
	name := d.Surveyor.Name
	return &views.AdminConfirm{
		AdminPage: h.base(r, "Clear the display name of "+name, views.AdminSurveyorsSection),
		Heading:   "Clear the display name of " + name,
		Action:    surveyorPath(name) + "/clear-display-name",
		Consequences: []string{
			name + " is shown with their Surveyor name again everywhere. They can choose a new display name in their settings.",
			"The old display name is kept in the admin log: “" + d.Surveyor.Display + "”",
		},
		NeedReason: true,
		Button:     "Clear the display name",
		Back:       surveyorPath(name),
	}, nil
}

func (h *Handlers) clearDisplayForm(w http.ResponseWriter, r *http.Request) {
	h.confirmed(w, r, h.clearDisplayConfirm)
}

func (h *Handlers) clearDisplay(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	name := chi.URLParam(r, "name")
	err := h.store.ClearDisplayName(r.Context(), viewer(r).SurveyorID, name, r.PostForm.Get("reason"))
	if errors.Is(err, db.ErrInvalid) {
		h.againConfirm(w, r, h.clearDisplayConfirm, err)
		return
	}
	h.done(w, r, err, surveyorPath(name), "display-cleared")
}

// Place names.

func (h *Handlers) placesPage(r *http.Request, form views.PlaceOverride, msg string) (*views.AdminPlaces, error) {
	d, err := h.store.Places(r.Context())
	if err != nil {
		return nil, err
	}
	d.AdminPage = h.base(r, "Place names", views.AdminPlacesSection)
	d.Form, d.Error = form, msg
	return d, nil
}

func (h *Handlers) places(w http.ResponseWriter, r *http.Request) {
	d, err := h.placesPage(r, views.PlaceOverride{Country: "th"}, "")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, views.PageAdminPlaces, d)
}

func overrideFrom(r *http.Request) views.PlaceOverride {
	f := r.PostForm
	return views.PlaceOverride{Country: f.Get("country"), Name: f.Get("name"), Kind: f.Get("kind"),
		Action: f.Get("action"), Replacement: f.Get("replacement"), Note: f.Get("note")}
}

func (h *Handlers) placesAgain(w http.ResponseWriter, r *http.Request, form views.PlaceOverride, err error) {
	d, perr := h.placesPage(r, form, db.InvalidMessage(err))
	if perr != nil {
		h.fail(w, r, perr)
		return
	}
	h.render(w, r, http.StatusBadRequest, views.PageAdminPlaces, d)
}

func (h *Handlers) createPlace(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	o := overrideFrom(r)
	err := h.store.CreatePlaceOverride(r.Context(), viewer(r).SurveyorID, o)
	if errors.Is(err, db.ErrInvalid) {
		h.placesAgain(w, r, o, err)
		return
	}
	h.done(w, r, err, "/admin/places", "place-added")
}

func (h *Handlers) updatePlace(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	o := overrideFrom(r)
	o.Country, o.Name = chi.URLParam(r, "country"), chi.URLParam(r, "name")
	err := h.store.UpdatePlaceOverride(r.Context(), viewer(r).SurveyorID, o)
	if errors.Is(err, db.ErrInvalid) {
		h.placesAgain(w, r, views.PlaceOverride{Country: "th"}, fmt.Errorf("%w: %s: %s", db.ErrInvalid, o.Name, db.InvalidMessage(err)))
		return
	}
	h.done(w, r, err, "/admin/places", "place-saved")
}

// Invitations.

func (h *Handlers) invitesPage(r *http.Request, email, note, msg string) (*views.AdminInvites, error) {
	list, err := h.acc.Invitations(r.Context())
	if err != nil {
		return nil, err
	}
	return &views.AdminInvites{
		AdminPage:   h.base(r, "Invitations", views.AdminInvitesSection),
		Invitations: list, InviteOnly: h.acc.InviteOnly(), Email: email, Note: note, Error: msg,
	}, nil
}

func (h *Handlers) invites(w http.ResponseWriter, r *http.Request) {
	d, err := h.invitesPage(r, "", "", "")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, views.PageAdminInvites, d)
}

// inviteMessage turns an invitation error into a line for the form, or ""
// when it is not the admin's input that failed.
func inviteMessage(err error, email string) string {
	var ae *auth.AccountError
	switch {
	case errors.As(err, &ae):
		return email + " is already a Surveyor: " + ae.Name + "."
	case errors.Is(err, auth.ErrBadEmail):
		return "Please enter a valid e-mail address."
	case errors.Is(err, auth.ErrNoteTooLong):
		return fmt.Sprintf("The note can be at most %d characters.", views.InviteNoteMax)
	case errors.Is(err, auth.ErrTooManyLinks):
		return "Too many links went to this address in the last hour. Try again later."
	case errors.Is(err, auth.ErrInviteNotSent):
		return "The invitation is saved, but the e-mail could not be sent. Try “Send a new link” in a few minutes."
	}
	return ""
}

func (h *Handlers) inviteAgain(w http.ResponseWriter, r *http.Request, err error, email, note string) bool {
	msg := inviteMessage(err, strings.TrimSpace(email))
	if msg == "" {
		return false
	}
	if errors.Is(err, auth.ErrInviteNotSent) {
		slog.Error("admin invite", "err", err)
	}
	d, perr := h.invitesPage(r, email, note, msg)
	if perr != nil {
		h.fail(w, r, perr)
		return true
	}
	h.render(w, r, http.StatusBadRequest, views.PageAdminInvites, d)
	return true
}

func (h *Handlers) invite(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	email, note := r.PostForm.Get("email"), r.PostForm.Get("note")
	err := h.acc.Invite(r.Context(), viewer(r), email, note)
	if h.inviteAgain(w, r, err, email, note) {
		return
	}
	h.done(w, r, err, "/admin/invites", "invited")
}

func (h *Handlers) resendInvite(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	email := r.PostForm.Get("email")
	err := h.acc.ResendInvite(r.Context(), viewer(r), email)
	if h.inviteAgain(w, r, err, "", "") {
		return
	}
	h.done(w, r, err, "/admin/invites", "resent")
}

func (h *Handlers) revokeInvite(w http.ResponseWriter, r *http.Request) {
	if !h.acc.CheckForm(w, r) {
		return
	}
	err := h.acc.RevokeInvite(r.Context(), viewer(r), r.PostForm.Get("email"))
	if errors.Is(err, auth.ErrBadEmail) {
		err = views.ErrNotFound
	}
	h.done(w, r, err, "/admin/invites", "revoked")
}

func (h *Handlers) log(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Log(r.Context(), pageParam(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d.AdminPage = h.base(r, "Admin log", views.AdminLogSection)
	h.render(w, r, http.StatusOK, views.PageAdminLog, d)
}
