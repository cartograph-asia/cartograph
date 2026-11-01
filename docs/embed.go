// Package docs embeds the Markdown behind the site's static pages, so the
// api binary serves them without access to the source tree: every file in
// pages/ and PRINCIPLES.md (the /principles page, kept where CLAUDE.md
// points to it).
package docs

import "embed"

// FS holds PRINCIPLES.md and pages/*.md.
//
//go:embed PRINCIPLES.md pages/*.md
var FS embed.FS
