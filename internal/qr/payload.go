package qr

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/bcrowe306/linksy/internal/models"
)

type VCardData struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Organization string `json:"organization"`
	Title        string `json:"title"`
	Phone        string `json:"phone"`
	Mobile       string `json:"mobile"`
	Email        string `json:"email"`
	Website      string `json:"website"`
	Address      string `json:"address"`
	Note         string `json:"note"`
}

type WiFiData struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Encryption string `json:"encryption"` // "WPA", "WEP", "nopass"
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

type AppData struct {
	AppStoreURL  string `json:"app_store_url"`
	PlayStoreURL string `json:"play_store_url"`
	WebURL       string `json:"web_url"`
}

// BuildPayload formats raw typed data into standardized QR payloads
func BuildPayload(qrType models.QRType, rawData string, fallbackTarget string) (string, error) {
	switch qrType {
	case models.QRTypeWebsite:
		return strings.TrimSpace(fallbackTarget), nil

	case models.QRTypeDocument:
		return strings.TrimSpace(fallbackTarget), nil

	case models.QRTypeVCard:
		var vcard VCardData
		if err := json.Unmarshal([]byte(rawData), &vcard); err != nil {
			return fallbackTarget, err
		}
		var b strings.Builder
		b.WriteString("BEGIN:VCARD\r\nVERSION:3.0\r\n")
		b.WriteString(fmt.Sprintf("N:%s;%s;;;\r\n", vcard.LastName, vcard.FirstName))
		b.WriteString(fmt.Sprintf("FN:%s %s\r\n", vcard.FirstName, vcard.LastName))
		if vcard.Organization != "" {
			b.WriteString(fmt.Sprintf("ORG:%s\r\n", vcard.Organization))
		}
		if vcard.Title != "" {
			b.WriteString(fmt.Sprintf("TITLE:%s\r\n", vcard.Title))
		}
		if vcard.Phone != "" {
			b.WriteString(fmt.Sprintf("TEL;TYPE=WORK,VOICE:%s\r\n", vcard.Phone))
		}
		if vcard.Mobile != "" {
			b.WriteString(fmt.Sprintf("TEL;TYPE=CELL,VOICE:%s\r\n", vcard.Mobile))
		}
		if vcard.Email != "" {
			b.WriteString(fmt.Sprintf("EMAIL;TYPE=PREF,INTERNET:%s\r\n", vcard.Email))
		}
		if vcard.Website != "" {
			b.WriteString(fmt.Sprintf("URL:%s\r\n", vcard.Website))
		}
		if vcard.Address != "" {
			b.WriteString(fmt.Sprintf("ADR;TYPE=WORK:;;%s;;;;\r\n", vcard.Address))
		}
		if vcard.Note != "" {
			b.WriteString(fmt.Sprintf("NOTE:%s\r\n", vcard.Note))
		}
		b.WriteString("END:VCARD\r\n")
		return b.String(), nil

	case models.QRTypeWiFi:
		var wifi WiFiData
		if err := json.Unmarshal([]byte(rawData), &wifi); err != nil {
			return fallbackTarget, err
		}
		enc := wifi.Encryption
		if enc == "" {
			enc = "WPA"
		}
		h := "false"
		if wifi.Hidden {
			h = "true"
		}
		return fmt.Sprintf("WIFI:S:%s;T:%s;P:%s;H:%s;;", wifi.SSID, enc, wifi.Password, h), nil

	case models.QRTypeEmail:
		var email EmailData
		if err := json.Unmarshal([]byte(rawData), &email); err != nil {
			return fallbackTarget, err
		}
		q := url.Values{}
		if email.Subject != "" {
			q.Set("subject", email.Subject)
		}
		if email.Body != "" {
			q.Set("body", email.Body)
		}
		queryStr := q.Encode()
		if queryStr != "" {
			return fmt.Sprintf("mailto:%s?%s", email.Email, queryStr), nil
		}
		return fmt.Sprintf("mailto:%s", email.Email), nil

	case models.QRTypeSMS:
		var sms SMSData
		if err := json.Unmarshal([]byte(rawData), &sms); err != nil {
			return fallbackTarget, err
		}
		if sms.Message != "" {
			return fmt.Sprintf("smsto:%s:%s", sms.Phone, sms.Message), nil
		}
		return fmt.Sprintf("smsto:%s", sms.Phone), nil

	case models.QRTypeApp:
		return strings.TrimSpace(fallbackTarget), nil

	case models.QRTypeText:
		return rawData, nil

	default:
		return fallbackTarget, nil
	}
}
