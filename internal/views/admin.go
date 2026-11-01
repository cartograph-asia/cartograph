package views

import "time"

// Page names of the admin area (docs/ADMIN_BRIEF.md). Each is one template
// in internal/web; internal/admin renders them.
const (
	PageAdminOverview  = "admin-overview"  // *AdminOverview
	PageAdminPhotos    = "admin-photos"    // *AdminPhotos
	PageAdminPhoto     = "admin-photo"     // *AdminPhotoDetail
	PageAdminConfirm   = "admin-confirm"   // *AdminConfirm
	PageAdminFeatured  = "admin-featured"  // *AdminFeaturedList
	PageAdminSurveyors = "admin-surveyors" // *AdminSurveyors
	PageAdminSurveyor  = "admin-surveyor"  // *AdminSurveyor
	PageAdminPlaces    = "admin-places"    // *AdminPlaces
	PageAdminInvites   = "admin-invites"   // *AdminInvites
	PageAdminLog       = "admin-log"       // *AdminLog
	PageSignInAgain    = "signin-again"    // *SignInAgain, rendered by internal/auth's admin guard
)

// Sizes of the admin lists.
const (
	AdminLatest     = 24 // uploads on the overview
	AdminSignIns    = 20 // sign-ins on the overview
	AdminActions    = 10 // admin actions on the overview
	AdminPerPage    = 48 // photos per page on /admin/photos
	AdminLogPerPage = 50 // actions per page on /admin/log
	FeaturedPast    = 20 // past plots of the week on /admin/featured
	FeaturedTextMax = 500
	ReasonMax       = 200 // characters of a hide or block reason
	InviteNoteMax   = 300
)

// Admin sections, for the admin navigation line (AdminPage.Section).
const (
	AdminOverviewSection  = "overview"
	AdminPhotosSection    = "photos"
	AdminInvitesSection   = "invites"
	AdminFeaturedSection  = "featured"
	AdminSurveyorsSection = "surveyors"
	AdminPlacesSection    = "places"
	AdminLogSection       = "log"
)

// AdminPage is the base of every admin page.
type AdminPage struct {
	Page
	Section string // one of the Admin*Section names
	Done    string // one line saying what the last action did; empty if none
}

// AdminPhoto is a photo as the admin area sees it, hidden or not.
type AdminPhoto struct {
	PhotoItem
	HiddenAt       *time.Time
	HiddenReason   string
	FilesRemovedAt *time.Time // the image files were deleted from the bucket
}

// RemoveFilesAfter is how long a photo must be hidden before its image
// files can be removed (docs/ADMIN_BRIEF.md, decision 2).
const RemoveFilesAfter = 24 * time.Hour

// CanRemoveFiles reports whether the photo's files may be removed at now.
func (p AdminPhoto) CanRemoveFiles(now time.Time) bool {
	return p.HiddenAt != nil && p.FilesRemovedAt == nil && !now.Before(p.HiddenAt.Add(RemoveFilesAfter))
}

// Hidden reports whether the photo is hidden.
func (p AdminPhoto) Hidden() bool { return p.HiddenAt != nil }

// AdminCounters is the overview's counters line.
type AdminCounters struct {
	Surveyors      int
	NewSurveyors   int // joined in the last 7 days
	PendingInvites int
	Photos24h      int
	Photos7d       int
	Plots          int // surveyed plots
	Hidden         int // hidden photos
	Blocked        int // blocked Surveyors
	Refusals7d     int // invitation-only mails sent in the last 7 days
}

// AdminSignIn is one session: who signed in and when.
type AdminSignIn struct {
	Surveyor   string // empty: no name chosen yet
	Email      string
	SignedInAt time.Time
	LastSeen   time.Time
}

// AdminAction is one row of the audit trail.
type AdminAction struct {
	ID     int64
	At     time.Time
	Admin  string // the admin's Surveyor name
	Action string // e.g. "photo.hide"
	Target string // human-readable target: a Surveyor name, an address, "photo 3f2a…"
	Link   string // admin page of the target; empty if none
	Reason string
	Before string // JSON text; empty for none
	After  string
}

// AdminOverview is /admin.
type AdminOverview struct {
	AdminPage
	Counters      AdminCounters
	Uploads       []AdminPhoto // the AdminLatest newest, hidden ones included
	SignIns       []AdminSignIn
	LinksSent     int // magic links created in the last 24 h
	LinksRedeemed int // of all links, redeemed in the last 24 h
	Actions       []AdminAction
}

// AdminPhotos is /admin/photos.
type AdminPhotos struct {
	AdminPage
	Photos     []AdminPhoto
	Pagination Pagination
	HiddenOnly bool   // ?hidden=1
	Surveyor   string // ?surveyor=name; empty for all
}

// AdminPhotoDetail is /admin/photo/{id}.
type AdminPhotoDetail struct {
	AdminPage
	Photo         AdminPhoto
	Lat, Lng      float64 // exact position; shown only here
	Email         string  // the Surveyor's address; empty for photos without an account
	PlotPhotos    int     // visible photos of the plot
	FirstSurveyor string  // the plot's first surveyor now; empty if the plot is blank
	Featured      []AdminFeatured
	Actions       []AdminAction
}

// HideEffect is what hiding a photo would do, for the confirm page.
type HideEffect struct {
	Claims   bool   // the photo holds the plot's claim
	PassesTo string // the Surveyor the claim passes to; empty if the plot turns blank
	Featured bool   // the photo is the current plot of the week
}

// AdminConfirm is the confirm page of a destructive action (hide a photo,
// block a Surveyor, clear a bio). It POSTs to Action with the CSRF field,
// "reason" when NeedReason, and Option as a checkbox when set.
type AdminConfirm struct {
	AdminPage
	Heading      string
	Action       string   // form action path
	Consequences []string // what will happen, in words
	Photo        *AdminPhoto
	NeedReason   bool
	Reason       string // prefilled after an error
	Option       string // name of an optional checkbox, e.g. "hide_photos"; empty for none
	OptionLabel  string
	Button       string
	Back         string // path of the page the admin came from
	Error        string
}

// AdminFeatured is one plot-of-the-week entry.
type AdminFeatured struct {
	ID       int
	Photo    PhotoItem
	Text     string
	StartsAt time.Time
	EndsAt   *time.Time
}

// FeaturedForm is the "new plot of the week" form.
type FeaturedForm struct {
	Photo  string // photo id or URL as typed
	Text   string
	Starts string // datetime-local value in Bangkok time; empty = now
	Ends   string // empty = until replaced
	Error  string
}

// AdminFeaturedList is /admin/featured.
type AdminFeaturedList struct {
	AdminPage
	Current   *AdminFeatured
	Scheduled []AdminFeatured // not started yet, soonest first
	Past      []AdminFeatured // ended or replaced, newest first, up to FeaturedPast
	Form      FeaturedForm
}

// AdminSurveyorItem is a Surveyor as the admin area sees them.
type AdminSurveyorItem struct {
	Name      string
	Display   string // display name; empty for none
	Email     string
	JoinedAt  time.Time
	Photos    int // all their photos, hidden ones included
	Hidden    int
	Sessions  int // unexpired sessions
	BlockedAt *time.Time
}

// AdminSurveyors is /admin/surveyors.
type AdminSurveyors struct {
	AdminPage
	Surveyors  []AdminSurveyorItem
	Pagination Pagination
}

// AdminSurveyor is /admin/surveyor/{name}.
type AdminSurveyor struct {
	AdminPage
	Surveyor      AdminSurveyorItem
	Bio           string
	BlockedReason string
	HiddenByBlock int          // photos hidden together with the block
	Photos        []AdminPhoto // the AdminPerPage newest
	Sessions      []AdminSignIn
	Actions       []AdminAction
}

// Place-override actions (table place_overrides).
const (
	OverrideUndecided = "undecided"
	OverrideKeep      = "keep"
	OverrideReplace   = "replace"
)

// PlaceOverride is one row of place_overrides.
type PlaceOverride struct {
	Country     string
	Name        string // place name_en as imported
	Kind        string // OSM place value
	Action      string // OverrideUndecided, OverrideKeep or OverrideReplace
	Replacement string
	Note        string
	Plots       int // plots whose stored place name is Name
}

// AdminPlaces is /admin/places.
type AdminPlaces struct {
	AdminPage
	Overrides []PlaceOverride
	Countries []string      // place_naming countries, for the new-row form
	Form      PlaceOverride // the new-row form, prefilled after an error
	Error     string
}

// Invitation is one row of invitations.
type Invitation struct {
	Email      string
	InvitedBy  string // the admin's Surveyor name
	Note       string
	InvitedAt  time.Time
	AcceptedAt *time.Time
	AcceptedAs string // Surveyor name the address chose
	RevokedAt  *time.Time
}

// Pending reports whether the invitation is neither accepted nor revoked.
func (i Invitation) Pending() bool { return i.AcceptedAt == nil && i.RevokedAt == nil }

// AdminInvites is /admin/invites.
type AdminInvites struct {
	AdminPage
	Invitations []Invitation // newest first
	InviteOnly  bool         // SIGNUP=invite
	Email       string       // the form, prefilled after an error
	Note        string
	Error       string
}

// AdminLog is /admin/log.
type AdminLog struct {
	AdminPage
	Actions    []AdminAction
	Pagination Pagination
}

// SignInAgain asks an admin whose session is older than
// ADMIN_SESSION_MAX_AGE to sign in again.
type SignInAgain struct {
	Page
	SignIn string // the sign-in path to continue to after signing out, with ?next= back here
}
