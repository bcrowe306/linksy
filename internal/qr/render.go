package qr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/jung-kurt/gofpdf"
	"golang.org/x/image/draw"

	"github.com/bcrowe306/linksy/internal/models"
)

// ----------------- SVG -----------------

func hex(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func num(f float64) string {
	s := strconv.FormatFloat(f, 'f', 2, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

func (p Path) svgD(b *strings.Builder) {
	for _, s := range p {
		b.WriteByte(byte(s.Op))
		for i, v := range s.Pts {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(num(v))
		}
	}
}

// encodePNG re-encodes images so only clean PNG data is ever embedded.
func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// SVG renders the scene as a standalone SVG document. All values written are
// numbers, validated colors, escaped text, or re-encoded PNG data.
func (sc *Scene) SVG() []byte {
	var b, defs strings.Builder
	gradID := 0
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 %s %s" width="%s" height="%s" shape-rendering="geometricPrecision">`,
		num(sc.W), num(sc.H), num(sc.W), num(sc.H))
	var body strings.Builder
	for _, el := range sc.Elements {
		switch e := el.(type) {
		case *Fill:
			if len(e.Parts) == 0 {
				continue
			}
			fill := hex(e.Paint.Color)
			if e.Paint.Kind != "solid" {
				gradID++
				id := "g" + strconv.Itoa(gradID)
				if e.Paint.Kind == "radial" {
					fmt.Fprintf(&defs, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%s" cy="%s" r="%s"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></radialGradient>`,
						id, num(e.Paint.X+e.Paint.W/2), num(e.Paint.Y+e.Paint.H/2), num(e.Paint.W*0.71), hex(e.Paint.Start), hex(e.Paint.End))
				} else {
					x0, y0, x1, y1 := e.Paint.LinearPoints()
					fmt.Fprintf(&defs, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient>`,
						id, num(x0), num(y0), num(x1), num(y1), hex(e.Paint.Start), hex(e.Paint.End))
				}
				fill = "url(#" + id + ")"
			}
			body.WriteString(`<path d="`)
			for _, p := range e.Parts {
				p.Path.svgD(&body)
			}
			body.WriteString(`" fill="` + fill + `"`)
			if e.EvenOdd {
				body.WriteString(` fill-rule="evenodd"`)
			}
			body.WriteString(`/>`)
		case *Text:
			fmt.Fprintf(&body, `<text x="%s" y="%s" text-anchor="middle" font-family="'Go', 'Inter', 'Helvetica Neue', Arial, sans-serif" font-weight="700" font-size="%s" letter-spacing="0.04em" fill="%s">%s</text>`,
				num(e.X), num(e.Y), num(e.Size), hex(e.Color), html.EscapeString(e.Value))
		case *Picture:
			fmt.Fprintf(&body, `<image x="%s" y="%s" width="%s" height="%s" preserveAspectRatio="xMidYMid meet" href="data:image/png;base64,%s"/>`,
				num(e.X), num(e.Y), num(e.W), num(e.H), base64.StdEncoding.EncodeToString(encodePNG(e.Img)))
		}
	}
	if defs.Len() > 0 {
		b.WriteString("<defs>" + defs.String() + "</defs>")
	}
	b.WriteString(body.String())
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

// ----------------- Raster -----------------

func (p Path) trace(dc *gg.Context, sx, sy, scale float64) {
	for _, s := range p {
		pt := s.Pts
		switch s.Op {
		case MoveTo:
			dc.MoveTo(sx+pt[0]*scale, sy+pt[1]*scale)
		case LineTo:
			dc.LineTo(sx+pt[0]*scale, sy+pt[1]*scale)
		case CubeTo:
			dc.CubicTo(sx+pt[0]*scale, sy+pt[1]*scale, sx+pt[2]*scale, sy+pt[3]*scale, sx+pt[4]*scale, sy+pt[5]*scale)
		case Close:
			dc.ClosePath()
		}
	}
}

func (p Paint) gg(scale float64) gg.Pattern {
	switch p.Kind {
	case "linear":
		x0, y0, x1, y1 := p.LinearPoints()
		g := gg.NewLinearGradient(x0*scale, y0*scale, x1*scale, y1*scale)
		g.AddColorStop(0, p.Start)
		g.AddColorStop(1, p.End)
		return g
	case "radial":
		cx, cy := (p.X+p.W/2)*scale, (p.Y+p.H/2)*scale
		g := gg.NewRadialGradient(cx, cy, 0, cx, cy, p.W*0.71*scale)
		g.AddColorStop(0, p.Start)
		g.AddColorStop(1, p.End)
		return g
	}
	return gg.NewSolidPattern(p.Color)
}

// Raster draws the scene into an RGBA image. background is painted first when
// non-nil (JPEG has no alpha channel).
func (sc *Scene) Raster(background *color.RGBA) image.Image {
	w, h := int(sc.W+0.5), int(sc.H+0.5)
	dc := gg.NewContext(w, h)
	if background != nil {
		dc.SetColor(*background)
		dc.Clear()
	}
	for _, el := range sc.Elements {
		switch e := el.(type) {
		case *Fill:
			if len(e.Parts) == 0 {
				continue
			}
			for _, p := range e.Parts {
				p.Path.trace(dc, 0, 0, 1)
			}
			if e.EvenOdd {
				dc.SetFillRule(gg.FillRuleEvenOdd)
			} else {
				dc.SetFillRule(gg.FillRuleWinding)
			}
			dc.SetFillStyle(e.Paint.gg(1))
			dc.Fill()
		case *Text:
			face := labelFace(e.Size)
			dc.SetFontFace(face)
			dc.SetColor(e.Color)
			dc.DrawStringAnchored(e.Value, e.X, e.Y, 0.5, 0)
			face.Close()
		case *Picture:
			dst := image.NewRGBA(image.Rect(0, 0, int(e.W+0.5), int(e.H+0.5)))
			draw.CatmullRom.Scale(dst, dst.Bounds(), e.Img, e.Img.Bounds(), draw.Over, nil)
			dc.DrawImage(dst, int(e.X+0.5), int(e.Y+0.5))
		}
	}
	return dc.Image()
}

// ----------------- PDF -----------------

// PDFLayout selects the page arrangement for PDF exports.
type PDFLayout string

const (
	PDFSheet PDFLayout = "sheet" // A4 page with title and scan prompt
	PDFCard  PDFLayout = "card"  // page sized to the code itself (stickers, print shops)
)

func (p Path) tracePDF(pdf *gofpdf.Fpdf, ox, oy, s float64) {
	for _, seg := range p {
		pt := seg.Pts
		switch seg.Op {
		case MoveTo:
			pdf.MoveTo(ox+pt[0]*s, oy+pt[1]*s)
		case LineTo:
			pdf.LineTo(ox+pt[0]*s, oy+pt[1]*s)
		case CubeTo:
			pdf.CurveBezierCubicTo(ox+pt[0]*s, oy+pt[1]*s, ox+pt[2]*s, oy+pt[3]*s, ox+pt[4]*s, oy+pt[5]*s)
		case Close:
			pdf.ClosePath()
		}
	}
}

// drawPDF draws the scene as vector shapes at (ox, oy) with the given width in mm.
func (sc *Scene) drawPDF(pdf *gofpdf.Fpdf, ox, oy, widthMM float64) {
	s := widthMM / sc.W
	pdf.AddUTF8FontFromBytes("lynqrlabel", "", LabelFontTTF)
	imgN := 0
	for _, el := range sc.Elements {
		switch e := el.(type) {
		case *Fill:
			style := "F"
			if e.EvenOdd {
				style = "F*"
			}
			if e.Paint.Kind == "solid" {
				c := e.Paint.Color
				pdf.SetFillColor(int(c.R), int(c.G), int(c.B))
				for _, p := range e.Parts {
					p.Path.tracePDF(pdf, ox, oy, s)
				}
				if len(e.Parts) > 0 {
					pdf.DrawPath(style)
				}
				continue
			}
			// PDF gradients can't be clipped to arbitrary paths here, so each
			// module takes the gradient color at its center (visually identical
			// at print sizes).
			for _, p := range e.Parts {
				c := e.Paint.At(p.Cx, p.Cy)
				pdf.SetFillColor(int(c.R), int(c.G), int(c.B))
				p.Path.tracePDF(pdf, ox, oy, s)
				pdf.DrawPath(style)
			}
		case *Text:
			sizeMM := e.Size * s
			pdf.SetFont("lynqrlabel", "", sizeMM/0.3528)
			pdf.SetTextColor(int(e.Color.R), int(e.Color.G), int(e.Color.B))
			pdf.Text(ox+e.X*s-pdf.GetStringWidth(e.Value)/2, oy+e.Y*s, e.Value)
		case *Picture:
			imgN++
			name := "logo" + strconv.Itoa(imgN)
			opts := gofpdf.ImageOptions{ImageType: "PNG"}
			pdf.RegisterImageOptionsReader(name, opts, bytes.NewReader(encodePNG(e.Img)))
			pdf.ImageOptions(name, ox+e.X*s, oy+e.Y*s, e.W*s, e.H*s, false, opts, 0, "")
		}
	}
}

// PDF renders a print-ready vector document.
func (sc *Scene) PDF(title string, layout PDFLayout) ([]byte, error) {
	var pdf *gofpdf.Fpdf
	if layout == PDFCard {
		widthMM := 60.0
		heightMM := widthMM * sc.H / sc.W
		pdf = gofpdf.NewCustom(&gofpdf.InitType{UnitStr: "mm", Size: gofpdf.SizeType{Wd: widthMM + 10, Ht: heightMM + 10}})
		pdf.SetMargins(0, 0, 0)
		pdf.SetAutoPageBreak(false, 0)
		pdf.AddPage()
		sc.drawPDF(pdf, 5, 5, widthMM)
	} else {
		pdf = gofpdf.New("P", "mm", "A4", "")
		pdf.SetAutoPageBreak(false, 0)
		pdf.AddPage()
		pdf.AddUTF8FontFromBytes("lynqrlabel", "", LabelFontTTF)
		if title == "" {
			title = "QR Code"
		}
		pdf.SetFont("lynqrlabel", "", 22)
		pdf.SetTextColor(17, 17, 20)
		pdf.SetXY(10, 24)
		pdf.CellFormat(190, 12, title, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 11)
		pdf.SetTextColor(100, 100, 110)
		pdf.CellFormat(190, 7, "Point your phone camera at the code to open it", "", 1, "C", false, 0, "")

		maxW, maxH := 130.0, 190.0
		widthMM := maxW
		if h := widthMM * sc.H / sc.W; h > maxH {
			widthMM = maxH * sc.W / sc.H
		}
		sc.drawPDF(pdf, (210-widthMM)/2, 52, widthMM)

		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(150, 150, 160)
		pdf.SetXY(10, 282)
		pdf.CellFormat(190, 5, "Created with Lynqr", "", 0, "C", false, 0, "")
	}
	pdf.SetTitle(title, true)
	pdf.SetCreator("Lynqr", true)
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed generating PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// ----------------- Public API -----------------

// RenderSVG renders a styled QR code as SVG.
func RenderSVG(content string, size int, style models.QRStyle) ([]byte, error) {
	sc, err := BuildScene(content, size, style)
	if err != nil {
		return nil, err
	}
	return sc.SVG(), nil
}

// GenerateImage renders a styled QR code to an image.
func GenerateImage(content string, size int, style models.QRStyle) (image.Image, error) {
	sc, err := BuildScene(content, size, style)
	if err != nil {
		return nil, err
	}
	return sc.Raster(nil), nil
}

// RenderPNG renders a styled QR code as PNG (transparent where unpainted).
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

// RenderJPEG renders a styled QR code as JPEG on the style's background color.
func RenderJPEG(content string, size int, style models.QRStyle) ([]byte, error) {
	st := NormalizeStyle(style)
	bg := ParseHexColor(st.BgColor, color.RGBA{255, 255, 255, 255})
	if st.FrameStyle != "none" {
		bg = ParseHexColor(st.FrameBgColor, bg)
	}
	st.IsTransparent = false
	sc, err := BuildScene(content, size, st)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, sc.Raster(&bg), &jpeg.Options{Quality: 95}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderPDF renders a print-ready vector PDF.
func RenderPDF(content, title string, style models.QRStyle, layout PDFLayout) ([]byte, error) {
	sc, err := BuildScene(content, 512, style)
	if err != nil {
		return nil, err
	}
	return sc.PDF(title, layout)
}
