package handler

import (
	"encoding/json"
	"html/template"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/web"
)

type generatorTab struct {
	Path, Label string
	Active      bool
}

type faq struct{ Q, A string }

type generatorCopy struct {
	Slug, Type, Label, Title, Description, Heading, Intro, HowTitle string
	Steps                                                           []string
	FAQ                                                             []faq
}

type generatorView struct {
	generatorCopy
	Tabs []generatorTab
}

var commonFAQ = []faq{
	{"Is this QR code generator really free?", "Yes. Codes made here are free for personal and commercial use, with no sign-up, watermark or expiry. They're static, so the content is stored inside the code itself and works forever."},
	{"What's the difference between static and dynamic QR codes?", "A static code contains the content directly and can't be changed after printing. A dynamic Lynqr code points to a short link you can re-point at any time, and every scan is counted with location and device analytics."},
}

var generators = []generatorCopy{
	{Slug: "", Type: "website", Label: "Website",
		Title: "Free QR Code Generator: create a QR code for any link", Description: "Create a free QR code for any website or link in seconds. Custom colors and shapes, high-resolution PNG and SVG downloads, no sign-up and no watermark.",
		Heading: "Free QR code generator", Intro: "Turn any link into a QR code in seconds. Pick your colors and shapes, then download a print-ready PNG or SVG.",
		HowTitle: "How to make a QR code for a link", Steps: []string{"Paste the website address you want people to open.", "Choose colors and a pattern. Keep the code dark on a light background so it scans easily.", "Download a PNG for web and documents, or an SVG for print, and test it with your phone camera before printing."},
		FAQ: append([]faq{{"How big should I print my QR code?", "A good rule is the scanning distance divided by 10: a code scanned from 30 cm away should be at least 3 cm wide. Leave a blank margin around it."}}, commonFAQ...)},
	{Slug: "wifi", Type: "wifi", Label: "Wi-Fi",
		Title: "Free Wi-Fi QR Code Generator: share your network password", Description: "Make a free Wi-Fi QR code so guests can join your network by scanning, no typing needed. Works with iPhone and Android. Your password is never stored.",
		Heading: "Wi-Fi QR code generator", Intro: "Let guests join your Wi-Fi by pointing their camera at a code. No more spelling out passwords.",
		HowTitle: "How to share Wi-Fi with a QR code", Steps: []string{"Enter your network name (SSID) exactly as it appears, including capital letters.", "Enter the password and choose the security type (WPA/WPA2 for most home and office routers).", "Download the code and print it for your café, office or guest room."},
		FAQ: append([]faq{{"Do Wi-Fi QR codes work on iPhone?", "Yes. The built-in camera on iPhone (iOS 11 and later) and on most Android phones recognizes Wi-Fi codes and offers to join the network."}, {"Is my password safe?", "The code is generated in memory and never stored by Lynqr. Anyone who scans the printed code can join, so only display it where you'd share the password."}}, commonFAQ...)},
	{Slug: "vcard", Type: "vcard", Label: "Contact card",
		Title: "Free vCard QR Code Generator: share contact details", Description: "Create a free vCard QR code for business cards. One scan saves your name, phone, email and company to a phone's contacts.",
		Heading: "vCard QR code generator", Intro: "Put your contact details on a business card, badge or email signature. One scan saves you to their contacts.",
		HowTitle: "How to make a contact QR code", Steps: []string{"Fill in the details you want to share. Everything is optional except a name.", "Style the code to match your card.", "Download as SVG for print. Contact codes hold a lot of data, so print them at least 2.5 cm wide."},
		FAQ: append([]faq{{"Can I update my details later?", "Not with a static code: the details are inside the code. With a dynamic Lynqr contact card you can edit your details any time without reprinting."}}, commonFAQ...)},
	{Slug: "email", Type: "email", Label: "Email",
		Title: "Free Email QR Code Generator: pre-filled email messages", Description: "Create a free QR code that opens a new email with the address, subject and message already filled in.",
		Heading: "Email QR code generator", Intro: "Scanning opens a ready-to-send email: great for support, feedback and sign-ups.",
		HowTitle: "How to make an email QR code", Steps: []string{"Enter the address the email should go to.", "Optionally add a subject and a message.", "Download and test the code with your phone."}, FAQ: commonFAQ},
	{Slug: "sms", Type: "sms", Label: "SMS",
		Title: "Free SMS QR Code Generator: pre-written text messages", Description: "Create a free QR code that opens a text message with the number and message already filled in. Perfect for opt-ins and contests.",
		Heading: "SMS QR code generator", Intro: "Scanning opens a text message with your number and message ready to send.",
		HowTitle: "How to make an SMS QR code", Steps: []string{"Enter the phone number, including the country code.", "Add the message people should send, such as a keyword.", "Download the code and test it on iPhone and Android."}, FAQ: commonFAQ},
	{Slug: "phone", Type: "phone", Label: "Phone",
		Title: "Free Phone Call QR Code Generator", Description: "Create a free QR code that starts a phone call when scanned. Add it to flyers, signs and packaging.",
		Heading: "Phone call QR code generator", Intro: "One scan starts a call to your number. Ideal for signs, vehicles and flyers.",
		HowTitle: "How to make a call QR code", Steps: []string{"Enter your phone number with the country code, such as +1 555 0100.", "Pick your colors.", "Download and print."}, FAQ: commonFAQ},
	{Slug: "text", Type: "text", Label: "Text",
		Title: "Free Text QR Code Generator", Description: "Turn any text into a free QR code that shows the message when scanned. Works offline, no link needed.",
		Heading: "Text QR code generator", Intro: "Encode a short message, serial number or instructions. It shows on screen when scanned, even offline.",
		HowTitle: "How to make a text QR code", Steps: []string{"Type your text. Shorter text makes a simpler, easier-to-scan code.", "Choose a style.", "Download your code."}, FAQ: commonFAQ},
}

func generatorPath(slug string) string {
	if slug == "" {
		return "/qr-code-generator"
	}
	return "/qr-code-generator/" + slug
}

// GeneratorSlugs lists the generator pages (for routing and the sitemap).
func GeneratorSlugs() []string {
	out := make([]string, len(generators))
	for i, g := range generators {
		out[i] = g.Slug
	}
	return out
}

// GeneratorPage serves the free QR generator for one content type.
func (h *Handler) GeneratorPage(slug string) fiber.Handler {
	return func(c fiber.Ctx) error {
		var cur generatorCopy
		tabs := make([]generatorTab, len(generators))
		for i, g := range generators {
			tabs[i] = generatorTab{Path: generatorPath(g.Slug), Label: g.Label, Active: g.Slug == slug}
			if g.Slug == slug {
				cur = g
			}
		}
		faqLD := make([]map[string]any, len(cur.FAQ))
		for i, f := range cur.FAQ {
			faqLD[i] = map[string]any{"@type": "Question", "name": f.Q, "acceptedAnswer": map[string]any{"@type": "Answer", "text": f.A}}
		}
		ld, _ := json.Marshal(map[string]any{"@context": "https://schema.org", "@graph": []any{
			map[string]any{"@type": "WebApplication", "name": cur.Heading, "url": "https://lynqr.app" + generatorPath(slug), "applicationCategory": "UtilitiesApplication",
				"operatingSystem": "Web", "offers": map[string]any{"@type": "Offer", "price": "0", "priceCurrency": "USD"}},
			map[string]any{"@type": "FAQPage", "mainEntity": faqLD},
		}})
		return h.renderPage(c, "generator", web.View{Title: cur.Title, Description: cur.Description, Path: generatorPath(slug),
			JSONLD: template.JS(ld), Data: generatorView{generatorCopy: cur, Tabs: tabs}})
	}
}
