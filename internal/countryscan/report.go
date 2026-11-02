package countryscan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// num formats n with thousands separators.
func num(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

func pct(a, b int) string {
	if b == 0 {
		return "–"
	}
	return fmt.Sprintf("%.0f %%", 100*float64(a)/float64(b))
}

// mdReplacer makes text from OpenStreetMap tags harmless in a table cell, a
// bullet and a code span of a Markdown report that GitHub renders.
var mdReplacer = strings.NewReplacer(
	"\r\n", " ", "\n", " ", "\r", " ", "\t", " ",
	"|", `\|`,
	"<", "&lt;", ">", "&gt;",
	"[", `\[`, "]", `\]`,
	"`", "’", // a backtick cannot be escaped inside a code span
)

// mdText escapes s for a table cell, a bullet or a code span. Underscores and
// asterisks are left alone: they only emphasise, they cannot break out.
func mdText(s string) string { return mdReplacer.Replace(s) }

// mdTexts applies mdText to each element.
func mdTexts(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = mdText(s)
	}
	return out
}

// counts renders a map as "key n, key n" by count, then key.
func counts(m map[string]int, limit int) string {
	type kv struct {
		k string
		n int
	}
	var l []kv
	for k, n := range m {
		l = append(l, kv{k, n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].k < l[j].k
	})
	if limit > 0 && len(l) > limit {
		l = l[:limit]
	}
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = fmt.Sprintf("`%s` %s", mdText(x.k), num(x.n))
	}
	return strings.Join(out, ", ")
}

func affixList(as []Affix) string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = fmt.Sprintf("“%s” %d", mdText(a.Text), a.Count)
	}
	return strings.Join(out, ", ")
}

func or(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

// Report renders the findings as Markdown. It holds nothing that changes
// between runs on the same extract: no date, no duration.
func (r *Result) Report() string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	c := r.Country

	w("# Country scan: %s\n\n", c.NameEn)
	w("Written by `go run ./cmd/countryscan -country %s` from `%s`; stage 1 of `docs/COUNTRY_IMPORT_BRIEF.md`. ", c.Code, r.Extract)
	w("Nothing was imported. The same extract gives the same report; run it again after an extract update and compare.\n\n")

	r.writeSummary(&b)

	w("## Outline and land\n\n")
	w("- Outline: %s; %s km² in %d polygons.\n", r.Shape.Source, num(int(r.Shape.AreaKm2)), r.Shape.Polygons)
	w("- Cells at resolution %d with their centre inside the outline: %s; %s of them are land (", r.Res, num(r.CountryCells), num(r.LandCells))
	if r.Water {
		w("not in an OSM water polygon).\n")
	} else {
		w("no water polygons were given, so sea inside the outline counts as land).\n")
	}
	w("- Land plots (resolution 8)%s: **%s**, about %s km².\n\n", map[bool]string{true: ", estimated", false: ""}[r.Size.Estimated],
		num(r.Size.LandPlots), num(int(float64(r.Size.LandPlots)*PlotKm2)))

	w("## Administrative levels\n\n")
	w("`boundary=administrative` relations by `admin_level`. *In the extract* counts every relation in the file, neighbouring countries cut by the extract included. ")
	w("*In scope* is what the other columns count: for a level whose geometry was built (*country*), the relations of this country (their ISO 3166-2 starts with %s-, or they have none and their centre lies in the outline); ", strings.ToUpper(c.Code))
	w("for a level with more than %s relations (*extract*), all relations of the extract except those with another country's ISO 3166-2. ", num(DefaultGeometryCap))
	w("Cover, gaps and overlap are shares of the country's land plots (resolution %d) that lie in one, in no and in two or more relations of the level.\n\n", r.Res)
	w("| Level | In extract | In scope | Scope | ISO 3166-2 | name:en | int_name | Latin name | Cover | Gaps | Overlap | Rings not closed |\n")
	w("|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, l := range r.Levels {
		cov, gap, ov, sk := "–", "–", "–", "–"
		if l.Geometry {
			cov, gap, ov, sk = fmt.Sprintf("%.1f %%", l.CoverPct), fmt.Sprintf("%.1f %%", l.GapPct), fmt.Sprintf("%.2f %%", l.OverlapPct), num(l.Skipped)
		}
		w("| %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n", l.Level, num(l.Extract), num(l.N), l.Scope,
			num(l.ISOHere), num(l.NameEn), num(l.IntName), num(l.LatinName), cov, gap, ov, sk)
	}
	w("\n")
	for _, l := range r.Levels {
		r.writeLevel(&b, l)
	}

	r.writeProposal(&b)
	r.writeNames(&b)
	r.writePlaces(&b)

	w("## Size\n\n")
	w("- Land plots: %s%s. `plot_numbers` rows after the import: at most that many (a plot already numbered by a neighbouring country is not numbered again; `overlaps.md` counts those).\n",
		num(r.Size.LandPlots), map[bool]string{true: " (estimated from resolution " + strconv.Itoa(r.Res) + ")", false: ""}[r.Size.Estimated])
	w("- Stage 3, scaled linearly from the Thailand measurements (%s land plots, %d s, about %d MiB): about %s s and %s MiB. A rough figure; it was not measured for this country.\n\n",
		num(ThailandPlots), ThailandSeconds, ThailandMiB, num(r.Size.Seconds), num(r.Size.MiB))

	r.writeConfig(&b)
	return b.String()
}

func (r *Result) writeSummary(b *strings.Builder) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	w("## Summary\n\n")
	if r.Province.Level > 0 {
		w("- **Provinces:** %s.\n", r.Province.Reason)
	} else {
		w("- **Provinces:** %s.\n", r.Province.Reason)
	}
	if len(r.Province.Candidates) > 0 {
		w("  Levels that tile the land well enough (cover ≥ %.0f %%, overlap ≤ %.0f %%): %s.\n", minCoverPct, maxOverlapPct, levelList(r.Province.Candidates))
	}
	w("- **Districts:** %s.", r.District.Reason)
	if r.District.Level > 0 {
		w(" Per province: %d to %d, median %d; %d districts have their centre in no province.", r.District.ChildMin, r.District.ChildMax, r.District.ChildMedian, r.District.Orphans)
	}
	w("\n- **Land plots:** %s%s.\n", num(r.Size.LandPlots), map[bool]string{true: " (estimated)", false: ""}[r.Size.Estimated])
	for _, n := range r.Names {
		if n.NoEnglish > 0 || n.EmptySlug > 0 || len(n.Duplicates) > 0 || n.AccentLoss > 0 {
			w("- **Names (%ss):** %d without an English-capable name, %d with an empty slug, %d slugs used twice, %d names whose slug loses letters without accent folding.\n",
				n.Role, n.NoEnglish, n.EmptySlug, len(n.Duplicates), n.AccentLoss)
		}
	}
	for _, l := range r.Levels {
		if l.CapExceeded {
			w("- Level %d has %s relations, more than the %s the scan builds geometry for: counted and read for names only.\n", l.Level, num(l.Extract), num(DefaultGeometryCap))
		}
	}
	if l := r.levelOf(r.Province.Level); l != nil && (l.Skipped > 0 || l.NoLand > 0) {
		w("- Provinces whose rings do not close in the extract: %d; without a land plot: %d.\n", l.Skipped, l.NoLand)
	}
	w("\n")
}

func levelList(ls []int) string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = strconv.Itoa(l)
	}
	return strings.Join(out, ", ")
}

func (r *Result) writeLevel(b *strings.Builder, l *Level) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	w("### Level %d\n\n", l.Level)
	if l.Level == MinLevel {
		w("Countries in the extract: %s.\n\n", join(mdTexts(l.Examples)))
		return
	}
	w("- %s in the extract, %s in scope (%s).", relations(l.Extract), num(l.N), l.Scope)
	if l.Geometry {
		w(" Left out: %s with another country's ISO 3166-2, %s with their centre outside the country (or not assembled and without ISO code).", num(l.OtherISO), num(l.Outside))
	} else {
		w(" Left out: %s with another country's ISO 3166-2.", num(l.OtherISO))
	}
	w("\n")
	if l.N == 0 {
		w("\n")
		return
	}
	w("- ISO 3166-2 prefixes: %s; with this country's prefix: %s of %s.\n", or(counts(l.Prefixes, 6), "none"), num(l.ISOHere), num(l.N))
	w("- Names: name %s, name:en %s, int_name %s, Latin name %s, any English-capable name %s (of %s).\n",
		num(l.HasName), num(l.NameEn), num(l.IntName), num(l.LatinName), num(l.English), num(l.N))
	w("- Script of `name`: %s.\n", or(counts(l.Scripts, 5), "no names"))
	w("- `name:*` tags: %s.\n", or(counts(l.NameKeys, 8), "none"))
	if l.Geometry {
		w("- Geometry: the relations add up to %s km² (%.1f %% of the outline); %s land plots lie in none of them, %s in two or more; %d relations hold no land plot; land plots per relation: %s to %s, median %s.\n",
			num(int(l.AreaKm2)), l.AreaPct, num(l.Gaps), num(l.Overlaps), l.NoLand, num(l.LandMin), num(l.LandMax), num(l.LandMedian))
		w("- Rings not closed in the extract (a member way is missing): %d in scope; %d of the level's relations could not be assembled at all.", l.Skipped, l.SkippedAny)
		if len(l.SkippedList) > 0 {
			w(" In scope: %s.", join(mdTexts(l.SkippedList)))
		}
		if l.FillErrors > 0 {
			w(" %d could not be filled with cells.", l.FillErrors)
		}
		w("\n")
	}
	w("- Examples: %s.\n\n", join(mdTexts(l.Examples)))
}

func (r *Result) writeProposal(b *strings.Builder) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	w("## Proposal: province and district levels\n\n")
	w("The rule is mechanical: the province level is the lowest level from 3 up with at least three relations, at least 80 %% of them with this country's ISO 3166-2 code, covering %.0f %% or more of the land plots with at most %.0f %% overlap; ", minCoverPct, maxISOOverlapPct)
	w("without ISO codes the same with at most %.0f %% overlap. The district level is the next level with more relations that covers %.0f %% or more with at most %.0f %% overlap. ", maxOverlapPct, minCoverPct, maxOverlapPct)
	w("It is a proposal; the table above is what to decide by.\n\n")
	w("- Provinces: %s.\n", r.Province.Reason)
	w("- Districts: %s.\n\n", r.District.Reason)
	if len(r.Province.Candidates) > 1 {
		w("Levels that qualify: %s. If the proposed levels are not the right ones (regions above provinces, cities that are their own level), the choice is Adrian's.\n\n", levelList(r.Province.Candidates))
	}
}

func (r *Result) writeNames(b *strings.Builder) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	w("## Names\n\n")
	if len(r.Names) == 0 {
		w("No province level was proposed, so no names were read.\n\n")
		return
	}
	w("English-capable name: the first of `name:en`, `int_name`, `name:%s-Latn`, `name` (when written in Latin letters). Local name: `name:%s`, else `name`. ", langSubtag(r.Country.Lang), langSubtag(r.Country.Lang))
	w("Prefixes and suffixes are the first and last words shared by at least 10 %% of the names (and three); for scripts written without spaces, the first and last one to six letters shared by at least 25 %%.\n\n")
	for _, n := range r.Names {
		w("### %ss (level %d, %s)\n\n", strings.ToUpper(n.Role[:1])+n.Role[1:], n.Level, relations(n.N))
		w("- English name from: %s; none for %d.\n", or(counts(n.EnglishKeys, 0), "–"), n.NoEnglish)
		w("- Local name from: %s; none for %d. English equals local for %d.\n", or(counts(n.LocalKeys, 0), "–"), n.NoLocal, n.SameAsLocal)
		w("- English prefixes: %s. Suffixes: %s.\n", or(affixList(n.Prefixes), "none"), or(affixList(n.Suffixes), "none"))
		w("- Local prefixes: %s. Suffixes: %s.\n", or(affixList(n.LocalPre), "none"), or(affixList(n.LocalSuf), "none"))
		w("- Slugs: %d names have no slug (no English-capable name, or nothing in it that a slug can hold); %d slugs are used more than once", n.EmptySlug, len(n.Duplicates))
		if len(n.Duplicates) > 0 {
			d := n.Duplicates
			if len(d) > 10 {
				d = d[:10]
			}
			w(": %s", join(mdTexts(d)))
		}
		w(".\n")
		if n.Role == "district" {
			w("  (A district's slug is built from its province's; the duplicates above are within one province.)\n")
		}
		if n.AccentLoss > 0 {
			w("- **Accents:** a slug without folding (as before stage 2) drops accented letters; `osmfile.Slug` folds them (NFD, đ → d), which changes the slug of %d names, e.g. %s.\n", n.AccentLoss, join(mdTexts(n.AccentEx)))
		}
		w("\n")
	}
	w("The slug of a new country's province carries the country as a suffix (`-%s`), decided on 2026-11-02, so it cannot collide with another country's.\n\n", r.Country.Slug)
}

func (r *Result) writePlaces(b *strings.Builder) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	w("## Places\n\n")
	w("`place=*` nodes inside the outline: %s. The importer reads `city`, `town`, `village`, `suburb`, `quarter` and `neighbourhood` that have an English-capable name.\n\n", num(r.Places.Nodes))
	w("| place | Nodes | Named | English-capable |\n|---|---:|---:|---:|\n")
	for i, k := range r.Places.Kinds {
		if i == 12 {
			break
		}
		w("| %s | %s | %s | %s |\n", mdText(k.Kind), num(k.Total), num(k.Named), num(k.Latin))
	}
	w("\n")
	if len(r.Places.Cities) == 0 {
		w("No `place=city` node with a population tag and an English-capable name, so there are no largest cities to look at.\n\n")
		return
	}
	w("Place nodes within %d km of the largest cities (by `population`), the input for `urban_kinds`:\n\n", CityRadiusKm)
	for _, c := range r.Places.Cities {
		var parts []string
		for i, k := range c.Kinds {
			if i == 6 {
				break
			}
			if k.Total > 0 {
				parts = append(parts, fmt.Sprintf("%s %s (%s English-capable)", mdText(k.Kind), num(k.Total), num(k.Latin)))
			}
		}
		w("- %s (population %s): %s.\n", mdText(c.Name), num(c.Population), or(join(parts), "no other place nodes"))
	}
	w("\n")
}

func (r *Result) writeConfig(b *strings.Builder) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	c := r.Country
	w("## Proposed configuration\n\n")
	w("A sketch of what stage 2 will hold for this country; the settings type comes with stage 2, so this is not code yet.\n\n")
	w("```go\n")
	w("// %s: from %s\n", c.NameEn, r.Extract)
	w("{Code: %q, Extract: %q,\n", c.Code, strings.TrimSuffix(r.Extract, "-latest.osm.pbf"))
	if r.Province.Level > 0 {
		l := r.levelOf(r.Province.Level)
		w("\tProvince: Rule{AdminLevel: %d, ISOPrefix: %q, Count: %d, Kind: \"province\"}, // kind word to decide: province, prefecture, state, region\n", r.Province.Level, strings.ToUpper(c.Code), l.N)
	} else {
		w("\tProvince: Rule{}, // no level found, see the proposal\n")
	}
	if r.District.Level > 0 {
		w("\tDistrict: Rule{AdminLevel: %d},\n", r.District.Level)
	} else {
		w("\tDistrict: Rule{}, // none\n")
	}
	w("\tEnglishName: []string{\"name:en\", \"int_name\", \"name:%s-Latn\", \"name\"}, LocalLang: %q,\n", langSubtag(c.Lang), c.Lang)
	for _, n := range r.Names {
		if n.N == 0 {
			continue
		}
		w("\tStrip%s: Strip{English: [2][]string{%s, %s}, Local: [2][]string{%s, %s}}, // prefixes, suffixes shared by at least half of the names\n",
			strings.ToUpper(n.Role[:1])+n.Role[1:], quoted(half(n.Prefixes, n.N)), quoted(half(n.Suffixes, n.N)), quoted(half(n.LocalPre, n.N)), quoted(half(n.LocalSuf, n.N)))
	}
	w("},\n")
	nm := r.Naming
	w("\n-- place_naming (a migration of its own, numbered by the lead)\n")
	w("insert into place_naming (country, urban_kinds, urban_radius_km, kinds, radius_km) values\n")
	w("  ('%s', '{%s}', %d, '%s', %d);\n", c.Code, strings.Join(nm.UrbanKinds, ","), nm.UrbanRadiusKm, nm.Kinds, nm.RadiusKm)
	w("```\n\n")
	w("`place_naming`: %s.\n", nm.Basis)
}

// half returns the affixes shared by at least half of n names.
func half(as []Affix, n int) []string {
	var out []string
	for _, a := range as {
		if 2*a.Count >= n {
			out = append(out, a.Text)
		}
	}
	return out
}

func quoted(ss []string) string {
	if len(ss) == 0 {
		return "nil"
	}
	out := make([]string, len(ss))
	for i, s := range ss {
		// A backtick would end the code block of the report.
		out[i] = strconv.Quote(strings.ReplaceAll(s, "`", "’"))
	}
	return "[]string{" + strings.Join(out, ", ") + "}"
}
