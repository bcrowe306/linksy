package email

import (
	"fmt"
	"html"
	"strings"
)

// Content is a transactional email body rendered with the Lynqr layout.
type Content struct {
	Preheader  string // inbox preview text
	Heading    string
	Paragraphs []string
	Button     *Button
	Details    [][2]string // label/value rows, e.g. browser and location
	Footnote   string      // small print under the button
	// Why the recipient got this, plus where to change it.
	Reason      string
	ManageURL   string
	Unsubscribe string // one-click unsubscribe URL (optional)
}

type Button struct {
	Label string
	URL   string
}

// Render returns the HTML and plain-text bodies.
func (c Content) Render() (htmlBody, text string) {
	var t strings.Builder
	t.WriteString(c.Heading + "\n\n")
	for _, p := range c.Paragraphs {
		t.WriteString(p + "\n\n")
	}
	for _, d := range c.Details {
		fmt.Fprintf(&t, "%s: %s\n", d[0], d[1])
	}
	if len(c.Details) > 0 {
		t.WriteString("\n")
	}
	if c.Button != nil {
		fmt.Fprintf(&t, "%s: %s\n\n", c.Button.Label, c.Button.URL)
	}
	if c.Footnote != "" {
		t.WriteString(c.Footnote + "\n\n")
	}
	t.WriteString("—\nLynqr · Dynamic QR codes & short links\n")
	if c.Reason != "" {
		t.WriteString(c.Reason + "\n")
	}
	if c.ManageURL != "" {
		t.WriteString("Email settings: " + c.ManageURL + "\n")
	}
	if c.Unsubscribe != "" {
		t.WriteString("Unsubscribe: " + c.Unsubscribe + "\n")
	}

	e := html.EscapeString
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>` + e(c.Heading) + `</title></head>`)
	b.WriteString(`<body style="margin:0;padding:0;background:#f4f4f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#18181b">`)
	if c.Preheader != "" {
		b.WriteString(`<div style="display:none;max-height:0;overflow:hidden;opacity:0">` + e(c.Preheader) + `</div>`)
	}
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f7;padding:32px 12px"><tr><td align="center">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px">`)
	b.WriteString(`<tr><td style="padding:0 8px 20px"><table role="presentation" cellpadding="0" cellspacing="0"><tr>` +
		`<td style="width:28px;height:28px;border-radius:8px;background:#6366f1;background:linear-gradient(135deg,#6366f1,#a855f7)"></td>` +
		`<td style="padding-left:10px;font-size:18px;font-weight:700;color:#18181b">Lynqr</td></tr></table></td></tr>`)
	b.WriteString(`<tr><td style="background:#ffffff;border:1px solid #e4e4e9;border-radius:16px;padding:32px">`)
	b.WriteString(`<h1 style="margin:0 0 16px;font-size:22px;line-height:1.3;color:#111114">` + e(c.Heading) + `</h1>`)
	for _, p := range c.Paragraphs {
		b.WriteString(`<p style="margin:0 0 14px;font-size:15px;line-height:1.6;color:#3f3f46">` + e(p) + `</p>`)
	}
	if len(c.Details) > 0 {
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:8px 0 18px;border:1px solid #e4e4e9;border-radius:12px">`)
		for i, d := range c.Details {
			border := "border-top:1px solid #e4e4e9;"
			if i == 0 {
				border = ""
			}
			b.WriteString(`<tr><td style="` + border + `padding:10px 14px;font-size:13px;color:#71717a;width:40%">` + e(d[0]) +
				`</td><td style="` + border + `padding:10px 14px;font-size:14px;color:#18181b;font-weight:600">` + e(d[1]) + `</td></tr>`)
		}
		b.WriteString(`</table>`)
	}
	if c.Button != nil {
		b.WriteString(`<table role="presentation" cellpadding="0" cellspacing="0" style="margin:22px 0 8px"><tr><td style="border-radius:10px;background:#4f46e5">` +
			`<a href="` + e(c.Button.URL) + `" style="display:inline-block;padding:12px 22px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:10px">` +
			e(c.Button.Label) + `</a></td></tr></table>`)
	}
	if c.Footnote != "" {
		b.WriteString(`<p style="margin:16px 0 0;font-size:13px;line-height:1.5;color:#71717a">` + e(c.Footnote) + `</p>`)
	}
	b.WriteString(`</td></tr><tr><td style="padding:20px 8px;font-size:12px;line-height:1.6;color:#71717a">`)
	if c.Reason != "" {
		b.WriteString(e(c.Reason) + `<br>`)
	}
	var links []string
	if c.ManageURL != "" {
		links = append(links, `<a href="`+e(c.ManageURL)+`" style="color:#4f46e5">Email settings</a>`)
	}
	if c.Unsubscribe != "" {
		links = append(links, `<a href="`+e(c.Unsubscribe)+`" style="color:#4f46e5">Unsubscribe</a>`)
	}
	links = append(links, `<a href="https://lynqr.app" style="color:#4f46e5">lynqr.app</a>`)
	b.WriteString(strings.Join(links, " · "))
	b.WriteString(`</td></tr></table></td></tr></table></body></html>`)
	return b.String(), t.String()
}
