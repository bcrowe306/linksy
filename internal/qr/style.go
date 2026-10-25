package qr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // register decoders for logo uploads
	_ "image/jpeg"
	_ "image/png"
	"math"
	"regexp"
	"strings"

	_ "golang.org/x/image/webp"

	"github.com/bcrowe306/linksy/internal/models"
)

// Option sets exposed to clients. Unknown values fall back to defaults so a
// stored style can never break rendering.
var (
	DotStyles      = []string{"square", "rounded", "extra-rounded", "dots", "classy"}
	EyeOuterShapes = []string{"square", "rounded", "extra-rounded", "circle", "leaf"}
	EyeInnerShapes = []string{"square", "rounded", "circle", "leaf"}
	FrameStyles    = []string{"none", "border", "banner-bottom", "banner-top", "card", "pill"}
	LogoShapes     = []string{"none", "circle", "rounded", "square"}
	GradientTypes  = []string{"none", "linear", "radial"}
)

const (
	MaxLogoBytes = 512 * 1024
	MaxLabelLen  = 40
	MinSize      = 128
	MaxSize      = 4096
)

var hexColor = regexp.MustCompile(`^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// normalizeHex returns a canonical "#rrggbb" or the fallback when invalid.
func normalizeHex(s, fallback string) string {
	s = strings.TrimSpace(s)
	if !hexColor.MatchString(s) {
		return fallback
	}
	s = strings.ToLower(strings.TrimPrefix(s, "#"))
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	return "#" + s
}

func oneOf(v string, allowed []string, fallback string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	// Legacy aliases from the previous editor
	switch v {
	case "circle":
		if contains(allowed, "dots") {
			return "dots"
		}
	case "box":
		if contains(allowed, "border") {
			return "border"
		}
	}
	return fallback
}

func contains(list []string, v string) bool {
	for _, a := range list {
		if a == v {
			return true
		}
	}
	return false
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// NormalizeStyle validates every field, applies defaults, and returns a style
// that is safe to embed in SVG/PDF output. It is applied before saving and
// again before rendering.
func NormalizeStyle(s models.QRStyle) models.QRStyle {
	s.FgColor = normalizeHex(s.FgColor, "#000000")
	s.BgColor = normalizeHex(s.BgColor, "#ffffff")
	s.GradientType = oneOf(s.GradientType, GradientTypes, "none")
	s.GradientStart = normalizeHex(s.GradientStart, s.FgColor)
	s.GradientEnd = normalizeHex(s.GradientEnd, s.FgColor)
	s.GradientAngle = ((s.GradientAngle % 360) + 360) % 360

	s.DotStyle = oneOf(s.DotStyle, DotStyles, "square")
	s.EyeShape = oneOf(s.EyeShape, EyeOuterShapes, "square")
	s.EyeInnerShape = oneOf(s.EyeInnerShape, EyeInnerShapes, innerFor(s.EyeShape))
	s.EyeColor = normalizeHex(s.EyeColor, "")
	s.EyeInnerColor = normalizeHex(s.EyeInnerColor, "")

	s.ErrorCorrection = strings.ToUpper(strings.TrimSpace(s.ErrorCorrection))
	if !contains([]string{"L", "M", "Q", "H"}, s.ErrorCorrection) {
		s.ErrorCorrection = "M"
	}
	// A quiet zone is required by the QR spec; 0 (unset) means the default of 2 modules.
	if s.Margin <= 0 {
		s.Margin = 2
	}
	s.Margin = clampInt(s.Margin, 1, 10)

	s.LogoURL = "" // remote logos are not fetched (SSRF); clients upload as data URIs
	if s.LogoBase64 != "" {
		if _, err := DecodeLogo(s.LogoBase64); err != nil {
			s.LogoBase64 = ""
		}
	}
	if s.LogoSize == 0 {
		s.LogoSize = 0.22
	}
	s.LogoSize = clampFloat(s.LogoSize, 0.12, 0.3)
	s.LogoBgShape = oneOf(s.LogoBgShape, LogoShapes, "rounded")
	s.LogoPadding = clampInt(s.LogoPadding, 0, 40)
	if s.LogoBase64 != "" && s.ErrorCorrection != "H" {
		// Covering modules with a logo needs maximum redundancy to stay scannable.
		s.ErrorCorrection = "H"
	}

	s.FrameStyle = oneOf(s.FrameStyle, FrameStyles, "none")
	if s.FrameStyle == "none" && (s.BorderWidth > 0 || strings.TrimSpace(s.LabelText) != "") {
		// Older styles described frames only through these fields.
		if strings.TrimSpace(s.LabelText) != "" {
			s.FrameStyle = "banner-bottom"
		} else {
			s.FrameStyle = "border"
		}
	}
	s.BorderWidth = clampInt(s.BorderWidth, 0, 40)
	s.BorderColor = normalizeHex(s.BorderColor, s.FgColor)
	s.BorderRadius = clampInt(s.BorderRadius, 0, 64)
	s.FrameBgColor = normalizeHex(s.FrameBgColor, s.BgColor)
	s.LabelText = strings.Join(strings.Fields(s.LabelText), " ")
	if r := []rune(s.LabelText); len(r) > MaxLabelLen {
		s.LabelText = string(r[:MaxLabelLen])
	}
	s.LabelColor = normalizeHex(s.LabelColor, "#ffffff")
	if s.LabelSize == 0 {
		s.LabelSize = 100
	}
	s.LabelSize = clampInt(s.LabelSize, 50, 160) // percent of the default label size
	return s
}

func innerFor(outer string) string {
	switch outer {
	case "circle":
		return "circle"
	case "rounded", "extra-rounded":
		return "rounded"
	case "leaf":
		return "leaf"
	}
	return "square"
}

var logoPrefix = regexp.MustCompile(`^data:image/(png|jpeg|jpg|gif|webp);base64,`)

// DecodeLogo validates and decodes an uploaded logo data URI.
func DecodeLogo(dataURI string) (image.Image, error) {
	m := logoPrefix.FindString(dataURI)
	if m == "" {
		return nil, fmt.Errorf("logo must be a PNG, JPEG, GIF or WebP data URI")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(dataURI[len(m):]))
	if err != nil {
		return nil, fmt.Errorf("logo is not valid base64")
	}
	if len(raw) > MaxLogoBytes {
		return nil, fmt.Errorf("logo must be smaller than %d KB", MaxLogoBytes/1024)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("logo could not be decoded")
	}
	if cfg.Width > 4096 || cfg.Height > 4096 {
		return nil, fmt.Errorf("logo dimensions must be at most 4096px")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

// ParseHexColor converts "#rrggbb" into an opaque color, or returns def.
func ParseHexColor(s string, def color.RGBA) color.RGBA {
	n := normalizeHex(s, "")
	if n == "" {
		return def
	}
	var r, g, b uint8
	fmt.Sscanf(n[1:], "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// relativeLuminance implements WCAG 2.x luminance for contrast checks.
func relativeLuminance(c color.RGBA) float64 {
	ch := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// ContrastRatio returns the WCAG contrast ratio between two colors (1–21).
func ContrastRatio(a, b color.RGBA) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}
