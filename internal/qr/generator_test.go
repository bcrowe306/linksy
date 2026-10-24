package qr_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/qr"
)

func createSampleBase64PNG() string {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestRenderPNG(t *testing.T) {
	style := models.QRStyle{
		FgColor:     "#1E40AF",
		BgColor:     "#F3F4F6",
		BorderWidth: 4,
		BorderColor: "#3B82F6",
	}

	data, err := qr.RenderPNG("https://example.com/test", 256, style)
	if err != nil {
		t.Fatalf("unexpected error rendering PNG: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("rendered PNG is empty")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode generated PNG: %v", err)
	}

	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Fatalf("invalid image dimensions: %v", img.Bounds())
	}
}

func TestAdvancedStylingAndLogo(t *testing.T) {
	logoB64 := createSampleBase64PNG()
	style := models.QRStyle{
		FgColor:         "#1E1B4B",
		BgColor:         "#FFFFFF",
		GradientType:    "linear",
		GradientStart:   "#4338CA",
		GradientEnd:     "#06B6D4",
		DotStyle:        "rounded",
		EyeShape:        "circle",
		EyeColor:        "#4F46E5",
		EyeInnerColor:   "#06B6D4",
		Margin:          2,
		ErrorCorrection: "H",
		LogoBase64:      logoB64,
		LogoSize:        0.24,
		LogoBgShape:     "circle",
		LogoPadding:     6,
		BorderWidth:     3,
		BorderColor:     "#3B82F6",
		LabelText:       "SCAN ME",
	}

	pngData, err := qr.RenderPNG("https://example.com/styled-qr", 300, style)
	if err != nil {
		t.Fatalf("failed rendering styled PNG: %v", err)
	}
	if len(pngData) == 0 {
		t.Fatal("rendered styled PNG is empty")
	}

	svgData, err := qr.RenderSVG("https://example.com/styled-qr", 300, style)
	if err != nil {
		t.Fatalf("failed rendering styled SVG: %v", err)
	}
	svgStr := string(svgData)
	if !strings.Contains(svgStr, "linearGradient") {
		t.Errorf("SVG missing linearGradient tag")
	}
	if !strings.Contains(svgStr, "image") {
		t.Errorf("SVG missing embedded logo image tag")
	}
}

func TestRenderJPEG(t *testing.T) {
	style := models.QRStyle{
		FgColor: "#000000",
		BgColor: "#FFFFFF",
	}

	data, err := qr.RenderJPEG("https://example.com/jpeg", 256, style)
	if err != nil {
		t.Fatalf("unexpected error rendering JPEG: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("rendered JPEG is empty")
	}

	// First two bytes of a valid JPEG are 0xFF, 0xD8
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatal("rendered data is not a valid JPEG stream")
	}
}

func TestRenderSVG(t *testing.T) {
	style := models.QRStyle{
		FgColor: "#10B981",
		BgColor: "#0F172A",
	}

	svgBytes, err := qr.RenderSVG("https://example.com/svg", 300, style)
	if err != nil {
		t.Fatalf("unexpected error rendering SVG: %v", err)
	}

	str := string(svgBytes)
	if !strings.HasPrefix(str, "<svg") {
		t.Errorf("SVG should start with <svg, got: %s", str[:min(len(str), 20)])
	}
	if !strings.Contains(str, "#10B981") {
		t.Errorf("SVG missing foreground color")
	}
	if !strings.Contains(str, "#0F172A") {
		t.Errorf("SVG missing background color")
	}
}

func TestRenderPDF(t *testing.T) {
	style := models.QRStyle{
		FgColor:   "#000000",
		BgColor:   "#FFFFFF",
		LabelText: "SCAN ME FOR MENU",
	}

	pdfBytes, err := qr.RenderPDF("https://example.com/menu", "Daily Lunch Menu", style)
	if err != nil {
		t.Fatalf("unexpected error rendering PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatal("rendered PDF is empty")
	}

	// PDF header is %PDF-
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatal("rendered data does not have valid PDF header")
	}
}

func TestBuildPayloads(t *testing.T) {
	// WiFi
	wifiJSON := `{"ssid":"OfficeWiFi","password":"secretpassword","encryption":"WPA","hidden":false}`
	wifiPayload, err := qr.BuildPayload(models.QRTypeWiFi, wifiJSON, "")
	if err != nil {
		t.Fatalf("failed building wifi payload: %v", err)
	}
	expectedWiFi := "WIFI:S:OfficeWiFi;T:WPA;P:secretpassword;H:false;;"
	if wifiPayload != expectedWiFi {
		t.Fatalf("expected wifi '%s', got '%s'", expectedWiFi, wifiPayload)
	}

	// vCard
	vcardJSON := `{"first_name":"Jane","last_name":"Doe","organization":"Acme Corp","phone":"+1234567890","email":"jane@example.com"}`
	vcardPayload, err := qr.BuildPayload(models.QRTypeVCard, vcardJSON, "")
	if err != nil {
		t.Fatalf("failed building vCard payload: %v", err)
	}
	if !strings.Contains(vcardPayload, "BEGIN:VCARD") || !strings.Contains(vcardPayload, "N:Doe;Jane;;;") {
		t.Fatalf("malformed vCard payload: %s", vcardPayload)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
