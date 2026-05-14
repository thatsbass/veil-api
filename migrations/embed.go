package migrations

import "embed"

// FS embeds all migration SQL files so the binary is self-contained.
//
//go:embed *.sql
var FS embed.FS
