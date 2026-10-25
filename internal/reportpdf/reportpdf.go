// Package reportpdf renders an analytics report as a printable PDF.
package reportpdf

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/jung-kurt/gofpdf"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/qr"
	"github.com/bcrowe306/linksy/internal/service"
)

// Series colors match the web charts (categorical slots 1 and 2, light mode).
var (
	clickRGB = [3]int{0x2a, 0x78, 0xd6}
	scanRGB  = [3]int{0xeb, 0x68, 0x34}
	inkRGB   = [3]int{17, 17, 20}
	mutedRGB = [3]int{110, 110, 120}
	gridRGB  = [3]int{225, 225, 230}
)

func compact(n int64) string {
	switch {
	case n >= 1_000_000:
		return strconv.FormatFloat(float64(n)/1e6, 'f', 1, 64) + "M"
	case n >= 10_000:
		return strconv.FormatFloat(float64(n)/1e3, 'f', 1, 64) + "K"
	}
	return strconv.FormatInt(n, 10)
}

func delta(cur, prev int64) string {
	if prev == 0 {
		if cur == 0 {
			return "no change"
		}
		return "new"
	}
	pct := float64(cur-prev) / float64(prev) * 100
	return fmt.Sprintf("%+.0f%% vs previous period", pct)
}

// Render builds the PDF for a report. subject names the scope ("All assets" or an asset title).
func Render(r *service.Report, workspace, subject string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("lynqr", "", qr.LabelFontTTF)
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetTitle("Lynqr analytics report", true)
	pdf.SetCreator("Lynqr", true)
	pdf.AddPage()

	text := func(size float64, rgb [3]int) {
		pdf.SetFont("Helvetica", "", size)
		pdf.SetTextColor(rgb[0], rgb[1], rgb[2])
	}
	heading := func(s string) {
		pdf.Ln(4)
		pdf.SetFont("lynqr", "", 12)
		pdf.SetTextColor(inkRGB[0], inkRGB[1], inkRGB[2])
		pdf.CellFormat(0, 8, s, "", 1, "L", false, 0, "")
	}

	pdf.SetFont("lynqr", "", 20)
	pdf.SetTextColor(inkRGB[0], inkRGB[1], inkRGB[2])
	pdf.CellFormat(0, 10, "Analytics report", "", 1, "L", false, 0, "")
	text(10, mutedRGB)
	pdf.CellFormat(0, 6, fmt.Sprintf("%s · %s · %s – %s (%s)", workspace, subject,
		r.From.Format("Jan 2, 2006"), r.To.Format("Jan 2, 2006 15:04"), r.Timezone), "", 1, "L", false, 0, "")

	// KPI tiles
	pdf.Ln(4)
	kpis := []struct {
		label     string
		cur, prev int64
	}{
		{"Total engagements", r.Totals.Total, r.Previous.Total},
		{"Link clicks", r.Totals.Clicks, r.Previous.Clicks},
		{"QR scans", r.Totals.Scans, r.Previous.Scans},
		{"Unique visitors", r.Totals.Visitors, r.Previous.Visitors},
	}
	x0, y0 := pdf.GetX(), pdf.GetY()
	w := 45.0
	for i, k := range kpis {
		x := x0 + float64(i)*(w+2)
		pdf.SetDrawColor(gridRGB[0], gridRGB[1], gridRGB[2])
		pdf.RoundedRect(x, y0, w, 24, 2, "1234", "D")
		pdf.SetXY(x+3, y0+3)
		text(8, mutedRGB)
		pdf.CellFormat(w-6, 4, k.label, "", 2, "L", false, 0, "")
		pdf.SetX(x + 3)
		pdf.SetFont("lynqr", "", 16)
		pdf.SetTextColor(inkRGB[0], inkRGB[1], inkRGB[2])
		pdf.CellFormat(w-6, 9, compact(k.cur), "", 2, "L", false, 0, "")
		pdf.SetX(x + 3)
		text(7, mutedRGB)
		pdf.CellFormat(w-6, 4, delta(k.cur, k.prev), "", 0, "L", false, 0, "")
	}
	pdf.SetXY(x0, y0+28)

	// Trend chart: two lines on one axis
	heading("Engagement over time")
	chartX, chartY, chartW, chartH := 22.0, pdf.GetY()+2, 170.0, 55.0
	var maxV int64 = 1
	for _, p := range r.Series {
		if p.Clicks > maxV {
			maxV = p.Clicks
		}
		if p.Scans > maxV {
			maxV = p.Scans
		}
	}
	pdf.SetLineWidth(0.15)
	for i := 0; i <= 4; i++ {
		y := chartY + chartH - chartH*float64(i)/4
		pdf.SetDrawColor(gridRGB[0], gridRGB[1], gridRGB[2])
		pdf.Line(chartX, y, chartX+chartW, y)
		text(7, mutedRGB)
		pdf.SetXY(chartX-12, y-2)
		pdf.CellFormat(10, 4, compact(maxV*int64(i)/4), "", 0, "R", false, 0, "")
	}
	n := len(r.Series)
	plot := func(rgb [3]int, val func(service.SeriesPoint) int64) {
		if n < 2 {
			return
		}
		pdf.SetDrawColor(rgb[0], rgb[1], rgb[2])
		pdf.SetLineWidth(0.6)
		for i := 1; i < n; i++ {
			x1 := chartX + chartW*float64(i-1)/float64(n-1)
			x2 := chartX + chartW*float64(i)/float64(n-1)
			y1 := chartY + chartH - chartH*float64(val(r.Series[i-1]))/float64(maxV)
			y2 := chartY + chartH - chartH*float64(val(r.Series[i]))/float64(maxV)
			pdf.Line(x1, y1, x2, y2)
		}
	}
	plot(clickRGB, func(p service.SeriesPoint) int64 { return p.Clicks })
	plot(scanRGB, func(p service.SeriesPoint) int64 { return p.Scans })
	if n > 0 {
		text(7, mutedRGB)
		layout := "Jan 2"
		if r.Granularity == "hour" {
			layout = "Jan 2 15:04"
		}
		pdf.SetXY(chartX, chartY+chartH+1)
		pdf.CellFormat(40, 4, r.Series[0].T.Format(layout), "", 0, "L", false, 0, "")
		pdf.SetXY(chartX+chartW-40, chartY+chartH+1)
		pdf.CellFormat(40, 4, r.Series[n-1].T.Format(layout), "", 0, "R", false, 0, "")
	}
	// Legend
	ly := chartY + chartH + 7
	for i, l := range []struct {
		name string
		rgb  [3]int
	}{{"Link clicks", clickRGB}, {"QR scans", scanRGB}} {
		lx := chartX + float64(i)*35
		pdf.SetDrawColor(l.rgb[0], l.rgb[1], l.rgb[2])
		pdf.SetLineWidth(0.8)
		pdf.Line(lx, ly+2, lx+6, ly+2)
		text(8, inkRGB)
		pdf.SetXY(lx+8, ly)
		pdf.CellFormat(25, 4, l.name, "", 0, "L", false, 0, "")
	}
	pdf.SetXY(15, ly+8)

	table := func(title string, cols []string, rows [][]string) {
		if len(rows) == 0 {
			return
		}
		heading(title)
		widths := []float64{110, 70}
		if len(cols) == 3 {
			widths = []float64{90, 50, 40}
		}
		text(8, mutedRGB)
		for i, c := range cols {
			align := "L"
			if i == len(cols)-1 {
				align = "R"
			}
			pdf.CellFormat(widths[i], 6, c, "B", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
		for _, row := range rows {
			pdf.SetFont("lynqr", "", 9)
			pdf.SetTextColor(inkRGB[0], inkRGB[1], inkRGB[2])
			for i, c := range row {
				align := "L"
				if i == len(row)-1 {
					align = "R"
				}
				pdf.CellFormat(widths[i], 6, c, "", 0, align, false, 0, "")
			}
			pdf.Ln(-1)
		}
	}

	var rows [][]string
	for _, a := range r.TopLinks {
		rows = append(rows, []string{a.Title, "/s/" + a.Label, compact(a.Count)})
	}
	table("Top links", []string{"Link", "Short URL", "Clicks"}, rows)
	rows = nil
	for _, a := range r.TopQRCodes {
		rows = append(rows, []string{a.Title, a.Label, compact(a.Count)})
	}
	table("Top QR codes", []string{"QR code", "Type", "Scans"}, rows)
	for _, sec := range []struct {
		title, col string
		get        func() [][]string
	}{
		{"Countries", "Country", func() [][]string { return nameRows(r.Countries, 10) }},
		{"Devices", "Device", func() [][]string { return nameRows(r.Devices, 10) }},
		{"Browsers", "Browser", func() [][]string { return nameRows(r.Browsers, 10) }},
		{"Traffic channels", "Channel", func() [][]string { return nameRows(r.Channels, 10) }},
		{"Referrers", "Referrer", func() [][]string { return nameRows(r.Referrers, 10) }},
	} {
		table(sec.title, []string{sec.col, "Engagements"}, sec.get())
	}

	text(7, mutedRGB)
	pdf.Ln(6)
	pdf.CellFormat(0, 5, "Generated by Lynqr · bot and link-preview traffic excluded", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func nameRows(list []models.NameCount, limit int) [][]string {
	rows := [][]string{}
	for i, n := range list {
		if i >= limit {
			break
		}
		rows = append(rows, []string{n.Name, compact(n.Count)})
	}
	return rows
}
