package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Logger обертка над zap.Logger с кастомными методами
type Logger struct {
	*zap.Logger
}

// Config содержит настройки логгера
type Config struct {
	Level       string // debug, info, warn, error
	Development bool   // включить режим разработки
}

// New создает новый экземпляр логгера
func New(cfg Config) (*Logger, error) {
	var zapCfg zap.Config

	// Выбираем базовую конфигурацию
	if cfg.Development || cfg.Level == "debug" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	// Устанавливаем уровень логирования
	switch cfg.Level {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Logger{Logger: zapLogger}, nil
}

// WithComponent создает дочерний логгер с именем компонента
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.Named(component),
	}
}

// WithError добавляет поле с ошибкой к логгеру
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Error(err)),
	}
}

// WithString добавляет строковое поле
func (l *Logger) WithString(key, value string) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.String(key, value)),
	}
}

// WithInt добавляет целочисленное поле
func (l *Logger) WithInt(key string, value int) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Int(key, value)),
	}
}

// WithAny добавляет произвольное поле
func (l *Logger) WithAny(key string, value interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Any(key, value)),
	}
}

// Infof логирует форматированное info-сообщение
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Sugar().Infof(format, args...)
}

// Errorf логирует форматированное сообщение об ошибке
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Sugar().Errorf(format, args...)
}

// Fatalf логирует форматированное критическое сообщение и завершает программу
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Sugar().Fatalf(format, args...)
}
