package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/qr"
)

// PublicQRTypes are the static types the free generator supports (no account,
// nothing stored, no tracking).
var PublicQRTypes = []models.QRType{models.QRTypeWebsite, models.QRTypeText, models.QRTypeWiFi, models.QRTypeVCard,
	models.QRTypeEmail, models.QRTypeSMS, models.QRTypePhone}

// PublicQRInput is the free generator's request. Only basic styling is offered;
// logos, frames and dynamic codes are part of Lynqr Pro.
type PublicQRInput struct {
	Type      models.QRType `json:"type"`
	TargetURL string        `json:"target_url"`
	RawData   string        `json:"raw_data"`
	FgColor   string        `json:"fg_color"`
	BgColor   string        `json:"bg_color"`
	DotStyle  string        `json:"dot_style"`
	EyeShape  string        `json:"eye_shape"`
	Size      int           `json:"size"`
}

// PublicQR renders a static QR code for anyone. It never stores anything.
func (s *Service) PublicQR(in PublicQRInput) (content string, style models.QRStyle, err error) {
	if in.Type == "" {
		in.Type = models.QRTypeWebsite
	}
	ok := false
	for _, t := range PublicQRTypes {
		ok = ok || t == in.Type
	}
	if !ok {
		return "", style, fmt.Errorf("the free generator supports website, text, Wi-Fi, contact, email, SMS and phone codes")
	}
	if len(in.RawData) > 4000 || len(in.TargetURL) > 2000 {
		return "", style, errors.New("that's too much content for a QR code")
	}
	if in.Type == models.QRTypeWebsite {
		in.TargetURL = strings.TrimSpace(in.TargetURL)
		if in.TargetURL != "" && !strings.Contains(in.TargetURL, "://") {
			in.TargetURL = "https://" + in.TargetURL
		}
		if err := s.checkTarget(in.TargetURL); err != nil {
			return "", style, err
		}
	}
	if content, err = qr.BuildPayload(in.Type, in.RawData, in.TargetURL); err != nil {
		return "", style, err
	}
	style = qr.NormalizeStyle(models.QRStyle{FgColor: in.FgColor, BgColor: in.BgColor, DotStyle: in.DotStyle,
		EyeShape: in.EyeShape, ErrorCorrection: "M", Margin: 2})
	return content, style, nil
}
