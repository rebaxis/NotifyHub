package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

// GenerateAgentID генерирует уникальный ID для агента
func GenerateAgentID() string {
	return fmt.Sprintf("agent-%s", uuid.New().String())
}

// GenerateSecret генерирует случайный секрет для аутентификации агента
func GenerateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashSecret хеширует секрет через SHA256
func HashSecret(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}

// VerifySecret проверяет, совпадает ли секрет с сохранённым хешем
func VerifySecret(secret string, hash []byte) bool {
	secretHash := HashSecret(secret)
	if len(secretHash) != len(hash) {
		return false
	}
	for i := range secretHash {
		if secretHash[i] != hash[i] {
			return false
		}
	}
	return true
}
