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
	URLLabel, URLPlaceholder                                        string
	Steps                                                           []string
	FAQ                                                             []faq
}

type generatorView struct {
	generatorCopy
	Tabs []generatorTab
}

var commonFAQ = []faq{
	{"Is this QR code generator really free, and will my code ever expire?", "Yes—100% free for personal and commercial use, with no sign-up, no scan limits, no watermark, and zero expiration. Unlike deceptive 'free trial' generators that deactivate your code after 14 days, our static QR codes encode your content directly into the pattern so they work forever."},
	{"What's the difference between static and dynamic QR codes?", "A static code contains the content directly and can't be changed after printing. A dynamic Lynqr code points to a short link you can re-point at any time without reprinting, supports hosted PDF uploads, and counts every scan with real-time location and device analytics."},
}

var generators = []generatorCopy{
	{Slug: "", Type: "website", Label: "Website",
		Title: "Free QR Code Generator: No Sign-Up, No Watermark, Never Expires", Description: "Create a free QR code for any website or link in seconds. Custom colors and shapes, print-ready PNG and SVG downloads, no sign-up, no watermark, and never expires.",
		Heading: "Free QR code generator", Intro: "Turn any link into a permanent QR code in seconds. Pick your colors and shapes, then download a print-ready PNG or SVG—no account required and no 14-day trial trap.",
		HowTitle: "How to make a free QR code for a link", Steps: []string{"Paste the website address you want people to open.", "Choose colors and a pattern. Keep the code dark on a light background so it scans easily.", "Download a PNG for web and documents, or an SVG for print, and test it with your phone camera before printing."},
		FAQ: append([]faq{{"How big should I print my QR code?", "A good rule is the scanning distance divided by 10: a code scanned from 30 cm away should be at least 3 cm wide. Leave a blank quiet zone around it."}}, commonFAQ...)},
	{Slug: "wifi", Type: "wifi", Label: "Wi-Fi",
		Title: "Free Wi-Fi QR Code Generator: Share Wi-Fi Without Typing Passwords", Description: "Make a free Wi-Fi QR code so guests can join your network by scanning, no typing needed. Works on iPhone and Android, never expires, and your password is never stored.",
		Heading: "Wi-Fi QR code generator", Intro: "Let guests join your Wi-Fi by pointing their camera at a code. No more spelling out passwords at your café, Airbnb or office.",
		HowTitle: "How to share Wi-Fi with a QR code", Steps: []string{"Enter your network name (SSID) exactly as it appears, including capital letters.", "Enter the password and choose the security type (WPA/WPA2 for most home and office routers).", "Download the code and print it for your café, office, Airbnb or guest room."},
		FAQ: append([]faq{{"Do Wi-Fi QR codes work on iPhone and Android?", "Yes. The built-in camera on iPhone (iOS 11 and later) and on modern Android phones recognizes Wi-Fi QR codes natively and offers to join the network in one tap."}, {"Is my Wi-Fi password safe?", "The code is generated in memory and never stored by Lynqr. Anyone who scans the printed code can join, so only display it where you'd share the password."}}, commonFAQ...)},
	{Slug: "vcard", Type: "vcard", Label: "Contact card",
		Title: "Free vCard QR Code Generator for Business Cards (Never Expires)", Description: "Create a free vCard QR code for business cards and badges. One scan saves your name, phone, email and company to a phone's contacts. Free SVG & PNG downloads.",
		Heading: "vCard QR code generator", Intro: "Put your contact details on a business card, badge or email signature. One scan saves you straight to their phone contacts.",
		HowTitle: "How to make a vCard contact QR code", Steps: []string{"Fill in the details you want to share. Everything is optional except a name.", "Style the code to match your business card.", "Download as SVG for print. Contact codes hold a lot of data, so print them at least 2.5 cm wide."},
		FAQ: append([]faq{{"Can I update my contact details later?", "Not with a static code: the details are encoded directly inside the pattern. With a dynamic Lynqr contact card you can edit your phone number, title or company any time without reprinting your business cards."}}, commonFAQ...)},
	{Slug: "google-review", Type: "website", Label: "Google Review",
		URLLabel: "Google Business review link", URLPlaceholder: "https://g.page/r/your-business/review",
		Title: "Free Google Review QR Code Generator (No Sign-Up, Never Expires)", Description: "Create a free Google Review QR code for your restaurant, clinic or local business. Customers scan once to open your Google review form. Free PNG & SVG, never expires.",
		Heading: "Google Review QR code generator", Intro: "Get more 5-star Google reviews from happy customers. Put a code on table tents, receipts, counter signs or thank-you cards.",
		HowTitle: "How to make a Google Review QR code", Steps: []string{"Copy your 'Ask for reviews' link from your Google Business Profile dashboard (it usually looks like https://g.page/r/.../review).", "Paste the link above and customize your QR code's colors and corner style.", "Download the print-ready SVG or high-res PNG and place it on receipts, menus or checkout counters."},
		FAQ: append([]faq{{"Where do I find my Google Review link?", "Sign in to your Google Business Profile, click 'Ask for reviews' (or 'Get more reviews'), and copy the short link Google provides. Paste that link into the generator above."}, {"Will my Google Review QR code ever stop working?", "Never. Static Google Review QR codes generated here point directly to your Google review URL with no middleman and no expiration date."}}, commonFAQ...)},
	{Slug: "pdf", Type: "website", Label: "PDF & Menu",
		URLLabel: "PDF or online menu URL", URLPlaceholder: "https://lynqr.app/sample-menu.pdf",
		Title: "Free PDF & Restaurant Menu QR Code Generator (No Expiration)", Description: "Create a free QR code for any PDF link, restaurant menu, wine list or brochure. Free forever with no sign-up, or use Lynqr Pro to host and replace PDFs without reprinting.",
		Heading: "PDF & restaurant menu QR code generator", Intro: "Turn any hosted PDF, digital restaurant menu, product manual or event program into a scannable QR code.",
		HowTitle: "How to make a QR code for a PDF or restaurant menu", Steps: []string{"Upload your PDF to your website or cloud drive and copy the public link (or use Lynqr Pro to host the PDF directly on our CDN).", "Paste the PDF or menu URL into the field above and customize the design.", "Download as SVG or PNG and test-scan on both iPhone and Android before printing table tents or flyers."},
		FAQ: append([]faq{{"Can I replace the PDF later without reprinting the QR code?", "If you use a static code pointing to a fixed URL, changing the filename breaks the link. With Lynqr Pro's Hosted Documents feature, you upload your PDF directly to Lynqr and can swap in a new seasonal menu or updated brochure anytime while keeping the exact same printed QR code."}}, commonFAQ...)},
	{Slug: "whatsapp", Type: "website", Label: "WhatsApp",
		URLLabel: "WhatsApp click-to-chat URL (wa.me)", URLPlaceholder: "https://wa.me/15550100?text=Hello",
		Title: "Free WhatsApp QR Code Generator: Click-to-Chat QR Codes", Description: "Create a free WhatsApp QR code using your wa.me link so customers can message your business in one scan. No sign-up, no watermark, never expires.",
		Heading: "WhatsApp QR code generator", Intro: "Let customers start a WhatsApp conversation with your business in a single scan—no saving phone numbers required.",
		HowTitle: "How to make a WhatsApp QR code", Steps: []string{"Format your WhatsApp link as https://wa.me/<number> using your full phone number with country code (digits only, e.g. https://wa.me/15550100).", "Optionally add ?text=Your%20message to pre-fill the greeting, then style your code.", "Download your PNG or SVG code and add it to packaging, storefronts or support cards."},
		FAQ: append([]faq{{"How do I format a WhatsApp wa.me link?", "Use https://wa.me/ followed by your full phone number in international format without plus signs, spaces or dashes (for example, https://wa.me/15551234567)."}}, commonFAQ...)},
	{Slug: "instagram", Type: "website", Label: "Instagram",
		URLLabel: "Instagram profile or post URL", URLPlaceholder: "https://instagram.com/yourhandle",
		Title: "Free Instagram QR Code Generator (No Sign-Up, Never Expires)", Description: "Create a free QR code for your Instagram profile, reel or post in seconds. Custom brand colors, high-res PNG and vector SVG downloads, no sign-up and no expiration.",
		Heading: "Instagram QR code generator", Intro: "Turn foot traffic, product packaging, event booths and business cards into Instagram followers with one scan.",
		HowTitle: "How to make an Instagram QR code", Steps: []string{"Paste the full URL to your Instagram profile (e.g. https://instagram.com/yourhandle) or a specific Reel/post.", "Customize the colors and dot pattern to match your aesthetic while keeping strong contrast.", "Download the SVG or PNG and print it on your packaging, signage or merch tags."},
		FAQ: append([]faq{{"Does scanning open the Instagram app directly?", "Yes. When someone scans an instagram.com link with their phone camera, iOS and Android open the profile directly inside the Instagram app if it is installed, or fall back to the mobile web browser."}}, commonFAQ...)},
	{Slug: "email", Type: "email", Label: "Email",
		Title: "Free Email QR Code Generator: Pre-Filled Email Messages", Description: "Create a free QR code that opens a new email with the recipient address, subject and message already filled in. No sign-up, never expires.",
		Heading: "Email QR code generator", Intro: "Scanning opens a ready-to-send email: great for customer support, feedback cards and event RSVPs.",
		HowTitle: "How to make an email QR code", Steps: []string{"Enter the address the email should go to.", "Optionally add a subject line and pre-written message body.", "Download and test the code with your phone."}, FAQ: commonFAQ},
	{Slug: "sms", Type: "sms", Label: "SMS",
		Title: "Free SMS QR Code Generator: Pre-Written Text Messages", Description: "Create a free QR code that opens an SMS text message with the phone number and keyword already filled in. Perfect for opt-ins, support and contests.",
		Heading: "SMS QR code generator", Intro: "Scanning opens a text message with your phone number and message ready to send.",
		HowTitle: "How to make an SMS QR code", Steps: []string{"Enter the destination phone number, including the country code.", "Add the message people should send, such as an opt-in keyword.", "Download the code and test it on iPhone and Android."}, FAQ: commonFAQ},
	{Slug: "phone", Type: "phone", Label: "Phone",
		Title: "Free Phone Call QR Code Generator (Tap to Call)", Description: "Create a free QR code that dials your phone number when scanned. Ideal for service vehicles, signs, flyers and packaging. Free PNG & SVG.",
		Heading: "Phone call QR code generator", Intro: "One scan starts a phone call to your business number. Ideal for signs, service vehicles and flyers.",
		HowTitle: "How to make a phone call QR code", Steps: []string{"Enter your phone number with the country code, such as +1 555 0100.", "Pick your colors and corner style.", "Download and print."}, FAQ: commonFAQ},
	{Slug: "text", Type: "text", Label: "Text",
		Title: "Free Plain Text QR Code Generator (Works Offline)", Description: "Turn any text, serial number or note into a free QR code that displays the message when scanned—works completely offline with no internet connection.",
		Heading: "Text QR code generator", Intro: "Encode a short message, serial number, coupon code or instructions. It shows on screen when scanned, even offline.",
		HowTitle: "How to make a text QR code", Steps: []string{"Type your text. Shorter text makes a simpler, easier-to-scan code.", "Choose a style and high-contrast colors.", "Download your PNG or SVG code."}, FAQ: commonFAQ},
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
		if cur.URLLabel == "" {
			cur.URLLabel = "Website URL"
		}
		if cur.URLPlaceholder == "" {
			cur.URLPlaceholder = "https://example.com/menu"
		}
		faqLD := make([]map[string]any, len(cur.FAQ))
		for i, f := range cur.FAQ {
			faqLD[i] = map[string]any{"@type": "Question", "name": f.Q, "acceptedAnswer": map[string]any{"@type": "Answer", "text": f.A}}
		}
		howSteps := make([]map[string]any, len(cur.Steps))
		for i, s := range cur.Steps {
			howSteps[i] = map[string]any{"@type": "HowToStep", "position": i + 1, "text": s}
		}
		crumbs := []map[string]any{
			{"@type": "ListItem", "position": 1, "name": "Home", "item": "https://lynqr.app/"},
			{"@type": "ListItem", "position": 2, "name": "Free QR Code Generator", "item": "https://lynqr.app/qr-code-generator"},
		}
		if slug != "" {
			crumbs = append(crumbs, map[string]any{"@type": "ListItem", "position": 3, "name": cur.Heading, "item": "https://lynqr.app" + generatorPath(slug)})
		}
		ld, _ := json.Marshal(map[string]any{"@context": "https://schema.org", "@graph": []any{
			map[string]any{"@type": "WebApplication", "name": cur.Heading, "description": cur.Description, "url": "https://lynqr.app" + generatorPath(slug), "applicationCategory": "UtilitiesApplication",
				"operatingSystem": "Web", "offers": map[string]any{"@type": "Offer", "price": "0", "priceCurrency": "USD"}},
			map[string]any{"@type": "BreadcrumbList", "itemListElement": crumbs},
			map[string]any{"@type": "HowTo", "name": cur.HowTitle, "step": howSteps},
			map[string]any{"@type": "FAQPage", "mainEntity": faqLD},
		}})
		return h.renderPage(c, "generator", web.View{Title: cur.Title, Description: cur.Description, Path: generatorPath(slug),
			JSONLD: template.JS(ld), Data: generatorView{generatorCopy: cur, Tabs: tabs}})
	}
}
