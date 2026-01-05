package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/notifyhub/notifyhub/internal/db"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
	"github.com/notifyhub/notifyhub/internal/model"
)

// AgentRepositoryInterface определяет интерфейс для работы с агентами
type AgentRepositoryInterface interface {
	Repository[*model.Agent]
	UpdateLastSeen(ctx context.Context, agentID string) error
}

// AgentRepository работает с агентами в базе данных
type AgentRepository struct {
	*BaseRepository[*model.Agent]
	db     *db.DB
	logger *logger.Logger
}

// NewAgentRepository создает репозиторий для работы с агентами
func NewAgentRepository(database *db.DB, log *logger.Logger) *AgentRepository {
	componentLogger := log.WithComponent("agent_repository")

	// Создаем базовый репозиторий с функциями для работы с агентами
	baseRepo := NewBaseRepository[*model.Agent](
		database,
		componentLogger,
		"agents",
		scanAgent,
		insertAgent,
		updateAgent,
	)

	return &AgentRepository{
		BaseRepository: baseRepo,
		db:             database,
		logger:         componentLogger,
	}
}

// scanAgent функция для сканирования строки БД в агента
func scanAgent(row *sql.Row) (*model.Agent, error) {
	agent := &model.Agent{}
	var metadataJSON []byte

	err := row.Scan(
		&agent.ID,
		&agent.Namespace,
		&metadataJSON,
		&agent.SecretHash,
		&agent.LastSeen,
		&agent.Status,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrAgentNotFound
		}
		return nil, fmt.Errorf("failed to scan agent: %w", err)
	}

	// Парсим JSON обратно в структуру
	if err := json.Unmarshal(metadataJSON, &agent.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return agent, nil
}

// insertAgent функция для вставки агента в БД
func insertAgent(ctx context.Context, database *db.DB, agent *model.Agent) error {
	// Превращаем метаданные в JSON для хранения
	metadataJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO agents (id, namespace, metadata, secret_hash, last_seen, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = database.ExecContext(ctx, query,
		agent.ID,
		agent.Namespace,
		metadataJSON,
		agent.SecretHash,
		agent.LastSeen,
		agent.Status,
		agent.CreatedAt,
		agent.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert agent: %w", err)
	}

	return nil
}

// updateAgent функция для обновления агента в БД
func updateAgent(ctx context.Context, database *db.DB, agent *model.Agent) error {
	// Снова превращаем метаданные в JSON
	metadataJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE agents
		SET namespace = $2, metadata = $3, last_seen = $4, status = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := database.ExecContext(ctx, query,
		agent.ID,
		agent.Namespace,
		metadataJSON,
		agent.LastSeen,
		agent.Status,
		agent.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Если ничего не обновилось - значит агента нет
	if rowsAffected == 0 {
		return model.ErrAgentNotFound
	}

	return nil
}

// UpdateLastSeen обновляет время последнего визита агента
func (r *AgentRepository) UpdateLastSeen(ctx context.Context, agentID string) error {
	query := `
		UPDATE agents
		SET last_seen = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, agentID)
	if err != nil {
		return fmt.Errorf("failed to update last_seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Агента нет - значит ошибка
	if rowsAffected == 0 {
		return model.ErrAgentNotFound
	}

	return nil
}
