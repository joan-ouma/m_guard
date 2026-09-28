package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateRandomToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashTokenSHA256(rawToken string) string {
	hasher := sha256.New()
	hasher.Write([]byte(rawToken))
	return hex.EncodeToString(hasher.Sum(nil))
}
