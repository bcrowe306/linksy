// Package email sends transactional email (Resend in production).
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// Message is one outgoing email.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
	// Tag groups messages in the provider dashboard (e.g. "admin_code").
	Tag string
}

// Sender delivers messages.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// Resend sends through the Resend HTTP API.
type Resend struct {
	Key  string
	From string
	Base string
	HTTP *http.Client
}

func NewResend(key, from string) *Resend {
	return &Resend{Key: key, From: from, Base: "https://api.resend.com", HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (r *Resend) Send(ctx context.Context, m Message) error {
	body := map[string]any{"from": r.From, "to": []string{m.To}, "subject": m.Subject, "text": m.Text}
	if m.HTML != "" {
		body["html"] = m.HTML
	}
	if m.Tag != "" {
		body["tags"] = []map[string]string{{"name": "category", "value": m.Tag}}
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.Base+"/emails", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach the email provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("email provider returned %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}

// Log prints messages instead of sending them (local development only).
type Log struct{}

func (Log) Send(_ context.Context, m Message) error {
	log.Printf("[email] to=%s subject=%q\n%s", m.To, m.Subject, m.Text)
	return nil
}

// Memory records messages (tests).
type Memory struct {
	mu   sync.Mutex
	Sent []Message
}

func (s *Memory) Send(_ context.Context, m Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sent = append(s.Sent, m)
	return nil
}

// Last returns the most recent message to an address.
func (s *Memory) Last(to string) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.Sent) - 1; i >= 0; i-- {
		if s.Sent[i].To == to {
			return s.Sent[i], true
		}
	}
	return Message{}, false
}
