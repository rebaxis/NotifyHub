package lib

import (
	"strings"
	"testing"
)

func TestGenerateAgentID(t *testing.T) {
	crypto := NewCrypto(CryptoConfig{BcryptCost: 10, SecretByteLength: 32})

	// Test that it generates an ID with correct prefix
	id := crypto.GenerateAgentID()

	if !strings.HasPrefix(id, "agent-") {
		t.Errorf("GenerateAgentID() должен начинаться с 'agent-', получили: %s", id)
	}

	// Test that it generates unique IDs
	id2 := crypto.GenerateAgentID()
	if id == id2 {
		t.Error("GenerateAgentID() должен генерировать уникальные ID")
	}

	// Test minimum length (agent- + UUID format)
	if len(id) < 42 { // agent- (6) + UUID with dashes (36)
		t.Errorf("GenerateAgentID() слишком короткий: %d символов", len(id))
	}
}

func TestGenerateSecret(t *testing.T) {
	crypto := NewCrypto(CryptoConfig{BcryptCost: 10, SecretByteLength: 32})

	// Test successful generation
	secret, err := crypto.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() вернул ошибку: %v", err)
	}

	if secret == "" {
		t.Error("GenerateSecret() вернул пустую строку")
	}

	// Test that it generates unique secrets
	secret2, err := crypto.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() второй вызов вернул ошибку: %v", err)
	}

	if secret == secret2 {
		t.Error("GenerateSecret() должен генерировать уникальные секреты")
	}

	// Test minimum length (32 bytes base64 encoded)
	if len(secret) < 40 {
		t.Errorf("GenerateSecret() слишком короткий: %d символов", len(secret))
	}
}

func TestHashSecret(t *testing.T) {
	crypto := NewCrypto(CryptoConfig{BcryptCost: 10, SecretByteLength: 32})

	secret := "test-secret-123"
	hash := crypto.HashSecret(secret)

	// bcrypt возвращает хеш длиной 60 символов
	if len(hash) != 60 {
		t.Errorf("HashSecret() вернул хеш длиной %d, ожидалось 60", len(hash))
	}

	// bcrypt генерирует разные хеши для одного секрета (из-за соли)
	// Проверяем что хеш не пустой
	if len(hash) == 0 {
		t.Error("HashSecret() вернул пустой хеш")
	}
}

func TestVerifySecret(t *testing.T) {
	crypto := NewCrypto(CryptoConfig{BcryptCost: 10, SecretByteLength: 32})

	secret := "test-secret"
	hash := crypto.HashSecret(secret)

	// Проверка правильного секрета
	if !crypto.VerifySecret(secret, hash) {
		t.Error("VerifySecret() должен вернуть true для правильного секрета")
	}

	// Проверка неправильного секрета
	if crypto.VerifySecret("wrong-secret", hash) {
		t.Error("VerifySecret() должен вернуть false для неправильного секрета")
	}

	// Проверка с невалидным хешем
	if crypto.VerifySecret(secret, []byte{1, 2, 3}) {
		t.Error("VerifySecret() должен вернуть false для невалидного хеша")
	}
}

func TestHashAndVerifyIntegration(t *testing.T) {
	crypto := NewCrypto(CryptoConfig{BcryptCost: 10, SecretByteLength: 32})

	// Генерируем секрет
	secret, err := crypto.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() ошибка: %v", err)
	}

	// Хешируем
	hash := crypto.HashSecret(secret)

	// Проверяем с правильным секретом
	if !crypto.VerifySecret(secret, hash) {
		t.Error("VerifySecret() вернул false для правильного секрета")
	}

	// Проверяем с неправильным секретом
	wrongSecret, _ := crypto.GenerateSecret()
	if crypto.VerifySecret(wrongSecret, hash) {
		t.Error("VerifySecret() вернул true для неправильного секрета")
	}
}
