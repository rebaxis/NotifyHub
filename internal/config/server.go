package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/go-playground/validator/v10"
)

// ServerConfig содержит настройки сервера
type ServerConfig struct {
	Port                   int    `validate:"required,min=1,max=65535"`
	DatabaseDSN            string `validate:"required"`
	AdminToken             string `validate:"required"`
	AgentRegistrationToken string `validate:"required"`
	LogLevel               string `validate:"required,oneof=debug info warn error"`
}

// LoadServerConfig загружает конфигурацию сервера из переменных окружения (приоритет) и флагов
func LoadServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{}

	// Определяем флаги
	port := flag.Int("port", 0, "Server port")
	dbDSN := flag.String("db", "", "Database DSN")
	adminToken := flag.String("admin-token", "", "Admin token")
	agentRegistrationToken := flag.String("agent-registration-token", "", "Agent registration token")
	logLevel := flag.String("log-level", "", "Log level: debug, info, warn, error")

	// Парсим флаги
	flag.Parse()

	// Загружаем конфигурацию, переменные окружения имеют приоритет над флагами
	cfg.Port = getIntConfig(*port, "PORT", 8080)
	cfg.DatabaseDSN = getStringConfig(*dbDSN, "DATABASE_DSN", "")
	cfg.AdminToken = getStringConfig(*adminToken, "ADMIN_TOKEN", "")
	cfg.AgentRegistrationToken = getStringConfig(*agentRegistrationToken, "AGENT_REGISTRATION_TOKEN", "")
	cfg.LogLevel = getStringConfig(*logLevel, "LOG_LEVEL", "info")

	// Валидируем конфигурацию
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// getStringConfig возвращает значение из env, если есть, иначе из флага, иначе дефолт
func getStringConfig(flagValue, envKey, defaultValue string) string {
	// Переменная окружения имеет наивысший приоритет
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}

	// Флаг второго приоритета
	if flagValue != "" {
		return flagValue
	}

	// Значение по умолчанию
	return defaultValue
}

// getIntConfig возвращает значение из env, если есть, иначе из флага или дефолт
func getIntConfig(flagValue int, envKey string, defaultValue int) int {
	// Переменная окружения имеет наивысший приоритет
	if envValue := os.Getenv(envKey); envValue != "" {
		if intValue, err := strconv.Atoi(envValue); err == nil {
			return intValue
		}
	}

	// Флаг второго приоритета
	if flagValue != 0 {
		return flagValue
	}

	// Значение по умолчанию
	return defaultValue
}
