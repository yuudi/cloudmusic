package migrations

import "embed"

// Files contains the versioned SQL migrations for the explicit migration command.
//
//go:embed *.sql
var Files embed.FS
