package web

import (
	"html/template"
	"regexp"
	"strings"
)

// markdown renders the small Markdown subset used by docs/PRINCIPLES.md:
// paragraphs, a top-level heading (dropped, the page has its own), numbered
// lists, **bold** and [links](url). Everything else is escaped text.
func markdown(src string) template.HTML {
	var out strings.Builder
	var para []string
	inList := false
	flush := func() {
		if len(para) > 0 {
			out.WriteString("<p>" + inline(strings.Join(para, " ")) + "</p>\n")
			para = nil
		}
	}
	closeList := func() {
		if inList {
			out.WriteString("</ol>\n")
			inList = false
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			flush()
			closeList()
		case strings.HasPrefix(line, "# "):
			flush()
			closeList()
		case strings.HasPrefix(line, "## "):
			flush()
			closeList()
			out.WriteString("<h2>" + inline(line[3:]) + "</h2>\n")
		case listItem.MatchString(line):
			flush()
			if !inList {
				out.WriteString("<ol>\n")
				inList = true
			}
			out.WriteString("<li>" + inline(listItem.ReplaceAllString(line, "")) + "</li>\n")
		default:
			para = append(para, line)
		}
	}
	flush()
	closeList()
	return template.HTML(out.String())
}

var (
	listItem = regexp.MustCompile(`^\d+\.\s+`)
	bold     = regexp.MustCompile(`\*\*(.+?)\*\*`)
	link     = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
)

func inline(s string) string {
	s = template.HTMLEscapeString(s)
	s = bold.ReplaceAllString(s, "<strong>$1</strong>")
	return link.ReplaceAllStringFunc(s, func(m string) string {
		p := link.FindStringSubmatch(m)
		href := p[2]
		if !strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "https://") && !strings.HasPrefix(href, "http://") {
			return p[1]
		}
		return `<a href="` + href + `">` + p[1] + `</a>`
	})
}
