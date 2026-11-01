package auth

import (
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/cartograph-asia/cartograph/internal/views"
)

var nameRE = regexp.MustCompile(views.NamePattern)

// validName reports whether name follows the Surveyor name rules.
func validName(name string) bool { return nameRE.MatchString(name) }

// normalizeEmail trims and lower-cases an address and checks that it is a
// bare address (no display name, no comments) of reasonable length.
func normalizeEmail(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > 254 || strings.ContainsAny(s, " \t\r\n<>\",;") {
		return "", false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || a.Name != "" {
		return "", false
	}
	at := strings.LastIndexByte(s, '@')
	if at < 1 || !strings.Contains(s[at+1:], ".") || strings.HasSuffix(s, ".") {
		return "", false
	}
	return s, true
}

// safeNext returns p if it is a local path on this site, else "/". It
// rejects absolute and scheme-relative URLs ("//evil", "/\evil"), control
// characters and overly long values, so ?next= can never redirect off-site.
func safeNext(p string) string {
	if p == "" || len(p) > 512 || p[0] != '/' || strings.HasPrefix(p, "//") || strings.ContainsAny(p, "\\") {
		return "/"
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return "/"
		}
	}
	u, err := url.Parse(p)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "/"
	}
	return p
}

// cleanBio normalises line endings, drops control characters other than
// newlines and tabs, and trims surrounding space. ok is false when the
// result is longer than views.BioMaxLen characters or not valid UTF-8.
func cleanBio(s string) (string, bool) {
	if !utf8.ValidString(s) {
		return "", false
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, s)
	s = strings.TrimSpace(s)
	return s, utf8.RuneCountInString(s) <= views.BioMaxLen
}

// cleanDisplayName normalises a display name (docs/DISPLAY_NAME_BRIEF.md):
// NFC, surrounding space trimmed, runs of white space collapsed to one
// space. It allows letters and marks of any script, digits, spaces and
// . ' ’ -, needs at least one letter and at most views.DisplayNameMax
// characters. "" with ok true means no display name.
func cleanDisplayName(s string) (string, bool) {
	if !utf8.ValidString(s) {
		return "", false
	}
	s = strings.Join(strings.Fields(norm.NFC.String(s)), " ")
	if s == "" {
		return "", true
	}
	if utf8.RuneCountInString(s) > views.DisplayNameMax {
		return "", false
	}
	letter := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.Is(unicode.M, r), unicode.IsDigit(r), r == ' ', r == '.', r == '\'', r == '’', r == '-':
		default:
			return "", false
		}
	}
	return s, letter
}
