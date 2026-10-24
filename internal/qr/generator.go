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

// ParseHexColor parses hex string like #RRGGBB, #RGB or rgba into color.RGBA
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
	} else if len(s) == 8 {
		r, _ := strconv.ParseUint(s[0:2], 16, 8)
		g, _ := strconv.ParseUint(s[2:4], 16, 8)
		b, _ := strconv.ParseUint(s[4:6], 16, 8)
		a, _ := strconv.ParseUint(s[6:8], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
	}
	return defaultColor
}

func getRecoveryLevel(level string) qrcode.RecoveryLevel {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "L":
		return qrcode.Low
	case "M":
		return qrcode.Medium
	case "Q":
		return qrcode.High
	case "H", "":
		return qrcode.Highest
	default:
		return qrcode.Highest
	}
}

// GenerateImage generates an RGBA image supporting all QR specs, dots, gradients, eyes, and logos
func GenerateImage(content string, size int, style models.QRStyle) (image.Image, error) {
	if size < 128 {
		size = 256
	}

	recovery := getRecoveryLevel(style.ErrorCorrection)
	// If logo is present, always enforce at least High recovery for scannability
	if style.LogoBase64 != "" || style.LogoURL != "" {
		recovery = qrcode.Highest
	}

	qr, err := qrcode.New(content, recovery)
	if err != nil {
		return nil, fmt.Errorf("failed generating QR matrix: %w", err)
	}

	bitmap := qr.Bitmap()
	numModules := len(bitmap)
	if numModules == 0 {
		return nil, fmt.Errorf("empty QR matrix")
	}

	// Calculate quiet zone / margin
	marginModules := style.Margin
	if marginModules < 0 {
		marginModules = 0
	}
	if marginModules > 10 {
		marginModules = 10
	}

	totalModules := numModules + (marginModules * 2)
	moduleSize := float64(size) / float64(totalModules)
	actualSize := int(float64(totalModules) * moduleSize)

	fgColor := ParseHexColor(style.FgColor, color.RGBA{0, 0, 0, 255})
	bgColor := ParseHexColor(style.BgColor, color.RGBA{255, 255, 255, 255})
	if style.IsTransparent {
		bgColor = color.RGBA{0, 0, 0, 0}
	}

	useGradient := style.GradientType == "linear" && style.GradientStart != "" && style.GradientEnd != ""
	gradStart := ParseHexColor(style.GradientStart, fgColor)
	gradEnd := ParseHexColor(style.GradientEnd, fgColor)

	// Create base canvas
	rgba := image.NewRGBA(image.Rect(0, 0, actualSize, actualSize))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Helper to check if a module coordinate is part of one of the 3 finder eyes
	isFinderEye := func(mx, my int) bool {
		// Top-left eye
		if mx < 7 && my < 7 {
			return true
		}
		// Top-right eye
		if mx >= numModules-7 && my < 7 {
			return true
		}
		// Bottom-left eye
		if mx < 7 && my >= numModules-7 {
			return true
		}
		return false
	}

	eyeColor := fgColor
	if style.EyeColor != "" {
		eyeColor = ParseHexColor(style.EyeColor, fgColor)
	}

	eyeInnerColor := eyeColor
	if style.EyeInnerColor != "" {
		eyeInnerColor = ParseHexColor(style.EyeInnerColor, eyeColor)
	}

	// Helper to calculate module color (gradient or solid)
	getModuleColor := func(px, py int) color.RGBA {
		if !useGradient {
			return fgColor
		}
		// Interpolate along angle
		progress := float64(px+py) / float64(actualSize*2)
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
		r := uint8(float64(gradStart.R)*(1-progress) + float64(gradEnd.R)*progress)
		g := uint8(float64(gradStart.G)*(1-progress) + float64(gradEnd.G)*progress)
		b := uint8(float64(gradStart.B)*(1-progress) + float64(gradEnd.B)*progress)
		return color.RGBA{R: r, G: g, B: b, A: 255}
	}

	// Draw each module
	dotStyle := strings.ToLower(style.DotStyle)
	if dotStyle == "" {
		dotStyle = "square"
	}

	for my := 0; my < numModules; my++ {
		for mx := 0; mx < numModules; mx++ {
			if !bitmap[my][mx] {
				continue
			}

			x0 := int(float64(mx+marginModules) * moduleSize)
			y0 := int(float64(my+marginModules) * moduleSize)
			x1 := int(float64(mx+marginModules+1) * moduleSize)
			y1 := int(float64(my+marginModules+1) * moduleSize)

			var col color.RGBA
			if isFinderEye(mx, my) {
				// Eye module color
				if isInnerEyeBall(mx, my, numModules) {
					col = eyeInnerColor
				} else {
					col = eyeColor
				}
			} else {
				col = getModuleColor(x0, y0)
			}

			if isFinderEye(mx, my) {
				// Finder patterns keep solid structure
				drawRect(rgba, x0, y0, x1, y1, col)
			} else {
				switch dotStyle {
				case "dots", "circle":
					drawCircle(rgba, x0, y0, x1, y1, col)
				case "rounded":
					drawRoundedRect(rgba, x0, y0, x1, y1, int(moduleSize*0.35), col)
				case "classy":
					drawRoundedRect(rgba, x0, y0, x1, y1, int(moduleSize*0.2), col)
				default:
					drawRect(rgba, x0, y0, x1, y1, col)
				}
			}
		}
	}

	// Center Logo Embedding
	if style.LogoBase64 != "" {
		logoImg, err := decodeBase64Image(style.LogoBase64)
		if err == nil && logoImg != nil {
			logoScale := style.LogoSize
			if logoScale < 0.15 || logoScale > 0.30 {
				logoScale = 0.22
			}
			bgShape := style.LogoBgShape
			if bgShape == "" {
				bgShape = "circle"
			}
			padding := style.LogoPadding
			if padding <= 0 {
				padding = 6
			}
			embedStyledLogo(rgba, logoImg, bgColor, logoScale, bgShape, padding)
		}
	}

	// Frame & Border Application
	finalImg := rgba
	if style.FrameStyle != "none" || style.BorderWidth > 0 || style.LabelText != "" {
		finalImg = applyFrame(rgba, style, fgColor, bgColor)
	}

	return finalImg, nil
}

func isInnerEyeBall(mx, my, total int) bool {
	// Top-left eyeball (2..4, 2..4)
	if mx >= 2 && mx <= 4 && my >= 2 && my <= 4 {
		return true
	}
	// Top-right eyeball
	if mx >= total-5 && mx <= total-3 && my >= 2 && my <= 4 {
		return true
	}
	// Bottom-left eyeball
	if mx >= 2 && mx <= 4 && my >= total-5 && my <= total-3 {
		return true
	}
	return false
}

func drawRect(dest *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dest.Set(x, y, col)
		}
	}
}

func drawCircle(dest *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	cx := float64(x0+x1) / 2.0
	cy := float64(y0+y1) / 2.0
	r := float64(x1-x0) / 2.0 * 0.92

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if (dx*dx)+(dy*dy) <= (r * r) {
				dest.Set(x, y, col)
			}
		}
	}
}

func drawRoundedRect(dest *image.RGBA, x0, y0, x1, y1 int, radius int, col color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			// Check 4 corners
			if x < x0+radius && y < y0+radius {
				dx := float64(x - (x0 + radius))
				dy := float64(y - (y0 + radius))
				if (dx*dx)+(dy*dy) > float64(radius*radius) {
					continue
				}
			} else if x >= x1-radius && y < y0+radius {
				dx := float64(x - (x1 - radius))
				dy := float64(y - (y0 + radius))
				if (dx*dx)+(dy*dy) > float64(radius*radius) {
					continue
				}
			} else if x < x0+radius && y >= y1-radius {
				dx := float64(x - (x0 + radius))
				dy := float64(y - (y1 - radius))
				if (dx*dx)+(dy*dy) > float64(radius*radius) {
					continue
				}
			} else if x >= x1-radius && y >= y1-radius {
				dx := float64(x - (x1 - radius))
				dy := float64(y - (y1 - radius))
				if (dx*dx)+(dy*dy) > float64(radius*radius) {
					continue
				}
			}
			dest.Set(x, y, col)
		}
	}
}

func decodeBase64Image(data string) (image.Image, error) {
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

func embedStyledLogo(dest *image.RGBA, logo image.Image, bgColor color.RGBA, scale float64, bgShape string, padding int) {
	destBounds := dest.Bounds()
	destW := destBounds.Dx()
	destH := destBounds.Dy()

	targetW := int(float64(destW) * scale)
	targetH := int(float64(destH) * scale)
	if targetW < 16 {
		targetW = 16
	}
	if targetH < 16 {
		targetH = 16
	}

	startX := (destW - targetW) / 2
	startY := (destH - targetH) / 2

	badgeW := targetW + padding*2
	badgeH := targetH + padding*2
	badgeX := startX - padding
	badgeY := startY - padding

	// Draw badge shape behind logo for high contrast
	if bgShape == "circle" {
		cx := float64(destW) / 2.0
		cy := float64(destH) / 2.0
		r := float64(badgeW) / 2.0
		for y := badgeY; y < badgeY+badgeH; y++ {
			for x := badgeX; x < badgeX+badgeW; x++ {
				dx := float64(x) - cx
				dy := float64(y) - cy
				if (dx*dx)+(dy*dy) <= (r * r) {
					dest.Set(x, y, bgColor)
				}
			}
		}
	} else if bgShape == "rounded" {
		drawRoundedRect(dest, badgeX, badgeY, badgeX+badgeW, badgeY+badgeH, 8, bgColor)
	} else if bgShape == "square" {
		drawRect(dest, badgeX, badgeY, badgeX+badgeW, badgeY+badgeH, bgColor)
	}

	// Resize and draw logo
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

func applyFrame(src *image.RGBA, style models.QRStyle, fgColor, bgColor color.RGBA) *image.RGBA {
	borderWidth := style.BorderWidth
	if borderWidth < 0 {
		borderWidth = 0
	}
	borderColor := ParseHexColor(style.BorderColor, fgColor)
	frameBg := ParseHexColor(style.FrameBgColor, bgColor)

	hasLabel := strings.TrimSpace(style.LabelText) != ""
	bannerHeight := 0
	if hasLabel {
		bannerHeight = 36
	}

	srcBounds := src.Bounds()
	pad := borderWidth + 12
	newW := srcBounds.Dx() + pad*2
	newH := srcBounds.Dy() + pad*2 + bannerHeight

	newImg := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{frameBg}, image.Point{}, draw.Src)

	// Draw border frame
	if borderWidth > 0 {
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				if x < borderWidth || x >= newW-borderWidth || y < borderWidth || y >= newH-borderWidth {
					newImg.Set(x, y, borderColor)
				}
			}
		}
	}

	// Place QR in center
	destRect := image.Rect(pad, pad, pad+srcBounds.Dx(), pad+srcBounds.Dy())
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
	// JPEG does not support transparency, enforce white background if transparent
	if style.IsTransparent {
		style.IsTransparent = false
		style.BgColor = "#FFFFFF"
	}
	img, err := GenerateImage(content, size, style)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 98}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderSVG renders the QR code as rich scalable vector SVG
func RenderSVG(content string, size int, style models.QRStyle) ([]byte, error) {
	if size < 128 {
		size = 256
	}

	recovery := getRecoveryLevel(style.ErrorCorrection)
	if style.LogoBase64 != "" {
		recovery = qrcode.Highest
	}

	qr, err := qrcode.New(content, recovery)
	if err != nil {
		return nil, err
	}

	bitmap := qr.Bitmap()
	numModules := len(bitmap)
	if numModules == 0 {
		return nil, fmt.Errorf("empty QR matrix")
	}

	margin := style.Margin
	if margin < 0 {
		margin = 0
	}
	total := numModules + margin*2

	fgHex := style.FgColor
	if fgHex == "" {
		fgHex = "#000000"
	}
	bgHex := style.BgColor
	if bgHex == "" {
		bgHex = "#ffffff"
	}

	dotStyle := strings.ToLower(style.DotStyle)
	useGradient := style.GradientType == "linear" && style.GradientStart != "" && style.GradientEnd != ""

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 %d %d" width="%d" height="%d">`+"\n", total, total, size, size))

	if useGradient {
		sb.WriteString("  <defs>\n")
		sb.WriteString(fmt.Sprintf(`    <linearGradient id="qrGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">` + "\n"))
		sb.WriteString(fmt.Sprintf(`      <stop offset="0%%" stop-color="%s" />`+"\n", style.GradientStart))
		sb.WriteString(fmt.Sprintf(`      <stop offset="100%%" stop-color="%s" />`+"\n", style.GradientEnd))
		sb.WriteString("    </linearGradient>\n")
		sb.WriteString("  </defs>\n")
		fgHex = "url(#qrGrad)"
	}

	if !style.IsTransparent {
		sb.WriteString(fmt.Sprintf(`  <rect width="%d" height="%d" fill="%s" />`+"\n", total, total, bgHex))
	}

	for y := 0; y < numModules; y++ {
		for x := 0; x < numModules; x++ {
			if !bitmap[y][x] {
				continue
			}
			mx := x + margin
			my := y + margin

			if dotStyle == "dots" || dotStyle == "circle" {
				sb.WriteString(fmt.Sprintf(`  <circle cx="%.2f" cy="%.2f" r="0.45" fill="%s" />`+"\n", float64(mx)+0.5, float64(my)+0.5, fgHex))
			} else if dotStyle == "rounded" {
				sb.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="1" height="1" rx="0.3" fill="%s" />`+"\n", mx, my, fgHex))
			} else {
				sb.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="1" height="1" fill="%s" />`+"\n", mx, my, fgHex))
			}
		}
	}

	// Center Logo in SVG
	if style.LogoBase64 != "" {
		logoSize := float64(total) * 0.22
		logoX := (float64(total) - logoSize) / 2.0
		logoY := (float64(total) - logoSize) / 2.0
		badgePad := 0.4
		// Badge
		if style.LogoBgShape == "circle" {
			sb.WriteString(fmt.Sprintf(`  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" />`+"\n", float64(total)/2.0, float64(total)/2.0, (logoSize/2.0)+badgePad, bgHex))
		} else {
			sb.WriteString(fmt.Sprintf(`  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="0.5" fill="%s" />`+"\n", logoX-badgePad, logoY-badgePad, logoSize+(badgePad*2), logoSize+(badgePad*2), bgHex))
		}
		sb.WriteString(fmt.Sprintf(`  <image x="%.2f" y="%.2f" width="%.2f" height="%.2f" xlink:href="%s" />`+"\n", logoX, logoY, logoSize, logoSize, style.LogoBase64))
	}

	sb.WriteString(`</svg>`)
	return []byte(sb.String()), nil
}

// RenderPDF renders a print-ready vector PDF document
func RenderPDF(content string, title string, style models.QRStyle) ([]byte, error) {
	pngBytes, err := RenderPNG(content, 600, style)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Linksy - "+title, false)
	pdf.AddPage()

	// Header branding
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetTextColor(15, 23, 42)
	if title == "" {
		title = "Linksy QR Code"
	}
	pdf.CellFormat(190, 16, title, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(190, 7, "Point your smartphone camera to scan", "", 1, "C", false, 0, "")
	pdf.Ln(8)

	// Embed QR image
	imgName := "qr_image_" + strconv.FormatInt(int64(len(content)), 10)
	imageOptions := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   true,
	}
	pdf.RegisterImageOptionsReader(imgName, imageOptions, bytes.NewReader(pngBytes))

	// Center 120mm x 120mm
	pdf.ImageOptions(imgName, 45, 55, 120, 120, false, imageOptions, 0, "")

	// Label underneath
	pdf.SetY(185)
	if style.LabelText != "" {
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(190, 10, style.LabelText, "", 1, "C", false, 0, "")
	}

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(190, 6, "Dynamic QR Powered by Linksy • Always Up-to-Date", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed generating PDF: %w", err)
	}

	return buf.Bytes(), nil
}
