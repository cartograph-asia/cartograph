package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// tokenLen is the size of magic-link and session tokens in bytes.
const tokenLen = 32

// newToken returns tokenLen random bytes.
func newToken() []byte {
	b := make([]byte, tokenLen)
	rand.Read(b) // never returns an error (crypto/rand, Go 1.24+)
	return b
}

func encodeToken(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// decodeToken parses a token from a link or cookie; anything that is not
// exactly tokenLen bytes of base64url is rejected.
func decodeToken(s string) ([]byte, bool) {
	if len(s) != base64.RawURLEncoding.EncodedLen(tokenLen) {
		return nil, false
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) != tokenLen {
		return nil, false
	}
	return b, true
}

// hashToken is what the database stores instead of the token.
func hashToken(raw []byte) []byte {
	h := sha256.Sum256(raw)
	return h[:]
}
