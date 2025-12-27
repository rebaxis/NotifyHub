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

// AgentRepository работает с агентами в базе данных
type AgentRepository struct {
	db     *db.DB
	logger *logger.Logger
}

// NewAgentRepository создает репозиторий для работы с агентами
func NewAgentRepository(database *db.DB, log *logger.Logger) *AgentRepository {
	return &AgentRepository{
		db:     database,
		logger: log.WithComponent("agent_repository"),
	}
}

// Create добавляет нового агента в базу
func (r *AgentRepository) Create(ctx context.Context, agent *model.Agent) error {
	// Превращаем метаданные в JSON для хранения
	metadataJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO agents (id, namespace, metadata, secret_hash, last_seen, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.ExecContext(ctx, query,
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
		return fmt.Errorf("failed to create agent: %w", err)
	}

	r.logger.WithString("agent_id", agent.ID).
		WithString("namespace", agent.Namespace).
		Info("agent created")

	return nil
}

// GetByID достает агента из базы по его ID
func (r *AgentRepository) GetByID(ctx context.Context, agentID string) (*model.Agent, error) {
	query := `
		SELECT id, namespace, metadata, secret_hash, last_seen, status, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	agent := &model.Agent{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, agentID).Scan(
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
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Парсим JSON обратно в структуру
	if err := json.Unmarshal(metadataJSON, &agent.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return agent, nil
}

// Update обновляет данные существующего агента
func (r *AgentRepository) Update(ctx context.Context, agent *model.Agent) error {
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

	result, err := r.db.ExecContext(ctx, query,
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

	r.logger.WithString("agent_id", agent.ID).
		WithString("status", agent.Status).
		Info("agent updated")

	return nil
}

// Exists проверяет, есть ли агент с таким ID в базе
func (r *AgentRepository) Exists(ctx context.Context, agentID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agents WHERE id = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, agentID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check agent existence: %w", err)
	}

	return exists, nil
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
