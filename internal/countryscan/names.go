package countryscan

import (
	"sort"
	"strings"
	"unicode"

	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// Script names, as printed in the report.
const (
	scriptLatin    = "Latin"
	scriptThai     = "Thai"
	scriptLao      = "Lao"
	scriptKhmer    = "Khmer"
	scriptMyanmar  = "Myanmar"
	scriptHan      = "Han"
	scriptKana     = "Kana"
	scriptHangul   = "Hangul"
	scriptCyrillic = "Cyrillic"
	scriptArabic   = "Arabic"
	scriptOther    = "other"
)

// scriptOf returns the script most of s's letters are written in, "" when s
// has no letters. Any kana makes a Han-and-kana name Kana (Japanese).
func scriptOf(s string) string {
	counts := map[string]int{}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		switch {
		case unicode.Is(unicode.Latin, r):
			counts[scriptLatin]++
		case unicode.Is(unicode.Thai, r):
			counts[scriptThai]++
		case unicode.Is(unicode.Lao, r):
			counts[scriptLao]++
		case unicode.Is(unicode.Khmer, r):
			counts[scriptKhmer]++
		case unicode.Is(unicode.Myanmar, r):
			counts[scriptMyanmar]++
		case unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			counts[scriptKana]++
		case unicode.Is(unicode.Han, r):
			counts[scriptHan]++
		case unicode.Is(unicode.Hangul, r):
			counts[scriptHangul]++
		case unicode.Is(unicode.Cyrillic, r):
			counts[scriptCyrillic]++
		case unicode.Is(unicode.Arabic, r):
			counts[scriptArabic]++
		default:
			counts[scriptOther]++
		}
	}
	if counts[scriptKana] > 0 {
		return scriptKana
	}
	best, bestN := "", 0
	for k, n := range counts {
		if n > bestN || (n == bestN && k < best) {
			best, bestN = k, n
		}
	}
	return best
}

// isLatin reports whether s has letters and all of them are Latin.
func isLatin(s string) bool {
	letters := false
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		letters = true
		if !unicode.Is(unicode.Latin, r) {
			return false
		}
	}
	return letters
}

// langSubtag is the primary language subtag of a BCP 47 tag ("zh-Hant" -> "zh").
func langSubtag(tag string) string {
	if i := strings.IndexByte(tag, '-'); i > 0 {
		return tag[:i]
	}
	return tag
}

// tagSet is the tags of one relation or node the scan reads.
type tagSet map[string]string

// englishName returns the English-capable name of t and the tag it came from:
// name:en, int_name, name:<lang>-Latn, else name when it is written in Latin
// letters. Empty when there is none.
func englishName(t tagSet, lang string) (name, key string) {
	for _, k := range []string{"name:en", "int_name", "name:" + lang + "-Latn", "name"} {
		v := strings.TrimSpace(t[k])
		if v == "" {
			continue
		}
		if k == "name" && !isLatin(v) {
			continue
		}
		return v, k
	}
	return "", ""
}

// localName returns the name in the country's language: name:<lang>, else
// name; empty if neither.
func localName(t tagSet, lang string) (name, key string) {
	for _, k := range []string{"name:" + lang, "name"} {
		if v := strings.TrimSpace(t[k]); v != "" {
			return v, k
		}
	}
	return "", ""
}

// plainSlug is the slug without accent folding, as osmfile.Slug made it
// before stage 2: lower-case ASCII letters and digits separated by single
// hyphens, so an accented letter is dropped ("Hà Nội" becomes "h-n-i").
// Apostrophes are dropped as in osmfile.Slug, so the two differ only where
// folding matters ("O'Brien" is "obrien" in both). It is kept only for the
// report's accent line, which counts the names whose slug folding changes;
// the scan's slugs come from osmfile.Slug, like the import's.
func plainSlug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		case osmfile.IsApostrophe(r):
			// dropped, as osmfile.Slug does
		default:
			dash = true
		}
	}
	return b.String()
}

// Affix is a leading or trailing word (or, for scripts written without
// spaces, a few leading or trailing letters) that many names share.
type Affix struct {
	Text  string
	Count int
}

// affixes finds the prefixes and suffixes that repeat across names: whole
// first and last words of names with spaces (shared by at least max(3, 10 %)
// of the names), and the first and last one to six letters of names in a
// script written without spaces (shared by at least max(3, 25 %), since
// every short run of letters repeats by chance). The most common come first,
// at most six per side; of several that are shared by the same names, only
// the longest is kept.
func affixes(names []string) (prefixes, suffixes []Affix) {
	wpre, wsuf, rpre, rsuf := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	total := 0
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		total++
		if words := strings.Fields(n); len(words) > 1 {
			wpre[words[0]]++
			wsuf[words[len(words)-1]]++
			continue
		}
		if isLatin(n) {
			continue
		}
		rs := []rune(n)
		for k := 1; k <= 6 && k < len(rs); k++ {
			rpre[string(rs[:k])]++
			rsuf[string(rs[len(rs)-k:])]++
		}
	}
	pick := func(words, runes map[string]int, suffix bool) []Affix {
		var out []Affix
		for k, c := range words {
			if c >= max(3, total/10) {
				out = append(out, Affix{k, c})
			}
		}
		for k, c := range runes {
			if c >= max(3, total/4) {
				out = append(out, Affix{k, c})
			}
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Count != out[j].Count {
				return out[i].Count > out[j].Count
			}
			if li, lj := len([]rune(out[i].Text)), len([]rune(out[j].Text)); li != lj {
				return li > lj
			}
			return out[i].Text < out[j].Text
		})
		var kept []Affix
		for _, a := range out {
			shadowed := false
			for _, k := range kept {
				if k.Count == a.Count && (suffix && strings.HasSuffix(k.Text, a.Text) || !suffix && strings.HasPrefix(k.Text, a.Text)) {
					shadowed = true
					break
				}
			}
			if !shadowed {
				kept = append(kept, a)
			}
		}
		if len(kept) > 6 {
			kept = kept[:6]
		}
		return kept
	}
	return pick(wpre, rpre, false), pick(wsuf, rsuf, true)
}
