// Package countries is the list of countries Cartograph has a page for
// (/{slug}), imported or not. A country's provinces carry its code in
// regions.country; a country is "open" once it has any. Adding a country is
// a line here (docs/COUNTRY_PAGES_BRIEF.md).
package countries

import "strings"

// Country is one entry of the list.
type Country struct {
	Code   string // ISO 3166-1 alpha-2, lower case: regions.country
	Slug   string // URL slug, the English short name: /thailand
	NameEn string
	Native string // the country's own name; may be empty
	Lang   string // BCP 47 language tag of Native, e.g. "th"
}

// All is the list, by English name: Southeast Asia plus Taiwan, South Korea
// and Japan (docs/decisions.md, 2026-11-03).
var All = []Country{
	{"bn", "brunei", "Brunei", "Brunei Darussalam", "ms"},
	{"kh", "cambodia", "Cambodia", "កម្ពុជា", "km"},
	{"id", "indonesia", "Indonesia", "Indonesia", "id"},
	{"jp", "japan", "Japan", "日本", "ja"},
	{"la", "laos", "Laos", "ລາວ", "lo"},
	{"my", "malaysia", "Malaysia", "Malaysia", "ms"},
	{"mm", "myanmar", "Myanmar", "မြန်မာ", "my"},
	{"ph", "philippines", "Philippines", "Pilipinas", "fil"},
	{"sg", "singapore", "Singapore", "Singapore", "en"},
	{"kr", "south-korea", "South Korea", "대한민국", "ko"},
	{"tw", "taiwan", "Taiwan", "臺灣", "zh-Hant"},
	{"th", "thailand", "Thailand", "ประเทศไทย", "th"},
	{"tl", "timor-leste", "Timor-Leste", "Timor-Leste", "pt"},
	{"vn", "vietnam", "Vietnam", "Việt Nam", "vi"},
}

// BySlug returns the country with the given slug, in any case.
func BySlug(slug string) (Country, bool) {
	slug = strings.ToLower(slug)
	for _, c := range All {
		if c.Slug == slug {
			return c, true
		}
	}
	return Country{}, false
}

// ByCode returns the country with the given ISO code, in any case.
func ByCode(code string) (Country, bool) {
	code = strings.ToLower(code)
	for _, c := range All {
		if c.Code == code {
			return c, true
		}
	}
	return Country{}, false
}

// Reserved are the first path segments the site uses that no Go router
// registers: the JSON API and health check (mounted as wildcards), robots.txt
// and what the front proxy serves itself (deploy/Caddyfile). A country's
// slug must not be one of them nor the first segment of any registered
// route; cmd/api's test walks the real router for the latter.
var Reserved = []string{"api", "healthz", "robots.txt", "map", "tiles", "img", "favicon.ico", "sitemap.xml"}
