// Package migrations embeds the versioned SQL schema migrations for each
// supported database dialect. Files are applied in lexical order and tracked
// in the schema_migrations table.
package migrations

import "embed"

//go:embed postgres/*.sql sqlite/*.sql
var FS embed.FS
