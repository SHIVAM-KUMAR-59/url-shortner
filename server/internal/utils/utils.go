package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/mail"
	"strings"
)

func NormalizeEmail(rawEmail string) (string, error) {
	// Parse and validate the address format
	addr, err := mail.ParseAddress(strings.TrimSpace(rawEmail))
	if err != nil {
		return "", err
	}

	// Split user and domain components
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return addr.Address, nil
	}

	// Local part can be case-sensitive, but domain is always case-insensitive
	localPart := parts[0]
	domainPart := strings.ToLower(parts[1])

	return localPart + "@" + domainPart, nil
}

const apiKeyLength = 32

func GenerateAPIKey() (string, error) {
	buf := make([]byte, apiKeyLength)

	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil

}

func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func DerefOrZero(userID *int64) int64 {
	if userID != nil {
		return *userID
	}

	return 0
}
