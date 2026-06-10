// Package migrations embeds the goose SQL migrations so they ship with the binary.
package migrations

import "embed"

// FS holds the embedded *.sql goose migrations.
//
//go:embed *.sql
var FS embed.FS
