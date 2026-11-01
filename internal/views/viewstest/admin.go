package viewstest

import (
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Admin fixtures (docs/ADMIN_BRIEF.md): an admin viewer, one hidden photo,
// one blocked Surveyor, a pending and an accepted invitation.

// AdminViewer returns a signed-in admin (adrian), signed in an hour ago.
func AdminViewer() *views.Viewer {
	v := Viewer()
	v.Admin, v.SignedInAt = true, Now.Add(-time.Hour)
	return v
}

// GonePhotoID is a photo Fake.Photo answers with views.ErrGone, as the
// database does for a hidden photo. The page fixtures keep every photo
// visible, so it is not one of them.
const GonePhotoID = "00000000-0000-4000-8000-999999999999"

// HiddenPhotoID is the admin fixtures' hidden photo.
var HiddenPhotoID = photoID(len(shots) - 2)

func adminPhotos(n int) []views.AdminPhoto {
	d := build()
	var out []views.AdminPhoto
	for i := len(d.photos) - 1; i >= 0 && len(out) < n; i-- {
		p := views.AdminPhoto{PhotoItem: d.photos[i]}
		if p.ID == HiddenPhotoID {
			at := Now.Add(-2 * time.Hour)
			p.HiddenAt, p.HiddenReason = &at, "Shows a face close up."
			p.FirstSurvey = false
		}
		out = append(out, p)
	}
	return out
}

func adminPhoto(id string) views.AdminPhoto {
	for _, p := range adminPhotos(100) {
		if p.ID == id {
			return p
		}
	}
	panic("viewstest: no photo " + id)
}

func adminPage(title, section string) views.AdminPage {
	return views.AdminPage{Page: views.Page{Title: title, Canonical: "/admin", NoIndex: true, Now: Now}, Section: section}
}

func adminActions() []views.AdminAction {
	return []views.AdminAction{
		{ID: 3, At: Now.Add(-time.Hour), Admin: "adrian", Action: "surveyor.block", Target: "spam-bot", Link: "/admin/surveyor/spam-bot",
			Reason: "Uploads screenshots.", After: `{"photos_hidden": 0, "sessions_ended": 1}`},
		{ID: 2, At: Now.Add(-2 * time.Hour), Admin: "adrian", Action: "photo.hide", Target: "photo " + HiddenPhotoID[:8], Link: "/admin/photo/" + HiddenPhotoID,
			Reason: "Shows a face close up.", Before: `{"plot": "…", "claim": {"photo_count": 1}}`, After: `{"plot": "…", "claim": {"photo_count": 0}}`},
		{ID: 1, At: Now.Add(-26 * time.Hour), Admin: "adrian", Action: "invite.create", Target: "noi@example.org", Link: "/admin/invites"},
	}
}

// AdminOverview returns /admin.
func AdminOverview() *views.AdminOverview {
	return &views.AdminOverview{
		AdminPage: adminPage("Admin", views.AdminOverviewSection),
		Counters:  views.AdminCounters{Surveyors: 6, NewSurveyors: 2, PendingInvites: 1, Photos24h: 3, Photos7d: 12, Plots: 13, Hidden: 1, Blocked: 1, Refusals7d: 4},
		Uploads:   adminPhotos(views.AdminLatest),
		SignIns: []views.AdminSignIn{
			{Surveyor: "adrian", Email: "adrian@example.org", SignedInAt: Now.Add(-time.Hour), LastSeen: Now.Add(-time.Minute)},
			{Email: "noi@example.org", SignedInAt: Now.Add(-3 * time.Hour), LastSeen: Now.Add(-3 * time.Hour)},
		},
		LinksSent: 4, LinksRedeemed: 2,
		Actions: adminActions(),
	}
}

// AdminPhotos returns /admin/photos?hidden=1.
func AdminPhotos() *views.AdminPhotos {
	return &views.AdminPhotos{
		AdminPage:  adminPage("Photos", views.AdminPhotosSection),
		Photos:     []views.AdminPhoto{adminPhoto(HiddenPhotoID)},
		Pagination: views.Pagination{Page: 1, PerPage: views.AdminPerPage, Total: 1},
		HiddenOnly: true,
	}
}

// AdminPhoto returns the admin page of the hidden photo, or of the featured one.
func AdminPhoto(hidden bool) *views.AdminPhotoDetail {
	id := FeaturedPhotoID
	if hidden {
		id = HiddenPhotoID
	}
	d := &views.AdminPhotoDetail{
		AdminPage: adminPage("Photo", views.AdminPhotosSection),
		Photo:     adminPhoto(id), Lat: 12.93201, Lng: 100.88713, Email: "ploy88@example.org",
		PlotPhotos: 2, FirstSurveyor: "adrian", Actions: adminActions()[1:2],
	}
	if !hidden {
		d.Featured = []views.AdminFeatured{{ID: 1, Photo: d.Photo.PhotoItem, Text: FeaturedText, StartsAt: Now.Add(-24 * time.Hour)}}
		d.Actions = nil
	}
	return d
}

// AdminConfirm returns the hide confirm page of the featured photo.
func AdminConfirm() *views.AdminConfirm {
	p := adminPhoto(FeaturedPhotoID)
	return &views.AdminConfirm{
		AdminPage: adminPage("Hide this photo", views.AdminPhotosSection),
		Heading:   "Hide this photo", Action: "/admin/photo/" + p.ID + "/hide", Photo: &p,
		Consequences: []string{"The plot passes to ploy88, whose photo is the next one; they become its first surveyor."},
		NeedReason:   true, Reason: "x", Error: "Give a reason; only admins see it.",
		Option: "hide_photos", OptionLabel: "Also hide all 3 visible photos, with the same reason",
		Button: "Hide the photo", Back: "/admin/photo/" + p.ID,
	}
}

// AdminFeatured returns /admin/featured.
func AdminFeatured() *views.AdminFeaturedList {
	p := adminPhoto(FeaturedPhotoID).PhotoItem
	end := Now.Add(-24 * time.Hour)
	return &views.AdminFeaturedList{
		AdminPage: adminPage("Plot of the week", views.AdminFeaturedSection),
		Current:   &views.AdminFeatured{ID: 2, Photo: p, Text: FeaturedText, StartsAt: Now.Add(-24 * time.Hour)},
		Scheduled: []views.AdminFeatured{{ID: 3, Photo: p, StartsAt: Now.Add(6 * 24 * time.Hour)}},
		Past:      []views.AdminFeatured{{ID: 1, Photo: p, Text: "Last week.", StartsAt: Now.Add(-8 * 24 * time.Hour), EndsAt: &end}},
		Form:      views.FeaturedForm{Photo: "nope", Error: "Paste a photo's address or id."},
	}
}

// AdminSurveyors returns /admin/surveyors.
func AdminSurveyors() *views.AdminSurveyors {
	blocked := Now.Add(-time.Hour)
	return &views.AdminSurveyors{
		AdminPage: adminPage("Surveyors", views.AdminSurveyorsSection),
		Surveyors: []views.AdminSurveyorItem{
			{Name: "spam-bot", Email: "bot@example.org", JoinedAt: Now.Add(-2 * time.Hour), Photos: 4, Hidden: 0, BlockedAt: &blocked},
			{Name: "adrian", Email: "adrian@example.org", JoinedAt: Now.Add(-60 * 24 * time.Hour), Photos: 12, Sessions: 2},
		},
		Pagination: views.Pagination{Page: 1, PerPage: views.SurveyorsPerPage, Total: 2},
	}
}

// AdminSurveyor returns the admin page of a Surveyor, blocked or not.
func AdminSurveyor(blocked bool) *views.AdminSurveyor {
	d := &views.AdminSurveyor{
		AdminPage: adminPage("ploy88", views.AdminSurveyorsSection),
		Surveyor:  views.AdminSurveyorItem{Name: "ploy88", Email: "ploy88@example.org", JoinedAt: Now.Add(-20 * 24 * time.Hour), Photos: 5, Hidden: 1, Sessions: 1},
		Bio:       "Jomtien every morning.",
		Photos:    adminPhotos(4),
		Sessions:  []views.AdminSignIn{{Surveyor: "ploy88", Email: "ploy88@example.org", SignedInAt: Now.Add(-48 * time.Hour), LastSeen: Now.Add(-time.Hour)}},
		Actions:   adminActions()[:1],
	}
	if blocked {
		at := Now.Add(-time.Hour)
		d.Surveyor.BlockedAt, d.BlockedReason, d.HiddenByBlock, d.Surveyor.Sessions, d.Sessions = &at, "Uploads screenshots.", 1, 0, nil
	}
	return d
}

// AdminPlaces returns /admin/places.
func AdminPlaces() *views.AdminPlaces {
	return &views.AdminPlaces{
		AdminPage: adminPage("Place names", views.AdminPlacesSection),
		Overrides: []views.PlaceOverride{
			{Country: "th", Name: "Ban Bang Krabueang", Kind: "village", Action: views.OverrideUndecided, Note: "Old village name.", Plots: 3},
			{Country: "th", Name: "Moo Ban Seri", Kind: "neighbourhood", Action: views.OverrideReplace, Replacement: "Seri Village", Note: "Estate.", Plots: 1},
		},
		Countries: []string{"th"},
		Form:      views.PlaceOverride{Country: "th"},
	}
}

// AdminInvites returns /admin/invites.
func AdminInvites() *views.AdminInvites {
	acc := Now.Add(-20 * time.Hour)
	return &views.AdminInvites{
		AdminPage: adminPage("Invitations", views.AdminInvitesSection),
		Invitations: []views.Invitation{
			{Email: "noi@example.org", InvitedBy: "adrian", Note: "Come and survey Jomtien.", InvitedAt: Now.Add(-26 * time.Hour)},
			{Email: "ploy88@example.org", InvitedBy: "adrian", InvitedAt: Now.Add(-30 * 24 * time.Hour), AcceptedAt: &acc, AcceptedAs: "ploy88"},
		},
		InviteOnly: true, Email: "ploy88@example.org", Error: "ploy88@example.org is already a Surveyor: ploy88.",
	}
}

// AdminLog returns /admin/log.
func AdminLog() *views.AdminLog {
	return &views.AdminLog{
		AdminPage:  adminPage("Admin log", views.AdminLogSection),
		Actions:    adminActions(),
		Pagination: views.Pagination{Page: 1, PerPage: views.AdminLogPerPage, Total: 3},
	}
}

// SignInAgain returns the admin guard's "sign in again" page.
func SignInAgain() *views.SignInAgain {
	return &views.SignInAgain{Page: views.Page{Title: "Sign in again", NoIndex: true}, SignIn: "/signin?next=%2Fadmin"}
}
