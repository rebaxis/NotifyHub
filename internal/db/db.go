package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
)

// DB оборачивает подключение к базе данных
type DB struct {
	*sql.DB
	logger *logger.Logger
}

// NewDB создает новое подключение к базе данных и запускает миграции
func NewDB(dsn string, log *logger.Logger) (*DB, error) {
	// Открываем подключение к базе
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настраиваем пул соединений
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection established successfully")

	db := &DB{
		DB:     sqlDB,
		logger: log.WithComponent("database"),
	}

	// Запускаем миграции
	if err := db.runMigrations(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// runMigrations применяет миграции базы данных
func (db *DB) runMigrations() error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		db.logger.Info("no new migrations to apply")
	} else {
		db.logger.Info("migrations applied successfully")
	}

	return nil
}

// Close закрывает подключение к базе данных
func (db *DB) Close() error {
	db.logger.Info("closing database connection")
	return db.DB.Close()
}

// HealthCheck проверяет доступность базы данных
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.PingContext(ctx)
}
