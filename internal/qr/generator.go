package qr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"

	"github.com/bcrowe306/linksy/internal/models"
)

// ParseHexColor parses a hex string like #RRGGBB or #RGB into color.RGBA
func ParseHexColor(s string, defaultColor color.RGBA) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		r, err1 := strconv.ParseUint(string([]byte{s[0], s[0]}), 16, 8)
		g, err2 := strconv.ParseUint(string([]byte{s[1], s[1]}), 16, 8)
		b, err3 := strconv.ParseUint(string([]byte{s[2], s[2]}), 16, 8)
		if err1 == nil && err2 == nil && err3 == nil {
			return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
		}
	} else if len(s) == 6 {
		r, err1 := strconv.ParseUint(s[0:2], 16, 8)
		g, err2 := strconv.ParseUint(s[2:4], 16, 8)
		b, err3 := strconv.ParseUint(s[4:6], 16, 8)
		if err1 == nil && err2 == nil && err3 == nil {
			return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
		}
	}
	return defaultColor
}

// GenerateImage generates an RGBA image of the QR code with colors, borders, and optional center logo
func GenerateImage(content string, size int, style models.QRStyle) (image.Image, error) {
	if size < 128 {
		size = 256
	}

	fgColor := ParseHexColor(style.FgColor, color.RGBA{0, 0, 0, 255})
	bgColor := ParseHexColor(style.BgColor, color.RGBA{255, 255, 255, 255})

	// Generate QR code bitmap with High error correction (30% recovery)
	qr, err := qrcode.New(content, qrcode.Highest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR matrix: %w", err)
	}

	qr.ForegroundColor = fgColor
	qr.BackgroundColor = bgColor

	// Create base QR image
	img := qr.Image(size)

	// Convert to RGBA for mutation
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Check if logo is provided
	if style.LogoBase64 != "" {
		logoImg, err := decodeBase64Image(style.LogoBase64)
		if err == nil && logoImg != nil {
			embedLogo(rgba, logoImg, bgColor)
		}
	}

	// Add optional border frame
	if style.BorderWidth > 0 {
		borderColor := ParseHexColor(style.BorderColor, fgColor)
		rgba = addBorder(rgba, style.BorderWidth, borderColor, bgColor)
	}

	return rgba, nil
}

func decodeBase64Image(data string) (image.Image, error) {
	// Strip data URI scheme prefix if present (e.g. data:image/png;base64,...)
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(data))
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

func embedLogo(dest *image.RGBA, logo image.Image, bgColor color.RGBA) {
	destBounds := dest.Bounds()
	destW := destBounds.Dx()
	destH := destBounds.Dy()

	// Logo size: 22% of QR dimensions (safe within 30% error correction margin)
	targetW := int(float64(destW) * 0.22)
	targetH := int(float64(destH) * 0.22)
	if targetW < 16 {
		targetW = 16
	}
	if targetH < 16 {
		targetH = 16
	}

	// Position logo in center
	startX := (destW - targetW) / 2
	startY := (destH - targetH) / 2

	// Draw background badge behind logo (with 4px padding) for high contrast
	badgePad := 4
	for y := startY - badgePad; y < startY+targetH+badgePad; y++ {
		for x := startX - badgePad; x < startX+targetW+badgePad; x++ {
			if x >= 0 && x < destW && y >= 0 && y < destH {
				dest.Set(x, y, bgColor)
			}
		}
	}

	// Simple nearest-neighbor resize and draw logo
	logoBounds := logo.Bounds()
	srcW := logoBounds.Dx()
	srcH := logoBounds.Dy()

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := logoBounds.Min.X + (x * srcW / targetW)
			srcY := logoBounds.Min.Y + (y * srcH / targetH)
			col := logo.At(srcX, srcY)
			_, _, _, a := col.RGBA()
			if a > 0 {
				dest.Set(startX+x, startY+y, col)
			}
		}
	}
}

func addBorder(src *image.RGBA, borderWidth int, borderColor, bgColor color.RGBA) *image.RGBA {
	srcBounds := src.Bounds()
	totalPadding := borderWidth + 8
	newW := srcBounds.Dx() + totalPadding*2
	newH := srcBounds.Dy() + totalPadding*2

	newImg := image.NewRGBA(image.Rect(0, 0, newW, newH))

	// Fill background
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			newImg.Set(x, y, bgColor)
		}
	}

	// Draw outer border frame
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			if x < borderWidth || x >= newW-borderWidth || y < borderWidth || y >= newH-borderWidth {
				newImg.Set(x, y, borderColor)
			}
		}
	}

	// Paste QR in inner area
	destRect := image.Rect(totalPadding, totalPadding, totalPadding+srcBounds.Dx(), totalPadding+srcBounds.Dy())
	draw.Draw(newImg, destRect, src, srcBounds.Min, draw.Over)

	return newImg
}

// RenderPNG renders the QR code to a PNG byte slice
func RenderPNG(content string, size int, style models.QRStyle) ([]byte, error) {
	img, err := GenerateImage(content, size, style)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderJPEG renders the QR code to a JPEG byte slice
func RenderJPEG(content string, size int, style models.QRStyle) ([]byte, error) {
	img, err := GenerateImage(content, size, style)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderSVG renders the QR code as vector SVG
func RenderSVG(content string, size int, style models.QRStyle) ([]byte, error) {
	if size < 128 {
		size = 256
	}
	fgHex := style.FgColor
	if fgHex == "" {
		fgHex = "#000000"
	}
	bgHex := style.BgColor
	if bgHex == "" {
		bgHex = "#ffffff"
	}

	qr, err := qrcode.New(content, qrcode.Highest)
	if err != nil {
		return nil, err
	}

	bitmap := qr.Bitmap()
	numModules := len(bitmap)
	if numModules == 0 {
		return nil, fmt.Errorf("empty QR matrix")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`+"\n", numModules, numModules, size, size))
	sb.WriteString(fmt.Sprintf(`  <rect width="%d" height="%d" fill="%s" />`+"\n", numModules, numModules, bgHex))
	sb.WriteString(fmt.Sprintf(`  <path fill="%s" d="`, fgHex))

	for y := 0; y < numModules; y++ {
		for x := 0; x < numModules; x++ {
			if bitmap[y][x] {
				sb.WriteString(fmt.Sprintf("M%d,%dh1v1h-1z ", x, y))
			}
		}
	}
	sb.WriteString(`" />` + "\n")
	sb.WriteString(`</svg>`)

	return []byte(sb.String()), nil
}

// RenderPDF renders the QR code formatted as a clean printable PDF page
func RenderPDF(content string, title string, style models.QRStyle) ([]byte, error) {
	pngBytes, err := RenderPNG(content, 512, style)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Linksy QR Code - "+title, false)
	pdf.AddPage()

	// Header branding
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(30, 41, 59)
	if title == "" {
		title = "Linksy QR Code"
	}
	pdf.CellFormat(190, 15, title, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(190, 8, "Scan with your camera or QR reader", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Embed QR image
	imgName := "qr_image"
	imageOptions := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   true,
	}
	pdf.RegisterImageOptionsReader(imgName, imageOptions, bytes.NewReader(pngBytes))

	// Center on page (A4 width = 210mm, QR width = 110mm => x = (210-110)/2 = 50)
	pdf.ImageOptions(imgName, 50, 60, 110, 110, false, imageOptions, 0, "")

	// Label underneath
	pdf.SetY(185)
	if style.LabelText != "" {
		pdf.SetFont("Helvetica", "B", 14)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(190, 10, style.LabelText, "", 1, "C", false, 0, "")
	}

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(190, 6, "Generated with Linksy • Dynamic QR & Link Management", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF output: %w", err)
	}

	return buf.Bytes(), nil
}
