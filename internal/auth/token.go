package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const (
	SessionPrefix    = "lqs_"     // Browser / CLI session token
	InvitationPrefix = "lqi_"     // Team invitation token
	APIKeyPrefix     = "lk_live_" // Developer API key
)

// NewToken returns a prefixed, URL-safe token with 256 bits of entropy.
func NewToken(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the SHA-256 hex digest stored in place of raw tokens.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
