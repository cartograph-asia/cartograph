package countries

import (
	"regexp"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	slugRe := regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
	codes, slugs := map[string]bool{}, map[string]bool{}
	for i, c := range All {
		if len(c.Code) != 2 || c.Code != strings.ToLower(c.Code) {
			t.Errorf("%s: code %q is not two lower-case letters", c.NameEn, c.Code)
		}
		if !slugRe.MatchString(c.Slug) {
			t.Errorf("%s: slug %q is not lower-case words with hyphens", c.NameEn, c.Slug)
		}
		if codes[c.Code] || slugs[c.Slug] {
			t.Errorf("%s: code %q or slug %q is used twice", c.NameEn, c.Code, c.Slug)
		}
		codes[c.Code], slugs[c.Slug] = true, true
		if c.NameEn == "" {
			t.Errorf("%q: no English name", c.Slug)
		}
		if (c.Native == "") != (c.Lang == "") {
			t.Errorf("%s: native name and language tag go together", c.NameEn)
		}
		if i > 0 && All[i-1].NameEn >= c.NameEn {
			t.Errorf("%s: the list is by English name", c.NameEn)
		}
	}
}

func TestSlugsAreNotReserved(t *testing.T) {
	for _, r := range Reserved {
		if c, ok := BySlug(r); ok {
			t.Errorf("slug %q of %s is a reserved path", r, c.NameEn)
		}
	}
}

func TestLookup(t *testing.T) {
	if c, ok := BySlug("Thailand"); !ok || c.Code != "th" {
		t.Errorf("BySlug(Thailand) = %v, %v", c, ok)
	}
	if c, ok := ByCode("VN"); !ok || c.Slug != "vietnam" {
		t.Errorf("ByCode(VN) = %v, %v", c, ok)
	}
	if _, ok := BySlug("th"); ok {
		t.Error("a code is not a slug")
	}
	if _, ok := ByCode("xx"); ok {
		t.Error("unknown code found")
	}
}
