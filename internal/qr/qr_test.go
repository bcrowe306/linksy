package qr

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
)

const sampleURL = "https://lynqr.app/q/Ab3dE9x"

func samplePNGDataURI(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{220, 38, 38, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func mustDecode(t *testing.T, content string, st models.QRStyle, label string) {
	t.Helper()
	sc, err := BuildScene(content, 640, st)
	if err != nil {
		t.Fatalf("%s: build: %v", label, err)
	}
	got, err := decodeImage(flatten(sc.Raster(nil), color.RGBA{255, 255, 255, 255}))
	if err != nil || got != content {
		t.Fatalf("%s: rendered code does not scan (got %q, err %v)", label, got, err)
	}
}

func TestEveryShapeScans(t *testing.T) {
	for _, dot := range DotStyles {
		for _, eye := range EyeOuterShapes {
			mustDecode(t, sampleURL, models.QRStyle{DotStyle: dot, EyeShape: eye, Margin: 2}, dot+"/"+eye)
		}
	}
	for _, inner := range EyeInnerShapes {
		mustDecode(t, sampleURL, models.QRStyle{EyeInnerShape: inner, Margin: 2}, "inner/"+inner)
	}
}

func TestFramesGradientsAndLogoScan(t *testing.T) {
	logo := samplePNGDataURI(t)
	for _, frame := range FrameStyles {
		st := models.QRStyle{
			FrameStyle: frame, LabelText: "SCAN ME", BorderWidth: 10, BorderRadius: 24, BorderColor: "#4f46e5",
			GradientType: "linear", GradientStart: "#1e1b4b", GradientEnd: "#4338ca", GradientAngle: 45,
			EyeColor: "#111827", EyeInnerColor: "#4f46e5", DotStyle: "extra-rounded", EyeShape: "rounded",
			LogoBase64: logo, LogoBgShape: "circle", Margin: 2,
		}
		mustDecode(t, sampleURL, st, "frame/"+frame)
	}
	mustDecode(t, sampleURL, models.QRStyle{GradientType: "radial", GradientStart: "#000000", GradientEnd: "#334155", Margin: 2}, "radial")
	mustDecode(t, sampleURL, models.QRStyle{IsTransparent: true, Margin: 3}, "transparent")
}

func TestFrameExtendsCanvas(t *testing.T) {
	plain, _ := BuildScene(sampleURL, 512, models.QRStyle{})
	banner, _ := BuildScene(sampleURL, 512, models.QRStyle{FrameStyle: "banner-bottom", LabelText: "Hi"})
	if plain.W != 512 || plain.H != 512 {
		t.Fatalf("plain scene should be 512x512, got %vx%v", plain.W, plain.H)
	}
	if !(banner.H > banner.W && banner.W > 512) {
		t.Fatalf("banner frame should add space and a label strip, got %vx%v", banner.W, banner.H)
	}
	var hasText bool
	for _, e := range banner.Elements {
		if tx, ok := e.(*Text); ok && tx.Value == "Hi" {
			hasText = true
		}
	}
	if !hasText {
		t.Fatal("banner label was not drawn")
	}
}

func TestSVGIsSafe(t *testing.T) {
	evil := models.QRStyle{
		FgColor:     `#000"/><script>alert(1)</script>`,
		BgColor:     `red;background:url(javascript:alert(1))`,
		LabelText:   `<script>alert("x")</script>`,
		FrameStyle:  "banner-bottom",
		LogoBase64:  `data:image/svg+xml;base64,PHN2Zz48c2NyaXB0PmFsZXJ0KDEpPC9zY3JpcHQ+PC9zdmc+`,
		BorderColor: `"><svg onload=alert(1)>`,
	}
	out, err := RenderSVG(sampleURL, 300, evil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, bad := range []string{"<script", "javascript:", "onload", "svg+xml"} {
		if strings.Contains(s, bad) {
			t.Fatalf("SVG contains unsafe content %q", bad)
		}
	}
	if !strings.Contains(s, "&lt;script&gt;") {
		t.Fatal("label text should be escaped, not dropped")
	}
}

func TestNormalizeStyle(t *testing.T) {
	st := NormalizeStyle(models.QRStyle{FgColor: "abc", DotStyle: "circle", FrameStyle: "box", Margin: 99, LogoSize: 5, LabelText: strings.Repeat("x", 80)})
	if st.FgColor != "#aabbcc" || st.DotStyle != "dots" || st.FrameStyle != "border" || st.Margin != 10 || st.LogoSize != 0.3 || len(st.LabelText) != MaxLabelLen {
		t.Fatalf("unexpected normalization: %+v", st)
	}
	if NormalizeStyle(models.QRStyle{LogoBase64: samplePNGDataURI(t), ErrorCorrection: "L"}).ErrorCorrection != "H" {
		t.Fatal("logos must force H error correction")
	}
	if NormalizeStyle(models.QRStyle{LogoURL: "http://169.254.169.254/"}).LogoURL != "" {
		t.Fatal("remote logo URLs must be dropped")
	}
}

func TestExportFormats(t *testing.T) {
	st := models.QRStyle{FrameStyle: "card", LabelText: "Menu", LogoBase64: samplePNGDataURI(t), BgColor: "#fef3c7"}

	p, err := RenderPNG(sampleURL, 1024, st)
	if err != nil || !bytes.HasPrefix(p, []byte("\x89PNG")) {
		t.Fatalf("png: %v", err)
	}
	img, _ := png.Decode(bytes.NewReader(p))
	if img.Bounds().Dx() <= 1024 {
		t.Fatalf("framed PNG should be wider than the code area, got %d", img.Bounds().Dx())
	}

	j, err := RenderJPEG(sampleURL, 512, models.QRStyle{BgColor: "#fef3c7", IsTransparent: true})
	if err != nil {
		t.Fatalf("jpeg: %v", err)
	}
	ji, _ := jpeg.Decode(bytes.NewReader(j))
	r, g, b, _ := ji.At(2, 2).RGBA()
	if r>>8 < 240 || g>>8 < 230 || b>>8 < 180 || b>>8 > 220 {
		t.Fatalf("JPEG should use the custom background, got %d,%d,%d", r>>8, g>>8, b>>8)
	}

	for _, layout := range []PDFLayout{PDFSheet, PDFCard} {
		pdf, err := RenderPDF(sampleURL, "Lunch menü", st, layout)
		if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatalf("pdf %s: %v", layout, err)
		}
	}

	if _, err := RenderPNG(strings.Repeat("x", 4000), 512, models.QRStyle{ErrorCorrection: "H"}); err == nil {
		t.Fatal("expected capacity error for oversized content")
	}
}

func TestVerify(t *testing.T) {
	good := Verify(sampleURL, models.QRStyle{Margin: 2})
	if !good.Scannable || len(good.Warnings) != 0 || good.Contrast < 20 {
		t.Fatalf("plain code should verify cleanly: %+v", good)
	}
	bad := Verify(sampleURL, models.QRStyle{FgColor: "#dddddd", BgColor: "#ffffff", Margin: 2})
	if len(bad.Warnings) == 0 || bad.Contrast > 2 {
		t.Fatalf("low contrast should warn: %+v", bad)
	}
}

func TestBuildPayloads(t *testing.T) {
	cases := []struct {
		typ      models.QRType
		raw      string
		target   string
		want     string
		contains []string
	}{
		{models.QRTypeWebsite, "", " https://example.com ", "https://example.com", nil},
		{models.QRTypeText, "hello world", "", "hello world", nil},
		{models.QRTypeWiFi, `{"ssid":"Cafe;Guest","password":"p:a\"ss","encryption":"WPA2"}`, "", `WIFI:T:WPA;S:Cafe\;Guest;P:p\:a\"ss;;`, nil},
		{models.QRTypeWiFi, `{"ssid":"Open","encryption":"nopass","hidden":true}`, "", "WIFI:T:nopass;S:Open;H:true;;", nil},
		{models.QRTypeWiFi, `{"ssid":"New","password":"x","encryption":"WPA3"}`, "", "WIFI:T:SAE;S:New;P:x;;", nil},
		{models.QRTypeEmail, `{"email":"hi@lynqr.app","subject":"Hello there","body":"a&b"}`, "", "mailto:hi@lynqr.app?subject=Hello%20there&body=a&b", nil},
		{models.QRTypeSMS, `{"phone":"+1 (555) 010-2000","message":"Hi"}`, "", "SMSTO:+15550102000:Hi", nil},
		{models.QRTypePhone, `{"phone":"+1 555 010 2000"}`, "", "tel:+15550102000", nil},
		{models.QRTypeApp, `{"app_store_url":"https://apps.apple.com/x"}`, "", "https://apps.apple.com/x", nil},
		{models.QRTypeVCard, `{"first_name":"Ada","last_name":"Lovelace","organization":"Analytical, Ltd","phone":"555-0100","email":"ada@example.com","city":"London"}`, "", "",
			[]string{"BEGIN:VCARD\r\nVERSION:3.0", "N:Lovelace;Ada;;;", "ORG:Analytical\\, Ltd", "TEL;TYPE=WORK,VOICE:5550100", "ADR;TYPE=WORK:;;;London;;;", "END:VCARD"}},
		{models.QRTypeVCard, `{"format":"vcard4","first_name":"Ada","mobile":"555"}`, "", "", []string{"VERSION:4.0", "TEL;TYPE=cell;VALUE=uri:tel:555"}},
		{models.QRTypeVCard, `{"format":"mecard","first_name":"Ada","last_name":"L","email":"a@b.co"}`, "", "MECARD:N:L,Ada;EMAIL:a@b.co;;", nil},
	}
	for _, c := range cases {
		got, err := BuildPayload(c.typ, c.raw, c.target)
		if err != nil {
			t.Errorf("%s: %v", c.typ, err)
			continue
		}
		if c.want != "" && got != c.want {
			t.Errorf("%s: got %q want %q", c.typ, got, c.want)
		}
		for _, sub := range c.contains {
			if !strings.Contains(got, sub) {
				t.Errorf("%s: %q missing %q", c.typ, got, sub)
			}
		}
	}

	invalid := []struct {
		typ models.QRType
		raw string
	}{
		{models.QRTypeWiFi, `{"password":"x"}`},
		{models.QRTypeEmail, `{"email":"not-an-email"}`},
		{models.QRTypeSMS, `{"phone":"abc"}`},
		{models.QRTypeVCard, `{"email":"a@b.co"}`},
		{models.QRTypeText, ""},
		{models.QRTypeText, strings.Repeat("x", MaxTextBytes+1)},
		{models.QRTypeApp, `{}`},
		{"bogus", "x"},
	}
	for _, c := range invalid {
		if _, err := BuildPayload(c.typ, c.raw, ""); err == nil {
			t.Errorf("%s %q: expected validation error", c.typ, c.raw)
		}
	}
}

func TestLabelStaysLegible(t *testing.T) {
	sc, _ := BuildScene(sampleURL, 512, models.QRStyle{FrameStyle: "card", LabelText: "WIFI", LabelColor: "#ffffff", FrameBgColor: "#ffffff", BorderColor: "#4f46e5"})
	for _, e := range sc.Elements {
		if tx, ok := e.(*Text); ok {
			if ContrastRatio(tx.Color, color.RGBA{255, 255, 255, 255}) < 3 {
				t.Fatalf("label color %v is unreadable on a white card", tx.Color)
			}
			return
		}
	}
	t.Fatal("no label drawn")
}
