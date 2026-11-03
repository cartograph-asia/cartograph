package osmfile

import (
	"fmt"
	"slices"
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
// is empty), or one of l.Include, whatever its tags; never one of l.Exclude
// (Excluded tells those apart). l.Parent is a geometric rule and is left to
// the caller. A zero Level matches nothing.
func MatchLevel(r *osm.Relation, l countries.Level) bool {
	return !slices.Contains(l.Exclude, int64(r.ID)) && matchRule(r, l)
}

// Excluded reports whether r would belong to level l by its tags but is one
// of l.Exclude, which the dry run reports.
func Excluded(r *osm.Relation, l countries.Level) bool {
	return slices.Contains(l.Exclude, int64(r.ID)) && matchRule(r, l)
}

// matchRule is MatchLevel without l.Exclude.
func matchRule(r *osm.Relation, l countries.Level) bool {
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

// LatinLocal reports whether imp's local names may be written in Latin
// script (LocalScripts names "Latin", or is empty and takes any name): only
// then is a local name equal to the English one the same name, and the
// importer stores it as NULL (DistinctLocal). Where the local script is
// another (Thai, Lao, Han), such a name is a Latin name:xx tag, kept as OSM
// has it, as Thailand's import always did; so the rule never changes a Thai
// name.
func LatinLocal(imp countries.Import) bool {
	return len(imp.LocalScripts) == 0 || slices.Contains(imp.LocalScripts, "Latin")
}

// PlaceNames returns a place node's English and local names: NamesFor, then
// DistinctLocal where LatinLocal holds. Place names are not stripped.
func PlaceNames(t osm.Tags, imp countries.Import) (en, local string) {
	en, local = NamesFor(t, imp)
	if LatinLocal(imp) {
		local = DistinctLocal(en, local)
	}
	return en, local
}

// KindWord returns the display word for a relation of level l from its names
// before stripping: the word of the first l.KindByAffix entry that the
// English or the local name starts or ends with, else l.Kind. Longer affixes
// are tried first, equal lengths in byte order, so the result does not
// depend on map order. Between two cased letters (Latin, Cyrillic, …) an
// affix must end at a word boundary ("State" does not match "Statesville");
// in scripts without case ("市", "시") it may touch the name.
func KindWord(en, local string, l countries.Level) string {
	if w, ok := kindByAffix(l, en, local); ok {
		return w
	}
	return l.Kind
}

// KindWordOf is KindWord for a relation with the tags t: when neither name
// carries an affix, the values of l.OfficialName are tried in their order,
// each with every affix, longer first, so "Surabaya" with official_name
// "Kota Surabaya" is a city.
func KindWordOf(t osm.Tags, en, local string, l countries.Level) string {
	if w, ok := kindByAffix(l, en, local); ok {
		return w
	}
	for _, k := range l.OfficialName {
		if w, ok := kindByAffix(l, t.Find(k)); ok {
			return w
		}
	}
	return l.Kind
}

// kindByAffix returns the word of the first l.KindByAffix entry, longer
// affixes first, that one of names starts or ends with.
func kindByAffix(l countries.Level, names ...string) (string, bool) {
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
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}
	for _, a := range affixes {
		for _, name := range names {
			if hasAffix(name, a) {
				return l.KindByAffix[a], true
			}
		}
	}
	return "", false
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

// KindName is the value for regions.kind_name of a relation whose names
// LevelNames gave: n.Kind (KindWordOf), or "" when that is kind anyway.
func (n Named) KindName(kind string) string {
	if n.Kind != kind {
		return n.Kind
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

// IsApostrophe reports whether r is a character Slug drops instead of
// treating it as a separator: the apostrophe ' (U+0027), the right single
// quotation mark ’ (U+2019), the modifier letters ʼ (U+02BC) and ʻ (U+02BB),
// the grave accent ` (U+0060) and the acute accent ´ (U+00B4), which OSM
// names use as an apostrophe or as a glottal mark.
func IsApostrophe(r rune) bool {
	switch r {
	case '\'', '’', 'ʼ', 'ʻ', '`', '´':
		return true
	}
	return false
}

// Slug turns an English name into a URL slug: accents and marks are dropped
// (NFD), đ becomes d (and a few letters NFD does not split, ø, ł, ß, æ),
// apostrophes (IsApostrophe) are dropped too, so "P'yŏngyang" is "pyongyang"
// and "Xi'an" is "xian"; then lower-case ASCII letters and digits are
// separated by single hyphens. Letters of other scripts and all other
// punctuation separate like spaces. A name without apostrophes gives what the
// slug without folding gave, so Thailand's stored slugs stay ("Cần Thơ" →
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
		case unicode.Is(unicode.Mn, r), IsApostrophe(r):
			// a combining mark stays with its letter; an apostrophe joins
			// the letters around it
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
	En, Local       string // StripName, then DistinctLocal (where LatinLocal)
	Kind            string // KindWordOf: by the raw names, else by Level.OfficialName
	// OfficialEn and OfficialLocal are the forms with an affix that
	// Level.OfficialName gives where the raw name carries none ("Kota
	// Bogor" for "Bogor"; officialForms), else "". ResolveSlugs restores
	// them in place of the raw names.
	OfficialEn, OfficialLocal string
	keepEqual                 bool // LatinLocal was false: a local name equal to En stays
}

// local is DistinctLocal unless the settings keep equal names.
func (n Named) local(en, local string) string {
	if n.keepEqual {
		return local
	}
	return DistinctLocal(en, local)
}

// LevelNames applies the name rules of level l in their order: NamesFor,
// KindWordOf on the names as OSM has them (then on l.OfficialName),
// StripName, the official forms with their affixes (officialForms),
// DistinctLocal (only where LatinLocal holds).
func LevelNames(t osm.Tags, imp countries.Import, l countries.Level) Named {
	n := Named{keepEqual: !LatinLocal(imp)}
	n.RawEn, n.RawLocal = NamesFor(t, imp)
	n.Kind = KindWordOf(t, n.RawEn, n.RawLocal, l)
	n.En, n.Local = StripName(n.RawEn, n.RawLocal, l)
	n.OfficialEn, n.OfficialLocal = officialForms(t, imp, l, n)
	n.Local = n.local(n.En, n.Local)
	return n
}

// officialForms returns the English and local forms with an affix that
// l.OfficialName gives for n, whose En and Local are the stripped names
// (Local before DistinctLocal). A language has one only where its raw name
// carries no affix (stripping left it as it was): the value of the first
// listed tag read for that language (officialFor) that is not the name
// itself but turns into it by l.Strip ("Kota Bogor" strips to "Bogor"). An
// English value holds no letter of imp.EnglishNot; a local value of plain
// "official_name" counts only if written in imp.LocalScripts, as plain
// "name" does in NamesFor.
func officialForms(t osm.Tags, imp countries.Import, l countries.Level, n Named) (en, local string) {
	if len(l.OfficialName) == 0 {
		return "", ""
	}
	find := func(name, raw string, english bool, suffixes, prefixes []string) string {
		if name == "" || name != strings.TrimSpace(raw) {
			return ""
		}
		for _, k := range l.OfficialName {
			if !officialFor(k, imp, english) {
				continue
			}
			v := strings.TrimSpace(t.Find(k))
			switch {
			case v == "" || v == name || strip(v, suffixes, prefixes) != name:
				continue
			case english && inScripts(v, imp.EnglishNot):
				continue
			case !english && k == "official_name" && len(imp.LocalScripts) > 0 && !inScripts(v, imp.LocalScripts):
				continue
			}
			return v
		}
		return ""
	}
	return find(n.En, n.RawEn, true, l.Strip.EnSuffix, l.Strip.EnPrefix),
		find(n.Local, n.RawLocal, false, l.Strip.LocalSuffix, l.Strip.LocalPrefix)
}

// officialFor reports whether the tag k of Level.OfficialName is read for
// the English name (english) or for the local one: always, unless k without
// its "official_" prefix is a tag of the other language's list only
// (imp.English, imp.Local): "official_name:en" is English only,
// "official_name:id" Indonesian only where Local holds "name:id",
// "official_name" both where both lists hold "name".
func officialFor(k string, imp countries.Import, english bool) bool {
	plain, ok := strings.CutPrefix(k, "official_")
	if !ok {
		return true
	}
	own, other := imp.English, imp.Local
	if !english {
		own, other = other, own
	}
	return slices.Contains(own, plain) || !slices.Contains(other, plain)
}

// Sibling is one of the regions ResolveSlugs compares: the provinces of one
// country, or the districts of one province.
type Sibling struct {
	OSMID int64
	Names Named
	// Stored is the slug already stored for this relation, "" if new; it
	// never changes. StoredEn and StoredLocal are the names stored with it
	// ("" for NULL): they tell whether the stored row carries its affixes.
	Stored                string
	StoredEn, StoredLocal string
	Slug                  string // set by ResolveSlugs
}

// ResolveSlugs gives every sibling its slug and settles duplicates
// (decision 7 of docs/COUNTRY_IMPORT_STAGE2_BRIEF.md). full builds a slug from
// its name part: for provinces Slug(en) plus the country suffix where the
// settings want one, for districts the province's stored slug, "-" and
// Slug(en).
//
// Collisions are decided on what a fresh import would give, so that a
// second run decides as the first did, whatever is stored by then:
//
//  1. Every sibling gets full(Slug(Names.En)).
//  2. Where siblings share a slug, all of them keep their affixes: Names.En
//     and Names.Local go back to the raw names, or to the official forms
//     where a raw name carries no affix (Named.OfficialEn, OfficialLocal;
//     Local still NULL when equal), and they get full(Slug(that En)).
//  3. Siblings that still share a slug get the OSM id appended to the name
//     part: full(Slug(En) + "-" + id).
//
// Then the stored rows are laid over it. A relation with a stored slug keeps
// it, renamed or not, and keeps its stored affix state: when its stored
// English and local names are the stripped (or the raw) form of its names
// now, it gets that form again, whatever the collisions above say; only a
// relation renamed in OSM takes the form of step 2. A new sibling whose slug
// is a stored sibling's (the stored one was renamed, or came first and
// alone) gives way: it keeps its affixes, and when that is still taken it
// gets its OSM id appended.
//
// An empty name part is left for the caller to refuse.
func ResolveSlugs(sibs []Sibling, full func(base string) string) {
	// fresh[i] is what a fresh import gives sibling i.
	type result struct {
		slug    string
		affixed bool
	}
	fresh := make([]result, len(sibs))
	shared := func() map[string]int {
		n := map[string]int{}
		for _, r := range fresh {
			n[r.slug]++
		}
		return n
	}
	for i := range sibs {
		fresh[i] = result{slug: full(Slug(sibs[i].Names.En))}
	}
	n := shared()
	for i := range sibs {
		if n[fresh[i].slug] >= 2 {
			fresh[i] = result{slug: full(Slug(sibs[i].Names.affixed().En)), affixed: true}
		}
	}
	n = shared()
	for i := range sibs {
		if n[fresh[i].slug] >= 2 {
			name := sibs[i].Names.En
			if fresh[i].affixed {
				name = sibs[i].Names.affixed().En
			}
			fresh[i].slug = full(fmt.Sprintf("%s-%d", Slug(name), sibs[i].OSMID))
		}
	}

	taken := map[string]bool{}
	for i := range sibs {
		s := &sibs[i]
		if s.Stored == "" {
			continue
		}
		s.Slug = s.Stored
		taken[s.Stored] = true
		affixed := fresh[i].affixed
		if st, known := s.storedAffixed(); known {
			affixed = st
		}
		if affixed {
			s.Names = s.Names.affixed()
		}
	}
	for i := range sibs {
		s := &sibs[i]
		if s.Stored != "" {
			continue
		}
		affixed, slug := fresh[i].affixed, fresh[i].slug
		if taken[slug] && !affixed {
			affixed, slug = true, full(Slug(s.Names.affixed().En))
		}
		if taken[slug] {
			slug = full(fmt.Sprintf("%s-%d", Slug(s.Names.affixed().En), s.OSMID))
		}
		if affixed {
			s.Names = s.Names.affixed()
		}
		s.Slug = slug
		taken[slug] = true
	}
}

// affixed returns n with the names that carry their affixes in place of the
// stripped ones: the raw names, or the official form where a raw name
// carries none (OfficialEn, OfficialLocal; Level.OfficialName).
func (n Named) affixed() Named {
	en, local := n.RawEn, n.RawLocal
	if n.OfficialEn != "" {
		en = n.OfficialEn
	}
	if n.OfficialLocal != "" {
		local = n.OfficialLocal
	}
	n.En = en
	n.Local = n.local(en, local)
	return n
}

// storedAffixed tells from the stored names whether the stored row carries
// its affixes: known is false when the stored names are neither form of the
// names now (the relation was renamed in OSM, or nothing is stored). Both
// names are compared first, then the English one alone (a local name that
// changed on its own).
func (s Sibling) storedAffixed() (affixed, known bool) {
	if s.Stored == "" {
		return false, false
	}
	stripped, raw := s.Names, s.Names.affixed()
	switch {
	case s.StoredEn == stripped.En && s.StoredLocal == stripped.Local:
		return false, true
	case s.StoredEn == raw.En && s.StoredLocal == raw.Local:
		return true, true
	case s.StoredEn == stripped.En:
		return false, true
	case s.StoredEn == raw.En:
		return true, true
	}
	return false, false
}
