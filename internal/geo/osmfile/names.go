package osmfile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/paulmach/osm"
	"golang.org/x/text/unicode/norm"

	"github.com/cartograph-asia/cartograph/internal/countries"
)

// The rules in this file turn a relation's or a node's OSM tags into the
// names, slugs and kind words the importer stores, driven by a country's
// countries.Import (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, stage 2). They are
// pure functions; for Thailand's settings every one of them gives what the
// import gave before the settings existed.

// MatchLevel reports whether r belongs to level l by its tags: a
// boundary=administrative relation at one of l.AdminLevels whose ISO3166-2
// starts with l.ISOPrefix and "-" (case-insensitive; no check when the prefix
// is empty), or one of l.Include, whatever its tags. l.Parent is a geometric
// rule and is left to the caller. A zero Level matches nothing.
func MatchLevel(r *osm.Relation, l countries.Level) bool {
	if len(l.AdminLevels) == 0 {
		return false
	}
	for _, id := range l.Include {
		if int64(r.ID) == id {
			return true
		}
	}
	t := r.Tags
	if t.Find("boundary") != "administrative" {
		return false
	}
	level, in := t.Find("admin_level"), false
	for _, n := range l.AdminLevels {
		if level == strconv.Itoa(n) {
			in = true
			break
		}
	}
	if !in {
		return false
	}
	return l.ISOPrefix == "" || strings.HasPrefix(strings.ToUpper(t.Find("ISO3166-2")), strings.ToUpper(l.ISOPrefix)+"-")
}

// NamesFor returns the English and local names from OSM tags, before
// stripping: English is the first of imp.English that is not empty and
// contains no letter of imp.EnglishNot; local is the first of imp.Local that
// is not empty, where plain "name" counts only if it contains a letter of
// imp.LocalScripts (any name when LocalScripts is empty). Values are trimmed.
func NamesFor(t osm.Tags, imp countries.Import) (en, local string) {
	for _, k := range imp.English {
		if v := strings.TrimSpace(t.Find(k)); v != "" && !inScripts(v, imp.EnglishNot) {
			en = v
			break
		}
	}
	for _, k := range imp.Local {
		v := strings.TrimSpace(t.Find(k))
		if v == "" || k == "name" && len(imp.LocalScripts) > 0 && !inScripts(v, imp.LocalScripts) {
			continue
		}
		local = v
		break
	}
	return en, local
}

// inScripts reports whether s holds a letter of one of the Unicode scripts
// named (unicode.Scripts keys: "Thai", "Latin", "Han", …).
func inScripts(s string, scripts []string) bool {
	for _, name := range scripts {
		table := unicode.Scripts[name]
		if table == nil {
			continue
		}
		for _, r := range s {
			if unicode.Is(table, r) {
				return true
			}
		}
	}
	return false
}

// StripName removes l.Strip's words from the English and local names: every
// entry once, in order, suffixes first, then prefixes, trimming spaces before
// and after each.
func StripName(en, local string, l countries.Level) (string, string) {
	return strip(en, l.Strip.EnSuffix, l.Strip.EnPrefix), strip(local, l.Strip.LocalSuffix, l.Strip.LocalPrefix)
}

func strip(s string, suffixes, prefixes []string) string {
	s = strings.TrimSpace(s)
	for _, x := range suffixes {
		s = strings.TrimSpace(strings.TrimSuffix(s, x))
	}
	for _, x := range prefixes {
		s = strings.TrimSpace(strings.TrimPrefix(s, x))
	}
	return s
}

// DistinctLocal returns local, or "" when it equals en: a second name that
// only repeats the first is stored as NULL (Singapore, the Philippines,
// Vietnamese communes that have only name).
func DistinctLocal(en, local string) string {
	if local == en {
		return ""
	}
	return local
}

// KindWord returns the display word for a relation of level l from its names
// before stripping: the word of the first l.KindByAffix entry that the
// English or the local name starts or ends with, else l.Kind. Longer affixes
// are tried first, equal lengths in byte order, so the result does not
// depend on map order. Between two cased letters (Latin, Cyrillic, …) an
// affix must end at a word boundary ("State" does not match "Statesville");
// in scripts without case ("市", "시") it may touch the name.
func KindWord(en, local string, l countries.Level) string {
	affixes := make([]string, 0, len(l.KindByAffix))
	for a := range l.KindByAffix {
		if a != "" {
			affixes = append(affixes, a)
		}
	}
	sort.Slice(affixes, func(i, j int) bool {
		if ni, nj := utf8.RuneCountInString(affixes[i]), utf8.RuneCountInString(affixes[j]); ni != nj {
			return ni > nj
		}
		return affixes[i] < affixes[j]
	})
	en, local = strings.TrimSpace(en), strings.TrimSpace(local)
	for _, a := range affixes {
		if hasAffix(en, a) || hasAffix(local, a) {
			return l.KindByAffix[a]
		}
	}
	return l.Kind
}

// KindName is the value for regions.kind_name: KindWord, or "" when that is
// the structural kind ("province", "district") anyway, so rows whose word
// follows from kind keep NULL.
func KindName(en, local string, l countries.Level, kind string) string {
	if w := KindWord(en, local, l); w != kind {
		return w
	}
	return ""
}

// hasAffix reports whether name starts or ends with a, at a word boundary
// where both letters at the joint are cased.
func hasAffix(name, a string) bool {
	if name == a {
		return true
	}
	if rest, ok := strings.CutPrefix(name, a); ok {
		last, _ := utf8.DecodeLastRuneInString(a)
		next, _ := utf8.DecodeRuneInString(rest)
		if !(cased(last) && cased(next)) {
			return true
		}
	}
	if rest, ok := strings.CutSuffix(name, a); ok {
		first, _ := utf8.DecodeRuneInString(a)
		prev, _ := utf8.DecodeLastRuneInString(rest)
		if !(cased(first) && cased(prev)) {
			return true
		}
	}
	return false
}

// cased reports whether r is a letter of a script with upper and lower case.
func cased(r rune) bool {
	return unicode.IsLetter(r) && (unicode.ToUpper(r) != unicode.ToLower(r) || unicode.IsUpper(r) || unicode.IsLower(r))
}

// Slug turns an English name into a URL slug: accents and marks are dropped
// (NFD), đ becomes d (and a few letters NFD does not split, ø, ł, ß, æ), then
// lower-case ASCII letters and digits are separated by single hyphens. Letters
// of other scripts separate like spaces. ASCII input gives what the slug
// without folding gave, so Thailand's stored slugs stay ("Cần Thơ" →
// "can-tho", "Chon Buri" → "chon-buri"). cmd/countryscan uses the same
// function, so scan and import cannot drift.
func Slug(s string) string {
	s = strings.NewReplacer("đ", "d", "Đ", "D", "ø", "o", "Ø", "O", "ł", "l", "Ł", "L", "ß", "ss", "æ", "ae", "Æ", "AE").Replace(s)
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		case unicode.Is(unicode.Mn, r):
			// a combining mark stays with its letter
		default:
			dash = true
		}
	}
	return b.String()
}

// PlaceKindsFor returns the place=* values imported for a country:
// imp.PlaceKinds, else PlaceKinds.
func PlaceKindsFor(imp countries.Import) map[string]bool {
	if len(imp.PlaceKinds) == 0 {
		return PlaceKinds
	}
	out := make(map[string]bool, len(imp.PlaceKinds))
	for _, k := range imp.PlaceKinds {
		out[k] = true
	}
	return out
}

// Named is one relation's names after the rules of its level.
type Named struct {
	RawEn, RawLocal string // NamesFor: before stripping (what ResolveSlugs restores)
	En, Local       string // StripName, then DistinctLocal
	Kind            string // KindWord of the raw names
}

// LevelNames applies the name rules of level l in their order: NamesFor,
// KindWord on the names as OSM has them, StripName, DistinctLocal.
func LevelNames(t osm.Tags, imp countries.Import, l countries.Level) Named {
	n := Named{}
	n.RawEn, n.RawLocal = NamesFor(t, imp)
	n.Kind = KindWord(n.RawEn, n.RawLocal, l)
	n.En, n.Local = StripName(n.RawEn, n.RawLocal, l)
	n.Local = DistinctLocal(n.En, n.Local)
	return n
}

// Sibling is one of the regions ResolveSlugs compares: the provinces of one
// country, or the districts of one province.
type Sibling struct {
	OSMID  int64
	Names  Named
	Stored string // the slug already stored for this relation, "" if new; it never changes
	Slug   string // set by ResolveSlugs
}

// ResolveSlugs gives every sibling its slug and settles duplicates
// (decision 7 of docs/COUNTRY_IMPORT_STAGE2_BRIEF.md). full builds a slug from
// its name part: for provinces Slug(en) plus the country suffix where the
// settings want one, for districts the province's stored slug, "-" and
// Slug(en). A relation with a stored slug keeps it, renamed or not.
//
//  1. Every sibling gets full(Slug(Names.En)).
//  2. Where siblings share a slug (a stored one counts as theirs), all of them
//     keep their affixes: Names.En and Names.Local go back to the raw names
//     (Local still NULL when equal), and the new ones get full(Slug(raw En)).
//  3. New siblings that still share a slug get the OSM id appended to the
//     name part: full(Slug(En) + "-" + id).
//
// An empty name part is left for the caller to refuse.
func ResolveSlugs(sibs []Sibling, full func(base string) string) {
	slugOf := func(s *Sibling, base string) {
		if s.Stored != "" {
			s.Slug = s.Stored
		} else {
			s.Slug = full(base)
		}
	}
	shared := func() map[string]int {
		n := map[string]int{}
		for _, s := range sibs {
			n[s.Slug]++
		}
		return n
	}
	for i := range sibs {
		slugOf(&sibs[i], Slug(sibs[i].Names.En))
	}
	n := shared()
	for i := range sibs {
		s := &sibs[i]
		if n[s.Slug] < 2 {
			continue
		}
		s.Names.En = s.Names.RawEn
		s.Names.Local = DistinctLocal(s.Names.RawEn, s.Names.RawLocal)
		slugOf(s, Slug(s.Names.En))
	}
	n = shared()
	for i := range sibs {
		s := &sibs[i]
		if n[s.Slug] < 2 || s.Stored != "" {
			continue
		}
		slugOf(s, fmt.Sprintf("%s-%d", Slug(s.Names.En), s.OSMID))
	}
}
