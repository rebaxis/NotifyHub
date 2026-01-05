package lib

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CryptoConfig содержит настройки криптографии
type CryptoConfig struct {
	BcryptCost       int
	SecretByteLength int
}

// Crypto предоставляет криптографические функции с настраиваемыми параметрами
type Crypto struct {
	config CryptoConfig
}

// NewCrypto создает новый экземпляр Crypto с заданной конфигурацией
func NewCrypto(cfg CryptoConfig) *Crypto {
	return &Crypto{
		config: cfg,
	}
}

// GenerateAgentID генерирует уникальный ID для агента
func (c *Crypto) GenerateAgentID() string {
	return fmt.Sprintf("agent-%s", uuid.New().String())
}

// GenerateSecret генерирует случайный секрет для аутентификации агента
func (c *Crypto) GenerateSecret() (string, error) {
	bytes := make([]byte, c.config.SecretByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashSecret хеширует секрет через bcrypt
func (c *Crypto) HashSecret(secret string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), c.config.BcryptCost)
	if err != nil {
		return []byte{}
	}
	return hash
}

// VerifySecret проверяет, совпадает ли секрет с сохранённым bcrypt хешем
func (c *Crypto) VerifySecret(secret string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(secret))
	return err == nil
}
