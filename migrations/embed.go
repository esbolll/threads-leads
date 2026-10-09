// Package migrations embeds the SQL migrations so the binary stays self-contained.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
