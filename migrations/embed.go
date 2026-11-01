// Package migrations embeds the SQL migration files so that the backend
// binary can apply them at start-up without access to the source tree.
package migrations

import "embed"

// FS holds every *.sql file in this directory. Files are applied in lexical
// order of their names; never edit a file that has been applied, add a new one.
//
//go:embed *.sql
var FS embed.FS
