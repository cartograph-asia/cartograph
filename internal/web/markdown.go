package web

import (
	"html/template"
	"regexp"
	"strings"
)

// mdDoc is a static page rendered from Markdown.
type mdDoc struct {
	Title   string        // text of the "# " heading
	Summary string        // first paragraph as plain text, for the meta description
	Body    template.HTML // everything after the title
}

// markdown renders the small Markdown subset of the pages in docs/: a "# "
// title (returned, not rendered: the page has its own h1), "## " and "### "
// headings, paragraphs (the first one is the lede), numbered and bulleted
// lists, pipe tables, **bold**, `code` and [links](url). Everything else is
// escaped text.
func markdown(src string) mdDoc {
	var doc mdDoc
	var out strings.Builder
	var para []string
	var table [][]string
	list := "" // "ol", "ul" or ""
	flush := func() {
		if len(para) > 0 {
			text := strings.Join(para, " ")
			if doc.Summary == "" {
				doc.Summary = plain(text)
				out.WriteString(`<p class="lede">` + inline(text) + "</p>\n")
			} else {
				out.WriteString("<p>" + inline(text) + "</p>\n")
			}
			para = nil
		}
		if list != "" {
			out.WriteString("</" + list + ">\n")
			list = ""
		}
		if table != nil {
			writeTable(&out, table)
			table = nil
		}
	}
	item := func(kind, text string) {
		if list != kind {
			flush()
			out.WriteString("<" + kind + ">\n")
			list = kind
		}
		out.WriteString("<li>" + inline(text) + "</li>\n")
	}
	for _, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "# "):
			flush()
			if doc.Title == "" {
				doc.Title = plain(line[2:])
			}
		case strings.HasPrefix(line, "## "):
			flush()
			out.WriteString("<h2>" + inline(line[3:]) + "</h2>\n")
		case strings.HasPrefix(line, "### "):
			flush()
			out.WriteString("<h3>" + inline(line[4:]) + "</h3>\n")
		case strings.HasPrefix(line, "|"):
			if table == nil {
				flush()
			}
			table = append(table, cells(line))
		case orderedItem.MatchString(line):
			item("ol", orderedItem.ReplaceAllString(line, ""))
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			item("ul", line[2:])
		default:
			para = append(para, line)
		}
	}
	flush()
	doc.Body = template.HTML(out.String())
	return doc
}

// cells splits a "| a | b |" table row.
func cells(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	parts := strings.Split(line, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// writeTable writes rows as a table. A row of dashes after the first row
// marks it as the header; an all-empty header is left out.
func writeTable(out *strings.Builder, rows [][]string) {
	out.WriteString("<table>\n")
	if len(rows) > 1 && isRule(rows[1]) {
		if strings.Join(rows[0], "") != "" {
			out.WriteString("<thead><tr>")
			for _, c := range rows[0] {
				out.WriteString("<th>" + inline(c) + "</th>")
			}
			out.WriteString("</tr></thead>\n")
		}
		rows = rows[2:]
	}
	out.WriteString("<tbody>\n")
	for _, r := range rows {
		out.WriteString("<tr>")
		for i, c := range r {
			tag := "td"
			if i == 0 {
				tag = "th"
			}
			out.WriteString("<" + tag + ">" + inline(c) + "</" + tag + ">")
		}
		out.WriteString("</tr>\n")
	}
	out.WriteString("</tbody>\n</table>\n")
}

func isRule(row []string) bool {
	for _, c := range row {
		if !tableRule.MatchString(c) {
			return false
		}
	}
	return true
}

var (
	orderedItem = regexp.MustCompile(`^\d+\.\s+`)
	tableRule   = regexp.MustCompile(`^:?-+:?$`)
	bold        = regexp.MustCompile(`\*\*(.+?)\*\*`)
	link        = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	code        = regexp.MustCompile("`([^`]+)`")
)

// inline renders bold, code and links in s; code spans are left as typed.
func inline(s string) string {
	var out strings.Builder
	last := 0
	for _, m := range code.FindAllStringSubmatchIndex(s, -1) {
		out.WriteString(spans(s[last:m[0]]))
		out.WriteString("<code>" + template.HTMLEscapeString(s[m[2]:m[3]]) + "</code>")
		last = m[1]
	}
	out.WriteString(spans(s[last:]))
	return out.String()
}

func spans(s string) string {
	s = template.HTMLEscapeString(s)
	s = bold.ReplaceAllString(s, "<strong>$1</strong>")
	return link.ReplaceAllStringFunc(s, func(m string) string {
		p := link.FindStringSubmatch(m)
		href := p[2]
		if !strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "https://") &&
			!strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "mailto:") {
			return p[1]
		}
		return `<a href="` + href + `">` + p[1] + `</a>`
	})
}

// plain is s without Markdown markup, for titles and descriptions.
func plain(s string) string {
	s = link.ReplaceAllString(s, "$1")
	s = bold.ReplaceAllString(s, "$1")
	return code.ReplaceAllString(s, "$1")
}
