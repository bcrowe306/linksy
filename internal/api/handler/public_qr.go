package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/qr"
	"github.com/bcrowe306/linksy/internal/service"
)

// PublicQR powers the free QR code generator: ?format=json (default) returns an
// SVG preview with a scan check; svg/png return the file.
func (h *Handler) PublicQR(c fiber.Ctx) error {
	var in service.PublicQRInput
	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	content, style, err := h.svc.PublicQR(in)
	if err != nil {
		return fail(c, err)
	}
	c.Set("Cache-Control", "no-store")
	size := in.Size
	if size < 128 || size > 2048 {
		size = 1024
	}
	switch c.Query("format", "json") {
	case "svg":
		b, err := qr.RenderSVG(content, size, style)
		if err != nil {
			return fail(c, err)
		}
		c.Set("Content-Disposition", `attachment; filename="qr-code.svg"`)
		return c.Type("svg").Send(b)
	case "png":
		b, err := qr.RenderPNG(content, size, style)
		if err != nil {
			return fail(c, err)
		}
		c.Set("Content-Disposition", `attachment; filename="qr-code-`+strconv.Itoa(size)+`.png"`)
		return c.Type("png").Send(b)
	}
	b, err := qr.RenderSVG(content, 320, style)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"svg": string(b), "check": qr.Verify(content, style)})
}
