package migrations

import "embed"

// Files contains SQL migrations bundled into the binary.
//
//go:embed *.sql
var Files embed.FS
