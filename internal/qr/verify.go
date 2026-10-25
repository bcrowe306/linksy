package qr

import (
	"fmt"
	"image"
	"image/color"

	"github.com/makiuchi-d/gozxing"
	gozqr "github.com/makiuchi-d/gozxing/qrcode"

	"github.com/bcrowe306/linksy/internal/models"
)

// Check summarizes whether a styled code is likely to scan reliably.
type Check struct {
	Scannable bool     `json:"scannable"`
	Contrast  float64  `json:"contrast"`
	Modules   int      `json:"modules"`
	Warnings  []string `json:"warnings"`
}

// decodeImage tries to read a QR code from an image.
func decodeImage(img image.Image) (string, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}
	hints := map[gozxing.DecodeHintType]interface{}{gozxing.DecodeHintType_TRY_HARDER: true}
	res, err := gozqr.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		return "", err
	}
	return res.GetText(), nil
}

// flatten composites an image over a background so transparent codes are
// checked the way a camera would see them on a light surface.
func flatten(img image.Image, bg color.RGBA) image.Image {
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			af := float64(a) / 0xffff
			mix := func(c uint32, base uint8) uint8 {
				return uint8(float64(c>>8) + float64(base)*(1-af))
			}
			out.Set(x, y, color.RGBA{mix(r, bg.R), mix(g, bg.G), mix(bl, bg.B), 255})
		}
	}
	return out
}

// Verify renders the code, decodes it, and reports contrast and capacity issues.
func Verify(content string, style models.QRStyle) Check {
	st := NormalizeStyle(style)
	chk := Check{Warnings: []string{}}

	fg := ParseHexColor(st.FgColor, color.RGBA{0, 0, 0, 255})
	bg := ParseHexColor(st.BgColor, color.RGBA{255, 255, 255, 255})
	if st.IsTransparent {
		bg = color.RGBA{255, 255, 255, 255}
	}
	worst := ContrastRatio(fg, bg)
	if st.GradientType != "none" {
		for _, c := range []string{st.GradientStart, st.GradientEnd} {
			if r := ContrastRatio(ParseHexColor(c, fg), bg); r < worst {
				worst = r
			}
		}
	}
	for _, c := range []string{st.EyeColor, st.EyeInnerColor} {
		if c != "" {
			if r := ContrastRatio(ParseHexColor(c, fg), bg); r < worst {
				worst = r
			}
		}
	}
	chk.Contrast = float64(int(worst*10)) / 10
	if worst < 3 {
		chk.Warnings = append(chk.Warnings, fmt.Sprintf("Low contrast (%.1f:1). Aim for at least 4:1 between the code and its background.", worst))
	}
	if relativeLuminance(fg) > relativeLuminance(bg) {
		chk.Warnings = append(chk.Warnings, "Light code on a dark background (inverted) is not supported by some older scanners.")
	}

	bm, err := Matrix(content, st.ErrorCorrection)
	if err != nil {
		chk.Warnings = append(chk.Warnings, err.Error())
		return chk
	}
	chk.Modules = len(bm)
	if chk.Modules > 73 {
		chk.Warnings = append(chk.Warnings, "This code is very dense. Keep printed codes at least 3 cm wide, or shorten the content.")
	}
	if st.Margin < 2 && st.FrameStyle == "none" {
		chk.Warnings = append(chk.Warnings, "A quiet zone of at least 2 modules around the code improves scanning.")
	}

	sc, err := BuildScene(content, 640, st)
	if err != nil {
		chk.Warnings = append(chk.Warnings, err.Error())
		return chk
	}
	img := flatten(sc.Raster(nil), color.RGBA{255, 255, 255, 255})
	text, err := decodeImage(img)
	chk.Scannable = err == nil && text == content
	if !chk.Scannable {
		chk.Warnings = append(chk.Warnings, "We could not decode this design. Increase contrast, shrink the logo, or raise error correction.")
	}
	return chk
}
