package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
)

// ServerConfig содержит настройки сервера
type ServerConfig struct {
	// Server settings
	Port                   int    `validate:"required,min=1,max=65535"`
	DatabaseDSN            string `validate:"required"`
	AdminToken             string `validate:"required"`
	AgentRegistrationToken string `validate:"required"`
	LogLevel               string `validate:"required,oneof=debug info warn error"`

	// HTTP Server timeouts
	HTTPRequestTimeout  time.Duration `validate:"required"`
	HTTPReadTimeout     time.Duration `validate:"required"`
	HTTPWriteTimeout    time.Duration `validate:"required"`
	HTTPIdleTimeout     time.Duration `validate:"required"`
	HTTPShutdownTimeout time.Duration `validate:"required"`

	// Database connection pool settings
	DBMaxOpenConns    int           `validate:"required,min=1"`
	DBMaxIdleConns    int           `validate:"required,min=1"`
	DBConnMaxLifetime time.Duration `validate:"required"`
	DBPingTimeout     time.Duration `validate:"required"`
	DBMigrationsPath  string        `validate:"required"`

	// Health check settings
	HealthCheckTimeout time.Duration `validate:"required"`

	// Cleanup service settings
	CleanupInterval             time.Duration `validate:"required"`
	NotificationRetentionPeriod time.Duration `validate:"required"`
	AgentStaleTimeout           time.Duration `validate:"required"`

	// Security settings
	BcryptCost       int `validate:"required,min=4,max=31"`
	SecretByteLength int `validate:"required,min=16,max=64"`
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

	// HTTP Server timeouts
	cfg.HTTPRequestTimeout = getDurationConfig("HTTP_REQUEST_TIMEOUT", 60*time.Second)
	cfg.HTTPReadTimeout = getDurationConfig("HTTP_READ_TIMEOUT", 15*time.Second)
	cfg.HTTPWriteTimeout = getDurationConfig("HTTP_WRITE_TIMEOUT", 15*time.Second)
	cfg.HTTPIdleTimeout = getDurationConfig("HTTP_IDLE_TIMEOUT", 60*time.Second)
	cfg.HTTPShutdownTimeout = getDurationConfig("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second)

	// Database connection pool settings
	cfg.DBMaxOpenConns = getIntConfig(0, "DB_MAX_OPEN_CONNS", 25)
	cfg.DBMaxIdleConns = getIntConfig(0, "DB_MAX_IDLE_CONNS", 5)
	cfg.DBConnMaxLifetime = getDurationConfig("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	cfg.DBPingTimeout = getDurationConfig("DB_PING_TIMEOUT", 5*time.Second)
	cfg.DBMigrationsPath = getStringConfig("", "DB_MIGRATIONS_PATH", "file://migrations")

	// Health check settings
	cfg.HealthCheckTimeout = getDurationConfig("HEALTH_CHECK_TIMEOUT", 2*time.Second)

	// Cleanup service settings
	cfg.CleanupInterval = getDurationConfig("CLEANUP_INTERVAL", 5*time.Minute)
	cfg.NotificationRetentionPeriod = getDurationConfig("NOTIFICATION_RETENTION_PERIOD", 7*24*time.Hour) // 7 days
	cfg.AgentStaleTimeout = getDurationConfig("AGENT_STALE_TIMEOUT", 30*time.Minute)

	// Security settings
	cfg.BcryptCost = getIntConfig(0, "BCRYPT_COST", 10) // bcrypt.DefaultCost = 10
	cfg.SecretByteLength = getIntConfig(0, "SECRET_BYTE_LENGTH", 32)

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

// getDurationConfig возвращает значение из env, если есть, иначе дефолт
func getDurationConfig(envKey string, defaultValue time.Duration) time.Duration {
	// Переменная окружения имеет наивысший приоритет
	if envValue := os.Getenv(envKey); envValue != "" {
		if duration, err := time.ParseDuration(envValue); err == nil {
			return duration
		}
	}

	// Значение по умолчанию
	return defaultValue
}
