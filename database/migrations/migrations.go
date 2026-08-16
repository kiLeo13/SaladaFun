// Package migrations embeds Salada's ordered SQL history into the manual
// migration executable.
package migrations

import "embed"

// Files contains every Goose migration shipped with the executable.
//
//go:embed *.sql
var Files embed.FS
