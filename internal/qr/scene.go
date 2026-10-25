package qr

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/bcrowe306/linksy/internal/models"
)

// A Scene is a resolution-independent drawing of a styled QR code. The SVG,
// raster (PNG/JPEG) and PDF renderers all draw the same Scene, so every
// export matches the studio preview exactly.
type Scene struct {
	W, H     float64
	Elements []Element
}

// Element is one of *Fill, *Text or *Picture.
type Element interface{ isElement() }

// Fill paints one or more closed paths with a single paint.
type Fill struct {
	Parts   []Part
	Paint   Paint
	EvenOdd bool
}

// Part is a sub-shape; its center lets backends without gradient support
// approximate gradients per module (PDF).
type Part struct {
	Path   Path
	Cx, Cy float64
}

type Text struct {
	X, Y  float64 // anchor: horizontal center, baseline
	Size  float64
	Value string
	Color color.RGBA
}

type Picture struct {
	X, Y, W, H float64
	Img        image.Image
}

func (*Fill) isElement()    {}
func (*Text) isElement()    {}
func (*Picture) isElement() {}

// Paint is a solid color or a gradient across a box.
type Paint struct {
	Kind       string // "solid", "linear", "radial"
	Color      color.RGBA
	Start, End color.RGBA
	Angle      float64
	X, Y, W, H float64
}

func solid(c color.RGBA) Paint { return Paint{Kind: "solid", Color: c} }

// At returns the paint color at a point (used for per-part gradient approximation).
func (p Paint) At(x, y float64) color.RGBA {
	if p.Kind == "solid" {
		return p.Color
	}
	var t float64
	if p.Kind == "radial" {
		cx, cy := p.X+p.W/2, p.Y+p.H/2
		r := math.Hypot(p.W, p.H) / 2
		t = math.Hypot(x-cx, y-cy) / r
	} else {
		x0, y0, x1, y1 := p.LinearPoints()
		dx, dy := x1-x0, y1-y0
		t = ((x-x0)*dx + (y-y0)*dy) / (dx*dx + dy*dy)
	}
	t = clampFloat(t, 0, 1)
	lerp := func(a, b uint8) uint8 { return uint8(math.Round(float64(a)*(1-t) + float64(b)*t)) }
	return color.RGBA{lerp(p.Start.R, p.End.R), lerp(p.Start.G, p.End.G), lerp(p.Start.B, p.End.B), 255}
}

// LinearPoints returns the gradient line for the paint's angle across its box
// (0° = left→right, 90° = top→bottom).
func (p Paint) LinearPoints() (x0, y0, x1, y1 float64) {
	rad := p.Angle * math.Pi / 180
	cx, cy := p.X+p.W/2, p.Y+p.H/2
	half := (math.Abs(math.Cos(rad))*p.W + math.Abs(math.Sin(rad))*p.H) / 2
	dx, dy := math.Cos(rad)*half, math.Sin(rad)*half
	return cx - dx, cy - dy, cx + dx, cy + dy
}

// ----------------- Paths -----------------

type SegOp byte

const (
	MoveTo SegOp = 'M'
	LineTo SegOp = 'L'
	CubeTo SegOp = 'C'
	Close  SegOp = 'Z'
)

type Seg struct {
	Op  SegOp
	Pts []float64
}

type Path []Seg

const kappa = 0.5522847498

// RoundRect builds a rectangle with per-corner radii (tl, tr, br, bl).
func RoundRect(x, y, w, h, tl, tr, br, bl float64) Path {
	lim := math.Min(w, h) / 2
	tl, tr, br, bl = math.Min(tl, lim), math.Min(tr, lim), math.Min(br, lim), math.Min(bl, lim)
	p := Path{{MoveTo, []float64{x + tl, y}}, {LineTo, []float64{x + w - tr, y}}}
	if tr > 0 {
		p = append(p, Seg{CubeTo, []float64{x + w - tr + tr*kappa, y, x + w, y + tr - tr*kappa, x + w, y + tr}})
	}
	p = append(p, Seg{LineTo, []float64{x + w, y + h - br}})
	if br > 0 {
		p = append(p, Seg{CubeTo, []float64{x + w, y + h - br + br*kappa, x + w - br + br*kappa, y + h, x + w - br, y + h}})
	}
	p = append(p, Seg{LineTo, []float64{x + bl, y + h}})
	if bl > 0 {
		p = append(p, Seg{CubeTo, []float64{x + bl - bl*kappa, y + h, x, y + h - bl + bl*kappa, x, y + h - bl}})
	}
	p = append(p, Seg{LineTo, []float64{x, y + tl}})
	if tl > 0 {
		p = append(p, Seg{CubeTo, []float64{x, y + tl - tl*kappa, x + tl - tl*kappa, y, x + tl, y}})
	}
	return append(p, Seg{Op: Close})
}

func Rect(x, y, w, h float64) Path { return RoundRect(x, y, w, h, 0, 0, 0, 0) }

func Circle(cx, cy, r float64) Path {
	k := r * kappa
	return Path{
		{MoveTo, []float64{cx + r, cy}},
		{CubeTo, []float64{cx + r, cy + k, cx + k, cy + r, cx, cy + r}},
		{CubeTo, []float64{cx - k, cy + r, cx - r, cy + k, cx - r, cy}},
		{CubeTo, []float64{cx - r, cy - k, cx - k, cy - r, cx, cy - r}},
		{CubeTo, []float64{cx + k, cy - r, cx + r, cy - k, cx + r, cy}},
		{Op: Close},
	}
}

// shapePath draws a named shape inside a square box of side s.
func shapePath(shape string, x, y, s float64) Path {
	switch shape {
	case "circle":
		return Circle(x+s/2, y+s/2, s/2)
	case "rounded":
		r := s * 0.22
		return RoundRect(x, y, s, s, r, r, r, r)
	case "extra-rounded":
		r := s * 0.36
		return RoundRect(x, y, s, s, r, r, r, r)
	case "leaf":
		r := s * 0.5
		return RoundRect(x, y, s, s, r, 0, r, 0)
	}
	return Rect(x, y, s, s)
}

// ----------------- Fonts -----------------

var labelFont = func() *opentype.Font {
	f, err := opentype.Parse(gobold.TTF)
	if err != nil {
		panic(err)
	}
	return f
}()

// LabelFontTTF is the font used for frame labels in raster and PDF output.
var LabelFontTTF = gobold.TTF

func labelFace(size float64) font.Face {
	face, _ := opentype.NewFace(labelFont, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	return face
}

// MeasureLabel returns the advance width of s at the given pixel size.
func MeasureLabel(s string, size float64) float64 {
	face := labelFace(size)
	defer face.Close()
	return float64(font.MeasureString(face, s)) / float64(fixed.I(1))
}

// ----------------- Scene construction -----------------

func recoveryLevel(ec string) qrcode.RecoveryLevel {
	switch ec {
	case "L":
		return qrcode.Low
	case "Q":
		return qrcode.High
	case "H":
		return qrcode.Highest
	}
	return qrcode.Medium
}

// Matrix encodes content into a QR module matrix (without quiet zone).
func Matrix(content string, ec string) ([][]bool, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("QR content is empty")
	}
	q, err := qrcode.New(content, recoveryLevel(ec))
	if err != nil {
		return nil, fmt.Errorf("content is too long to encode at this error-correction level: %w", err)
	}
	q.DisableBorder = true
	return q.Bitmap(), nil
}

// BuildScene lays out a styled QR code whose code area (including the quiet
// zone) is size×size pixels; frames extend the canvas around it.
func BuildScene(content string, size int, raw models.QRStyle) (*Scene, error) {
	st := NormalizeStyle(raw)
	if size < MinSize {
		size = MinSize
	}
	if size > MaxSize {
		size = MaxSize
	}
	bm, err := Matrix(content, st.ErrorCorrection)
	if err != nil {
		return nil, err
	}
	n := len(bm)
	Q := float64(size)
	mod := Q / float64(n+2*st.Margin)

	// Frame geometry decides where the code area sits on the canvas.
	fr := layoutFrame(st, Q)
	ox, oy := fr.qrX, fr.qrY
	sc := &Scene{W: fr.w, H: fr.h}
	sc.Elements = append(sc.Elements, fr.under...)

	fg := ParseHexColor(st.FgColor, color.RGBA{0, 0, 0, 255})
	bg := ParseHexColor(st.BgColor, color.RGBA{255, 255, 255, 255})

	if !st.IsTransparent {
		r := 0.0
		if st.FrameStyle == "none" {
			r = float64(st.BorderRadius) * Q / 512
		}
		sc.Elements = append(sc.Elements, &Fill{Parts: []Part{{Path: RoundRect(ox, oy, Q, Q, r, r, r, r)}}, Paint: solid(bg)})
	}

	codeX, codeY := ox+float64(st.Margin)*mod, oy+float64(st.Margin)*mod
	codeW := float64(n) * mod

	dotPaint := solid(fg)
	if st.GradientType != "none" {
		dotPaint = Paint{Kind: st.GradientType, Start: ParseHexColor(st.GradientStart, fg), End: ParseHexColor(st.GradientEnd, fg),
			Angle: float64(st.GradientAngle), X: codeX, Y: codeY, W: codeW, H: codeW}
	}

	// Logo excavation: clear modules under the logo so it never overlaps dots.
	var logo image.Image
	cleared := func(int, int) bool { return false }
	var logoBox [4]float64
	if st.LogoBase64 != "" {
		if img, err := DecodeLogo(st.LogoBase64); err == nil {
			logo = img
			ls := codeW * st.LogoSize
			pad := float64(st.LogoPadding) * Q / 512
			lx, ly := codeX+(codeW-ls)/2, codeY+(codeW-ls)/2
			logoBox = [4]float64{lx, ly, ls, ls}
			clear := ls + 2*pad
			cx0, cy0 := codeX+(codeW-clear)/2, codeY+(codeW-clear)/2
			cleared = func(mx, my int) bool {
				x, y := float64(mx)*mod+mod/2+codeX, float64(my)*mod+mod/2+codeY
				if st.LogoBgShape == "circle" {
					return math.Hypot(x-(cx0+clear/2), y-(cy0+clear/2)) < clear/2+mod*0.3
				}
				return x > cx0-mod*0.3 && x < cx0+clear+mod*0.3 && y > cy0-mod*0.3 && y < cy0+clear+mod*0.3
			}
		}
	}

	finder := func(mx, my int) bool {
		return (mx < 7 && my < 7) || (mx >= n-7 && my < 7) || (mx < 7 && my >= n-7)
	}
	on := func(mx, my int) bool {
		if mx < 0 || my < 0 || mx >= n || my >= n {
			return false
		}
		return bm[my][mx] && !finder(mx, my) && !cleared(mx, my)
	}

	dots := &Fill{Paint: dotPaint}
	for my := 0; my < n; my++ {
		for mx := 0; mx < n; mx++ {
			if !on(mx, my) {
				continue
			}
			x, y := codeX+float64(mx)*mod, codeY+float64(my)*mod
			dots.Parts = append(dots.Parts, Part{Path: dotPath(st.DotStyle, x, y, mod, mx, my, on), Cx: x + mod/2, Cy: y + mod/2})
		}
	}
	sc.Elements = append(sc.Elements, dots)

	// Finder "eyes": outer ring (even-odd) + inner ball.
	outerPaint, innerPaint := dotPaint, dotPaint
	if st.EyeColor != "" {
		outerPaint = solid(ParseHexColor(st.EyeColor, fg))
		innerPaint = outerPaint
	}
	if st.EyeInnerColor != "" {
		innerPaint = solid(ParseHexColor(st.EyeInnerColor, fg))
	}
	outer := &Fill{Paint: outerPaint, EvenOdd: true}
	inner := &Fill{Paint: innerPaint}
	for _, o := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
		x, y := codeX+float64(o[0])*mod, codeY+float64(o[1])*mod
		ring := append(shapePath(st.EyeShape, x, y, 7*mod), ringHole(st.EyeShape, x+mod, y+mod, 5*mod)...)
		outer.Parts = append(outer.Parts, Part{Path: ring, Cx: x + 3.5*mod, Cy: y + 3.5*mod})
		inner.Parts = append(inner.Parts, Part{Path: shapePath(st.EyeInnerShape, x+2*mod, y+2*mod, 3*mod), Cx: x + 3.5*mod, Cy: y + 3.5*mod})
	}
	sc.Elements = append(sc.Elements, outer, inner)

	if logo != nil {
		lx, ly, ls := logoBox[0], logoBox[1], logoBox[2]
		if st.LogoBgShape != "none" {
			pad := float64(st.LogoPadding) * Q / 512
			bx, by, bs := lx-pad, ly-pad, ls+2*pad
			var p Path
			switch st.LogoBgShape {
			case "circle":
				p = Circle(bx+bs/2, by+bs/2, bs/2*1.1)
			case "rounded":
				r := bs * 0.22
				p = RoundRect(bx, by, bs, bs, r, r, r, r)
			default:
				p = Rect(bx, by, bs, bs)
			}
			sc.Elements = append(sc.Elements, &Fill{Parts: []Part{{Path: p}}, Paint: solid(bg)})
		}
		// Fit the logo inside its square while preserving aspect ratio.
		b := logo.Bounds()
		w, h := ls, ls
		if b.Dx() > b.Dy() {
			h = ls * float64(b.Dy()) / float64(b.Dx())
		} else if b.Dy() > b.Dx() {
			w = ls * float64(b.Dx()) / float64(b.Dy())
		}
		sc.Elements = append(sc.Elements, &Picture{X: lx + (ls-w)/2, Y: ly + (ls-h)/2, W: w, H: h, Img: logo})
	}

	sc.Elements = append(sc.Elements, fr.over...)
	return sc, nil
}

// ringHole returns the inner cut-out of a finder ring, matching the outer shape.
func ringHole(shape string, x, y, s float64) Path {
	switch shape {
	case "circle":
		return Circle(x+s/2, y+s/2, s/2)
	case "rounded":
		r := s * 0.16
		return RoundRect(x, y, s, s, r, r, r, r)
	case "extra-rounded":
		r := s * 0.3
		return RoundRect(x, y, s, s, r, r, r, r)
	case "leaf":
		r := s * 0.42
		return RoundRect(x, y, s, s, r, 0, r, 0)
	}
	return Rect(x, y, s, s)
}

// dotPath draws a data module, rounding corners that have no neighbors for
// the "connected" styles.
func dotPath(style string, x, y, m float64, mx, my int, on func(int, int) bool) Path {
	switch style {
	case "dots":
		return Circle(x+m/2, y+m/2, m*0.42)
	case "rounded":
		r := m * 0.3
		return RoundRect(x+m*0.04, y+m*0.04, m*0.92, m*0.92, r, r, r, r)
	case "extra-rounded", "classy":
		up, down, left, right := on(mx, my-1), on(mx, my+1), on(mx-1, my), on(mx+1, my)
		r := m / 2
		corner := func(a, b bool) float64 {
			if !a && !b {
				return r
			}
			return 0
		}
		tl, tr, br, bl := corner(up, left), corner(up, right), corner(down, right), corner(down, left)
		if style == "classy" {
			tr, bl = 0, 0
		}
		return RoundRect(x, y, m, m, tl, tr, br, bl)
	}
	// Slight overlap avoids hairline seams between adjacent squares when anti-aliased.
	return Rect(x-0.01*m, y-0.01*m, m*1.02, m*1.02)
}

// ----------------- Frames -----------------

type frameLayout struct {
	w, h     float64
	qrX, qrY float64
	under    []Element // drawn before the code
	over     []Element // drawn after (labels)
}

func layoutFrame(st models.QRStyle, Q float64) frameLayout {
	unit := Q / 512
	bw := float64(st.BorderWidth) * unit
	radius := float64(st.BorderRadius) * unit
	border := ParseHexColor(st.BorderColor, color.RGBA{0, 0, 0, 255})
	frameBg := ParseHexColor(st.FrameBgColor, color.RGBA{255, 255, 255, 255})
	labelColor := ParseHexColor(st.LabelColor, color.RGBA{255, 255, 255, 255})
	// The card frame prints its label on the frame background; banners print it
	// on the frame color. Keep the label legible if the chosen color would vanish.
	labelBg := border
	if st.FrameStyle == "card" {
		labelBg = frameBg
	}
	if ContrastRatio(labelColor, labelBg) < 2 {
		labelColor = color.RGBA{255, 255, 255, 255}
		if ContrastRatio(labelColor, labelBg) < 2 {
			labelColor = color.RGBA{17, 17, 20, 255}
			if st.FrameStyle == "card" && ContrastRatio(border, frameBg) >= 3 {
				labelColor = border
			}
		}
	}
	label := st.LabelText
	fontSize := 38 * unit * float64(st.LabelSize) / 100

	switch st.FrameStyle {
	case "border":
		if bw == 0 {
			bw = 8 * unit
		}
		pad := bw + 20*unit
		w := Q + 2*pad
		return frameLayout{w: w, h: w, qrX: pad, qrY: pad, under: []Element{
			&Fill{Parts: []Part{{Path: RoundRect(0, 0, w, w, radius, radius, radius, radius)}}, Paint: solid(border)},
			&Fill{Parts: []Part{{Path: insetRound(bw, bw, w-2*bw, w-2*bw, radius-bw)}}, Paint: solid(frameBg)},
		}}

	case "banner-bottom", "banner-top":
		if bw == 0 {
			bw = 10 * unit
		}
		if label == "" {
			label = "SCAN ME"
		}
		pad := bw + 14*unit
		banner := fontSize * 2.1 // label strip height
		F := Q + 2*pad           // bordered area around the code
		w := F
		h := F + banner - bw // banner shares one border edge
		frameY, textY := 0.0, F-bw+banner/2
		if st.FrameStyle == "banner-top" {
			frameY, textY = banner-bw, banner/2
		}
		return frameLayout{w: w, h: h, qrX: pad, qrY: frameY + pad,
			under: []Element{
				&Fill{Parts: []Part{{Path: RoundRect(0, 0, w, h, radius, radius, radius, radius)}}, Paint: solid(border)},
				&Fill{Parts: []Part{{Path: insetRound(bw, frameY+bw, w-2*bw, F-2*bw, radius-bw)}}, Paint: solid(frameBg)},
			},
			over: labelElements(label, w/2, textY, fontSize, w-2*pad, labelColor),
		}

	case "card":
		if label == "" {
			label = "SCAN ME"
		}
		pad := 36 * unit
		text := fontSize * 2.2
		w := Q + 2*pad
		h := Q + 2*pad + text
		under := []Element{}
		if bw > 0 {
			under = append(under,
				&Fill{Parts: []Part{{Path: RoundRect(0, 0, w, h, radius, radius, radius, radius)}}, Paint: solid(border)},
				&Fill{Parts: []Part{{Path: insetRound(bw, bw, w-2*bw, h-2*bw, radius-bw)}}, Paint: solid(frameBg)})
		} else {
			under = append(under, &Fill{Parts: []Part{{Path: RoundRect(0, 0, w, h, radius, radius, radius, radius)}}, Paint: solid(frameBg)})
		}
		return frameLayout{w: w, h: h, qrX: pad, qrY: pad, under: under,
			over: labelElements(label, w/2, Q+pad+text/2+pad/3, fontSize, w-2*pad, labelColor)}

	case "pill":
		if label == "" {
			label = "SCAN ME"
		}
		if bw == 0 {
			bw = 8 * unit
		}
		pad := bw + 18*unit
		pillH := fontSize * 1.9
		w := Q + 2*pad
		h := Q + 2*pad + pillH/2
		pillW := math.Min(MeasureLabel(label, fontSize)+fontSize*2.4, w-2*bw)
		px, py := (w-pillW)/2, Q+2*pad-pillH/2
		return frameLayout{w: w, h: h, qrX: pad, qrY: pad,
			under: []Element{
				&Fill{Parts: []Part{{Path: RoundRect(0, 0, w, Q+2*pad, radius, radius, radius, radius)}}, Paint: solid(border)},
				&Fill{Parts: []Part{{Path: insetRound(bw, bw, w-2*bw, Q+2*pad-2*bw, radius-bw)}}, Paint: solid(frameBg)},
			},
			over: append([]Element{&Fill{Parts: []Part{{Path: RoundRect(px, py, pillW, pillH, pillH/2, pillH/2, pillH/2, pillH/2)}}, Paint: solid(border)}},
				labelElements(label, w/2, py+pillH/2, fontSize, pillW-fontSize, labelColor)...),
		}
	}
	return frameLayout{w: Q, h: Q}
}

func insetRound(x, y, w, h, r float64) Path {
	if r < 0 {
		r = 0
	}
	return RoundRect(x, y, w, h, r, r, r, r)
}

// labelElements centers text vertically on cy, shrinking it to fit maxW.
func labelElements(label string, cx, cy, size, maxW float64, c color.RGBA) []Element {
	if label == "" {
		return nil
	}
	if w := MeasureLabel(label, size); w > maxW && w > 0 {
		size *= maxW / w
	}
	return []Element{&Text{X: cx, Y: cy + size*0.36, Size: size, Value: label, Color: c}}
}
