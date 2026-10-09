// Package migrations embeds the goose SQL migrations.
package migrations

import "embed"

// FS holds the goose SQL migration files, in the root of the file system.
//
//go:embed *.sql
var FS embed.FS
