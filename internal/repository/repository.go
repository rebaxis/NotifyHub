package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/notifyhub/notifyhub/internal/db"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
)

// Entity определяет базовый интерфейс для сущностей
type Entity interface {
	GetID() string
	SetID(id string)
}

// Repository определяет базовый интерфейс для всех репозиториев
type Repository[T Entity] interface {
	Create(ctx context.Context, entity T) error
	GetByID(ctx context.Context, id string) (T, error)
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// BaseRepository базовая реализация репозитория
type BaseRepository[T Entity] struct {
	db        *db.DB
	logger    *logger.Logger
	tableName string
	scanner   ScanFunc[T]
	inserter  InsertFunc[T]
	updater   UpdateFunc[T]
}

// ScanFunc функция для сканирования строки из БД в сущность
type ScanFunc[T Entity] func(row *sql.Row) (T, error)

// InsertFunc функция для вставки сущности в БД
type InsertFunc[T Entity] func(ctx context.Context, db *db.DB, entity T) error

// UpdateFunc функция для обновления сущности в БД
type UpdateFunc[T Entity] func(ctx context.Context, db *db.DB, entity T) error

// NewBaseRepository создает новый базовый репозиторий
func NewBaseRepository[T Entity](
	database *db.DB,
	log *logger.Logger,
	tableName string,
	scanner ScanFunc[T],
	inserter InsertFunc[T],
	updater UpdateFunc[T],
) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:        database,
		logger:    log,
		tableName: tableName,
		scanner:   scanner,
		inserter:  inserter,
		updater:   updater,
	}
}

// Create добавляет новую сущность в базу
func (r *BaseRepository[T]) Create(ctx context.Context, entity T) error {
	if err := r.inserter(ctx, r.db, entity); err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	r.logger.WithString("entity_id", entity.GetID()).
		WithString("table", r.tableName).
		Info("entity created")

	return nil
}

// GetByID получает сущность из базы по ID
func (r *BaseRepository[T]) GetByID(ctx context.Context, id string) (T, error) {
	query := fmt.Sprintf(`SELECT * FROM %s WHERE id = $1`, r.tableName)
	row := r.db.QueryRowContext(ctx, query, id)

	entity, err := r.scanner(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var zero T
			return zero, fmt.Errorf("entity not found")
		}
		var zero T
		return zero, fmt.Errorf("failed to get entity: %w", err)
	}

	return entity, nil
}

// Update обновляет существующую сущность
func (r *BaseRepository[T]) Update(ctx context.Context, entity T) error {
	if err := r.updater(ctx, r.db, entity); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("entity not found")
		}
		return fmt.Errorf("failed to update entity: %w", err)
	}

	r.logger.WithString("entity_id", entity.GetID()).
		WithString("table", r.tableName).
		Info("entity updated")

	return nil
}

// Delete удаляет сущность из базы
func (r *BaseRepository[T]) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, r.tableName)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity not found")
	}

	r.logger.WithString("entity_id", id).
		WithString("table", r.tableName).
		Info("entity deleted")

	return nil
}

// Exists проверяет, существует ли сущность с данным ID
func (r *BaseRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)`, r.tableName)

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check entity existence: %w", err)
	}

	return exists, nil
}
