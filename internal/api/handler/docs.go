package handler

import (
	"github.com/gofiber/fiber/v3"
)

// HandleAPIDocs renders an interactive, modern Scalar documentation page
func (h *Handler) HandleAPIDocs(c fiber.Ctx) error {
	html := `<!doctype html>
<html>
  <head>
    <title>Linksy API Reference & Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%236366f1'><path d='M4 4h7v7H4V4zm2 2v3h3V6H6zm7-2h7v7h-7V4zm2 2v3h3V6h-3zM4 13h7v7H4v-7zm2 2v3h3v-3H6zm10-2h1v1h-1v-1zm3 0h1v1h-1v-1zm-3 3h1v1h-1v-1zm3 0h1v1h-1v-1zm-1-2h1v1h-1v-1zm-2 4h1v1h-1v-1zm4 0h1v1h-1v-1z'/></svg>" />
    <style>
      body { margin: 0; background-color: #0f172a; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/api/v1/openapi.json"
      data-configuration='{"theme":"purple","darkMode":true,"showSidebar":true}'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

// HandleOpenAPISpec serves the OpenAPI 3.1 specification JSON
func (h *Handler) HandleOpenAPISpec(c fiber.Ctx) error {
	spec := fiber.Map{
		"openapi": "3.1.0",
		"info": fiber.Map{
			"title":       "Linksy REST API",
			"version":     "1.0.0",
			"description": "Public REST API for Linksy - The dynamic QR code and short link generator platform.",
			"contact": fiber.Map{
				"name": "Linksy Support",
				"url":  "https://github.com/bcrowe306/linksy",
			},
		},
		"servers": []fiber.Map{
			{
				"url":         "/",
				"description": "Current Environment Server",
			},
		},
		"components": fiber.Map{
			"securitySchemes": fiber.Map{
				"BearerAuth": fiber.Map{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT / API Key",
					"description":  "Pass your WorkOS bearer token or Linksy API key",
				},
			},
		},
		"paths": fiber.Map{
			"/api/v1/health": fiber.Map{
				"get": fiber.Map{
					"summary":     "Health Check",
					"description": "Check service status and responsiveness",
					"responses": fiber.Map{
						"200": fiber.Map{"description": "Service is healthy"},
					},
				},
			},
			"/s/{slug}": fiber.Map{
				"get": fiber.Map{
					"summary":     "Resolve Short Link",
					"description": "Resolves slug and performs a 302 redirect to target destination while tracking click analytics",
					"parameters": []fiber.Map{
						{"name": "slug", "in": "path", "required": true, "schema": fiber.Map{"type": "string"}},
					},
					"responses": fiber.Map{
						"302": fiber.Map{"description": "Redirects to destination URL"},
						"404": fiber.Map{"description": "Link not found or expired"},
					},
				},
			},
			"/api/v1/links": fiber.Map{
				"get": fiber.Map{
					"summary":     "List Short Links",
					"security":    []fiber.Map{{"BearerAuth": []string{}}},
					"description": "List short links belonging to the authorized team",
					"parameters": []fiber.Map{
						{"name": "limit", "in": "query", "schema": fiber.Map{"type": "integer", "default": 50}},
						{"name": "offset", "in": "query", "schema": fiber.Map{"type": "integer", "default": 0}},
					},
					"responses": fiber.Map{
						"200": fiber.Map{"description": "List of links"},
					},
				},
				"post": fiber.Map{
					"summary":     "Create Short Link",
					"security":    []fiber.Map{{"BearerAuth": []string{}}},
					"description": "Generate a new short link with optional custom slug",
					"requestBody": fiber.Map{
						"required": true,
						"content": fiber.Map{
							"application/json": fiber.Map{
								"schema": fiber.Map{
									"type":     "object",
									"required": []string{"target_url"},
									"properties": fiber.Map{
										"target_url":  fiber.Map{"type": "string", "example": "https://example.com/promo"},
										"custom_slug": fiber.Map{"type": "string", "example": "summer-sale"},
										"title":       fiber.Map{"type": "string", "example": "Summer Campaign"},
										"description": fiber.Map{"type": "string"},
										"expires_at":  fiber.Map{"type": "string", "format": "date-time"},
									},
								},
							},
						},
					},
					"responses": fiber.Map{
						"201": fiber.Map{"description": "Short link created"},
						"400": fiber.Map{"description": "Validation error or slug collision"},
					},
				},
			},
			"/api/v1/links/{id}": fiber.Map{
				"put": fiber.Map{
					"summary":     "Update Short Link Destination (Dynamic Routing)",
					"security":    []fiber.Map{{"BearerAuth": []string{}}},
					"description": "Change the destination of a short link without breaking existing URLs or QR codes",
					"parameters": []fiber.Map{
						{"name": "id", "in": "path", "required": true, "schema": fiber.Map{"type": "string"}},
					},
					"requestBody": fiber.Map{
						"content": fiber.Map{
							"application/json": fiber.Map{
								"schema": fiber.Map{
									"type": "object",
									"properties": fiber.Map{
										"target_url":  fiber.Map{"type": "string"},
										"title":       fiber.Map{"type": "string"},
										"description": fiber.Map{"type": "string"},
										"is_active":   fiber.Map{"type": "boolean"},
									},
								},
							},
						},
					},
					"responses": fiber.Map{
						"200": fiber.Map{"description": "Link destination updated"},
					},
				},
				"delete": fiber.Map{
					"summary":  "Delete Short Link",
					"security": []fiber.Map{{"BearerAuth": []string{}}},
					"parameters": []fiber.Map{
						{"name": "id", "in": "path", "required": true, "schema": fiber.Map{"type": "string"}},
					},
					"responses": fiber.Map{
						"204": fiber.Map{"description": "Link deleted"},
					},
				},
			},
			"/api/v1/qr": fiber.Map{
				"get": fiber.Map{
					"summary":  "List QR Codes",
					"security": []fiber.Map{{"BearerAuth": []string{}}},
					"responses": fiber.Map{
						"200": fiber.Map{"description": "List of QR codes"},
					},
				},
				"post": fiber.Map{
					"summary":     "Create Customized QR Code",
					"security":    []fiber.Map{{"BearerAuth": []string{}}},
					"description": "Create a customized QR code (website, vcard, wifi, doc, text, email, sms) with color styling, borders, and logo embedding",
					"requestBody": fiber.Map{
						"required": true,
						"content": fiber.Map{
							"application/json": fiber.Map{
								"schema": fiber.Map{
									"type":     "object",
									"required": []string{"type", "title"},
									"properties": fiber.Map{
										"type":       fiber.Map{"type": "string", "enum": []string{"website", "document", "vcard", "wifi", "app", "text", "email", "sms"}},
										"title":      fiber.Map{"type": "string"},
										"target_url": fiber.Map{"type": "string"},
										"raw_data":   fiber.Map{"type": "string"},
										"is_dynamic": fiber.Map{"type": "boolean", "default": true},
										"style": fiber.Map{
											"type": "object",
											"properties": fiber.Map{
												"fg_color":     fiber.Map{"type": "string", "example": "#000000"},
												"bg_color":     fiber.Map{"type": "string", "example": "#ffffff"},
												"border_width": fiber.Map{"type": "integer", "example": 4},
												"border_color": fiber.Map{"type": "string", "example": "#3b82f6"},
												"label_text":   fiber.Map{"type": "string", "example": "SCAN ME"},
												"logo_base64":  fiber.Map{"type": "string"},
											},
										},
									},
								},
							},
						},
					},
					"responses": fiber.Map{
						"201": fiber.Map{"description": "QR code created"},
					},
				},
			},
			"/api/v1/qr/{id}/export": fiber.Map{
				"get": fiber.Map{
					"summary":     "Export QR Code",
					"description": "Export the QR code in multiple formats: PNG, JPEG, SVG, or printable PDF",
					"parameters": []fiber.Map{
						{"name": "id", "in": "path", "required": true, "schema": fiber.Map{"type": "string"}},
						{"name": "format", "in": "query", "schema": fiber.Map{"type": "string", "enum": []string{"png", "jpeg", "svg", "pdf"}, "default": "png"}},
						{"name": "size", "in": "query", "schema": fiber.Map{"type": "integer", "default": 512}},
						{"name": "download", "in": "query", "schema": fiber.Map{"type": "boolean", "default": false}},
					},
					"responses": fiber.Map{
						"200": fiber.Map{"description": "Image or PDF byte stream"},
					},
				},
			},
			"/api/v1/documents/upload": fiber.Map{
				"post": fiber.Map{
					"summary":     "Upload Document & Auto-Generate QR Code",
					"security":    []fiber.Map{{"BearerAuth": []string{}}},
					"description": "Upload a PDF, document, or menu file and receive a dynamic QR code for instant sharing",
					"requestBody": fiber.Map{
						"content": fiber.Map{
							"multipart/form-data": fiber.Map{
								"schema": fiber.Map{
									"type": "object",
									"properties": fiber.Map{
										"file": fiber.Map{"type": "string", "format": "binary"},
									},
								},
							},
						},
					},
					"responses": fiber.Map{
						"201": fiber.Map{"description": "Document uploaded and dynamic QR code generated"},
					},
				},
			},
			"/api/v1/analytics/summary": fiber.Map{
				"get": fiber.Map{
					"summary":     "Get Analytics Summary & Metrics",
					"security":    []fiber.Map{{"BearerAuth": []string{}}},
					"description": "Get aggregate scans, clicks, top referrers, device breakdown, browser metrics, and geography",
					"parameters": []fiber.Map{
						{"name": "link_id", "in": "query", "schema": fiber.Map{"type": "string"}},
						{"name": "qr_id", "in": "query", "schema": fiber.Map{"type": "string"}},
					},
					"responses": fiber.Map{
						"200": fiber.Map{"description": "Analytics breakdown"},
					},
				},
			},
		},
	}

	return c.JSON(spec)
}
