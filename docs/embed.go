// Package docs embeds the OpenAPI specification and the user guides so the
// server can serve them without reading from disk.
package docs

import "embed"

//go:embed openapi.yaml
var OpenAPIYAML []byte

//go:embed guides/*.md
var Guides embed.FS
