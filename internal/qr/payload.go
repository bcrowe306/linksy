package qr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/bcrowe306/linksy/internal/models"
)

type VCardData struct {
	Format       string `json:"format"` // vcard3 (default), vcard4, mecard
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Organization string `json:"organization"`
	Title        string `json:"title"`
	Phone        string `json:"phone"`
	Mobile       string `json:"mobile"`
	Email        string `json:"email"`
	Website      string `json:"website"`
	Street       string `json:"street"`
	City         string `json:"city"`
	Region       string `json:"region"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
	Address      string `json:"address"` // legacy single-line address
	Note         string `json:"note"`
}

type WiFiData struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Encryption string `json:"encryption"` // WPA (WPA/WPA2), WPA3, WEP, nopass
	Hidden     bool   `json:"hidden"`
}

type EmailData struct {
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type SMSData struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

type PhoneData struct {
	Phone string `json:"phone"`
}

// AppData routes scanners to the right store; it is always served dynamically.
type AppData struct {
	AppStoreURL  string `json:"app_store_url"`
	PlayStoreURL string `json:"play_store_url"`
	WebURL       string `json:"web_url"`
}

// MaxTextBytes keeps plain-text payloads inside reliable QR capacity.
const MaxTextBytes = 1200

var (
	ErrEmptyPayload = errors.New("this QR code has no content yet")
	phoneChars      = regexp.MustCompile(`[^0-9+*#]`)
	emailRe         = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func cleanPhone(p string) string { return phoneChars.ReplaceAllString(p, "") }

// escapeWiFi escapes the characters reserved by the WIFI: and MECARD: schemes.
func escapeWiFi(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, `:`, `\:`, `"`, `\"`)
	return r.Replace(s)
}

// escapeVCard escapes text values per RFC 6350 §3.4.
func escapeVCard(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `,`, `\,`, `;`, `\;`, "\r\n", `\n`, "\n", `\n`)
	return r.Replace(strings.TrimSpace(s))
}

func decode(raw string, v any) error {
	if strings.TrimSpace(raw) == "" {
		return ErrEmptyPayload
	}
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return fmt.Errorf("invalid content data: %w", err)
	}
	return nil
}

// BuildPayload formats typed content into the string encoded in the QR code.
// Dynamic types (website, document, app, dynamic vCard) encode a redirect URL
// instead; see service.CreateQRCode.
func BuildPayload(qrType models.QRType, rawData string, target string) (string, error) {
	switch qrType {
	case models.QRTypeWebsite, models.QRTypeDocument:
		if strings.TrimSpace(target) == "" {
			return "", ErrEmptyPayload
		}
		return strings.TrimSpace(target), nil

	case models.QRTypeApp:
		var a AppData
		if err := decode(rawData, &a); err != nil {
			return "", err
		}
		for _, u := range []string{a.WebURL, a.AppStoreURL, a.PlayStoreURL} {
			if u != "" {
				return u, nil
			}
		}
		return "", errors.New("add at least one App Store, Google Play or web link")

	case models.QRTypeVCard:
		var v VCardData
		if err := decode(rawData, &v); err != nil {
			return "", err
		}
		return BuildVCard(v)

	case models.QRTypeWiFi:
		var w WiFiData
		if err := decode(rawData, &w); err != nil {
			return "", err
		}
		if strings.TrimSpace(w.SSID) == "" {
			return "", errors.New("Wi-Fi network name (SSID) is required")
		}
		enc := strings.ToUpper(strings.TrimSpace(w.Encryption))
		switch enc {
		case "", "WPA", "WPA2":
			enc = "WPA"
		case "WPA3", "SAE":
			enc = "SAE"
		case "WEP":
		case "NOPASS", "NONE", "OPEN":
			enc = "nopass"
		default:
			enc = "WPA"
		}
		var b strings.Builder
		fmt.Fprintf(&b, "WIFI:T:%s;S:%s;", enc, escapeWiFi(w.SSID))
		if enc != "nopass" {
			fmt.Fprintf(&b, "P:%s;", escapeWiFi(w.Password))
		}
		if w.Hidden {
			b.WriteString("H:true;")
		}
		b.WriteString(";")
		return b.String(), nil

	case models.QRTypeEmail:
		var e EmailData
		if err := decode(rawData, &e); err != nil {
			return "", err
		}
		addr := strings.TrimSpace(e.Email)
		if !emailRe.MatchString(addr) {
			return "", errors.New("a valid recipient email address is required")
		}
		q := []string{}
		if e.Subject != "" {
			q = append(q, "subject="+url.PathEscape(e.Subject))
		}
		if e.Body != "" {
			q = append(q, "body="+url.PathEscape(e.Body))
		}
		if len(q) > 0 {
			return "mailto:" + addr + "?" + strings.Join(q, "&"), nil
		}
		return "mailto:" + addr, nil

	case models.QRTypeSMS:
		var s SMSData
		if err := decode(rawData, &s); err != nil {
			return "", err
		}
		phone := cleanPhone(s.Phone)
		if phone == "" {
			return "", errors.New("a phone number is required")
		}
		if s.Message != "" {
			return "SMSTO:" + phone + ":" + s.Message, nil
		}
		return "SMSTO:" + phone, nil

	case models.QRTypePhone:
		var p PhoneData
		if err := decode(rawData, &p); err != nil {
			return "", err
		}
		phone := cleanPhone(p.Phone)
		if phone == "" {
			return "", errors.New("a phone number is required")
		}
		return "tel:" + phone, nil

	case models.QRTypeText:
		if strings.TrimSpace(rawData) == "" {
			return "", ErrEmptyPayload
		}
		if len(rawData) > MaxTextBytes {
			return "", fmt.Errorf("text is limited to %d characters", MaxTextBytes)
		}
		return rawData, nil
	}
	return "", fmt.Errorf("unsupported QR type %q", qrType)
}

// BuildVCard renders contact data as vCard 3.0, vCard 4.0 or MeCard.
func BuildVCard(v VCardData) (string, error) {
	if strings.TrimSpace(v.FirstName+v.LastName+v.Organization) == "" {
		return "", errors.New("add a name or organization for the contact")
	}
	street := v.Street
	if street == "" {
		street = v.Address
	}

	if strings.EqualFold(v.Format, "mecard") {
		var b strings.Builder
		fmt.Fprintf(&b, "MECARD:N:%s,%s;", escapeWiFi(v.LastName), escapeWiFi(v.FirstName))
		for _, p := range []string{v.Phone, v.Mobile} {
			if p := cleanPhone(p); p != "" {
				fmt.Fprintf(&b, "TEL:%s;", p)
			}
		}
		if v.Email != "" {
			fmt.Fprintf(&b, "EMAIL:%s;", escapeWiFi(v.Email))
		}
		if v.Organization != "" {
			fmt.Fprintf(&b, "ORG:%s;", escapeWiFi(v.Organization))
		}
		if v.Website != "" {
			fmt.Fprintf(&b, "URL:%s;", escapeWiFi(v.Website))
		}
		if addr := strings.Join(nonEmpty(street, v.City, v.Region, v.PostalCode, v.Country), ", "); addr != "" {
			fmt.Fprintf(&b, "ADR:%s;", escapeWiFi(addr))
		}
		if v.Note != "" {
			fmt.Fprintf(&b, "NOTE:%s;", escapeWiFi(v.Note))
		}
		b.WriteString(";")
		return b.String(), nil
	}

	v4 := strings.EqualFold(v.Format, "vcard4")
	var b strings.Builder
	version := "3.0"
	if v4 {
		version = "4.0"
	}
	b.WriteString("BEGIN:VCARD\r\nVERSION:" + version + "\r\n")
	fmt.Fprintf(&b, "N:%s;%s;;;\r\n", escapeVCard(v.LastName), escapeVCard(v.FirstName))
	fn := strings.TrimSpace(v.FirstName + " " + v.LastName)
	if fn == "" {
		fn = v.Organization
	}
	fmt.Fprintf(&b, "FN:%s\r\n", escapeVCard(fn))
	if v.Organization != "" {
		fmt.Fprintf(&b, "ORG:%s\r\n", escapeVCard(v.Organization))
	}
	if v.Title != "" {
		fmt.Fprintf(&b, "TITLE:%s\r\n", escapeVCard(v.Title))
	}
	tel := func(kind, number string) {
		if p := cleanPhone(number); p != "" {
			if v4 {
				fmt.Fprintf(&b, "TEL;TYPE=%s;VALUE=uri:tel:%s\r\n", strings.ToLower(kind), p)
			} else {
				fmt.Fprintf(&b, "TEL;TYPE=%s,VOICE:%s\r\n", kind, p)
			}
		}
	}
	tel("WORK", v.Phone)
	tel("CELL", v.Mobile)
	if v.Email != "" {
		fmt.Fprintf(&b, "EMAIL;TYPE=INTERNET:%s\r\n", escapeVCard(v.Email))
	}
	if v.Website != "" {
		fmt.Fprintf(&b, "URL:%s\r\n", strings.TrimSpace(v.Website))
	}
	if len(nonEmpty(street, v.City, v.Region, v.PostalCode, v.Country)) > 0 {
		fmt.Fprintf(&b, "ADR;TYPE=WORK:;;%s;%s;%s;%s;%s\r\n",
			escapeVCard(street), escapeVCard(v.City), escapeVCard(v.Region), escapeVCard(v.PostalCode), escapeVCard(v.Country))
	}
	if v.Note != "" {
		fmt.Fprintf(&b, "NOTE:%s\r\n", escapeVCard(v.Note))
	}
	b.WriteString("END:VCARD\r\n")
	return b.String(), nil
}

func nonEmpty(vals ...string) []string {
	out := []string{}
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}
