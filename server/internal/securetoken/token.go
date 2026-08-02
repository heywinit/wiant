package securetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const byteLength = 32

// New returns a cryptographically random URL-safe token and its storage digest.
func New() (string, []byte, error) {
	raw := make([]byte, byteLength)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, Digest(token), nil
}

// Digest returns the SHA-256 representation used to look up a token without storing it in plaintext.
func Digest(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}
