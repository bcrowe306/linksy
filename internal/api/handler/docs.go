package handler

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/fiber/v3"
	"go.yaml.in/yaml/v3"

	"github.com/bcrowe306/linksy/docs"
)

// HandleAPIDocs renders an interactive, modern Scalar documentation page
func (h *Handler) HandleAPIDocs(c fiber.Ctx) error {
	dark, bg := "true", "#0c0c0e"
	if c.Query("theme") == "light" { // only two fixed values ever reach the page
		dark, bg = "false", "#ffffff"
	}
	html := `<!doctype html>
<html>
  <head>
    <title>Lynqr API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%236366f1'><path d='M4 4h7v7H4V4zm2 2v3h3V6H6zm7-2h7v7h-7V4zm2 2v3h3V6h-3zM4 13h7v7H4v-7zm2 2v3h3v-3H6zm10-2h1v1h-1v-1zm3 0h1v1h-1v-1zm-3 3h1v1h-1v-1zm3 0h1v1h-1v-1zm-1-2h1v1h-1v-1zm-2 4h1v1h-1v-1zm4 0h1v1h-1v-1z'/></svg>" />
    <style>
      body { margin: 0; background-color: ` + bg + `; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/api/v1/openapi.json"
      data-configuration='{"theme":"purple","darkMode":` + dark + `,"showSidebar":true,"hideDownloadButton":false,"defaultHttpClient":{"targetKey":"shell","clientKey":"curl"}}'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

	c.Set("Content-Type", "text/html; charset=utf-8")
	// Embedded in the app's API Reference page, so same-origin framing is allowed.
	c.Set("X-Frame-Options", "SAMEORIGIN")
	// Scalar is loaded from jsDelivr and injects inline styles and fonts.
	c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'; "+
		"style-src 'self' 'unsafe-inline' https:; font-src 'self' https: data:; img-src 'self' data: https:; "+
		"connect-src 'self' https://cdn.jsdelivr.net; frame-ancestors 'self'; object-src 'none'")
	return c.SendString(html)
}

var (
	specOnce sync.Once
	specJSON []byte
	specErr  error
)

// HandleOpenAPISpec serves docs/openapi.yaml as JSON (the YAML is the source of truth).
func (h *Handler) HandleOpenAPISpec(c fiber.Ctx) error {
	specOnce.Do(func() {
		var doc any
		if specErr = yaml.Unmarshal(docs.OpenAPIYAML, &doc); specErr == nil {
			specJSON, specErr = json.Marshal(doc)
		}
	})
	if specErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "spec unavailable"})
	}
	c.Set("Content-Type", "application/json")
	c.Set("Cache-Control", "public, max-age=300")
	return c.Send(specJSON)
}

// HandleOpenAPIYAML serves the raw specification.
func (h *Handler) HandleOpenAPIYAML(c fiber.Ctx) error {
	c.Set("Content-Type", "application/yaml")
	c.Set("Cache-Control", "public, max-age=300")
	return c.Send(docs.OpenAPIYAML)
}
