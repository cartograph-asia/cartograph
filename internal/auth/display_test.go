package auth

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/views"
)

func TestCleanDisplayName(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"", "", true},
		{"   ", "", true},
		{"Adrian Altner", "Adrian Altner", true},
		{"  Adrian \t  Altner \n", "Adrian Altner", true},
		{"Jean-Luc O'Neill Jr.", "Jean-Luc O'Neill Jr.", true},
		{"Zoë", "Zoë", true},
		{"Zoë", "Zoë", true}, // NFC
		{"พลอย ชลบุรี", "พลอย ชลบุรี", true},
		{"น้อย", "น้อย", true}, // marks
		{"Ploy 88", "Ploy 88", true},
		{"88", "", false}, // no letter
		{"...", "", false},
		{strings.Repeat("a", views.DisplayNameMax), strings.Repeat("a", views.DisplayNameMax), true},
		{strings.Repeat("a", views.DisplayNameMax+1), "", false},
		{"Adrian‮Altner", "", false}, // bidi override
		{"Ad‍rian", "", false},       // zero-width joiner
		{"Adrian 🙂", "", false},
		{"@adrian", "", false},
		{"a/b", "", false},
		{"<b>Bold</b>", "", false},
		{"Ad\x00rian", "", false},
		{string([]byte{0xff, 'a'}), "", false},
	} {
		got, ok := cleanDisplayName(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("cleanDisplayName(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestDisplayNameForms(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	e.store.addSurveyor("somchai-k", "somchai@example.org", false)
	want(t, e.redeem(e.requestLink("new@example.org", "/")), http.StatusSeeOther, "/signin/name")
	want(t, e.get("/signin/name"), http.StatusOK, "")
	csrf := e.csrf()

	// Bad and taken display names keep the form, the field is named.
	for _, c := range []struct{ display, msg string }{
		{"Ad‮rian", msgBadDisplay},
		{"Somchai-K", msgDisplayTaken},
	} {
		e.post("/signin/name", url.Values{"name": {"nok"}, "display_name": {c.display}, "csrf": {csrf}})
		cn := e.last().data.(*views.ChooseName)
		if cn.Error != c.msg || cn.ErrorField != "display" || cn.Name != "nok" {
			t.Fatalf("%q: %+v", c.display, cn)
		}
	}
	// Your own Surveyor name is fine as a display name.
	want(t, e.post("/signin/name", url.Values{"name": {"nok"}, "display_name": {"NOK"}, "csrf": {csrf}}), http.StatusSeeOther, "/")
	if status, m := e.me(); status != http.StatusOK || m["name"] != "nok" || m["label"] != "NOK" {
		t.Fatalf("me: %d %v", status, m)
	}

	// Settings change and clear it.
	want(t, e.get("/settings"), http.StatusOK, "")
	if ep := e.last().data.(*views.EditProfile); ep.Display != "NOK" {
		t.Fatalf("settings: %+v", ep)
	}
	want(t, e.post("/settings", url.Values{"display_name": {"  Nok   Pattaya "}, "csrf": {e.csrf()}}), http.StatusSeeOther, "/settings?saved=1")
	if _, m := e.me(); m["label"] != "Nok Pattaya" {
		t.Fatalf("label after save: %v", m)
	}
	e.post("/settings", url.Values{"display_name": {"somchai-k"}, "csrf": {e.csrf()}})
	if ep := e.last().data.(*views.EditProfile); ep.Error != msgDisplayTaken || ep.ErrorField != "display" {
		t.Fatalf("taken in settings: %+v", ep)
	}
	want(t, e.post("/settings", url.Values{"display_name": {""}, "csrf": {e.csrf()}}), http.StatusSeeOther, "/settings?saved=1")
	if _, m := e.me(); m["label"] != "nok" {
		t.Fatalf("label after clearing: %v", m)
	}
}
