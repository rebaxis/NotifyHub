package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/notifyhub/notifyhub/internal/lib"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
	"github.com/notifyhub/notifyhub/internal/model"
	"github.com/notifyhub/notifyhub/internal/repository"
)

// AgentService управляет бизнес-логикой агентов
type AgentService struct {
	repo   repository.AgentRepositoryInterface
	logger *logger.Logger
}

// NewAgentService создает новый сервис для работы с агентами
func NewAgentService(repo repository.AgentRepositoryInterface, log *logger.Logger) (*AgentService, error) {
	return &AgentService{
		repo:   repo,
		logger: log.WithComponent("agent_service"),
	}, nil
}

// RegisterAgent регистрирует нового агента в системе
func (s *AgentService) RegisterAgent(ctx context.Context, req *model.RegisterAgentRequest) (*model.RegisterAgentResponse, error) {
	// Генерируем ID агента, если не передали
	agentID := req.AgentID
	if agentID == "" {
		agentID = lib.GenerateAgentID()
		s.logger.WithString("agent_id", agentID).Info("generated agent ID")
	}

	// Проверяем, нет ли такого агента уже
	exists, err := s.repo.Exists(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check agent existence: %w", err)
	}
	if exists {
		return nil, model.ErrAgentAlreadyExists
	}

	// Генерируем секретный ключ
	secret, err := lib.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// Хешируем секрет для хранения в базе
	secretHash := lib.HashSecret(secret)

	// Собираем объект агента
	now := time.Now()
	agent := &model.Agent{
		ID:         agentID,
		Namespace:  req.Namespace,
		Metadata:   req.Metadata,
		SecretHash: secretHash,
		LastSeen:   now,
		Status:     model.AgentStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Сохраняем в базу
	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	s.logger.WithString("agent_id", agentID).
		WithString("namespace", req.Namespace).
		Info("agent registered successfully")

	return &model.RegisterAgentResponse{
		AgentID: agentID,
		Secret:  secret,
	}, nil
}

// UpdateAgent обновляет метаданные агента и время последнего визита
func (s *AgentService) UpdateAgent(ctx context.Context, agentID string, req *model.UpdateAgentRequest) error {
	// Сначала достаем текущего агента
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		// Пробрасываем ErrAgentNotFound без обертки
		if errors.Is(err, model.ErrAgentNotFound) {
			return model.ErrAgentNotFound
		}
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Обновляем поля
	agent.Metadata = req.Metadata
	agent.LastSeen = time.Now()
	agent.UpdatedAt = time.Now()

	if req.Status != "" {
		agent.Status = req.Status
	}

	// Сохраняем изменения
	if err := s.repo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	s.logger.WithString("agent_id", agentID).Info("agent updated")

	return nil
}

// VerifyAgent проверяет учётные данные агента
func (s *AgentService) VerifyAgent(ctx context.Context, agentID, secret string) error {
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		// Пробрасываем ErrAgentNotFound без обертки
		if errors.Is(err, model.ErrAgentNotFound) {
			return model.ErrAgentNotFound
		}
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if !lib.VerifySecret(secret, agent.SecretHash) {
		s.logger.WithString("agent_id", agentID).Warn("invalid agent secret")
		return model.ErrUnauthorized
	}

	return nil
}

// GetNotifications возвращает список оповещений для агента
func (s *AgentService) GetNotifications(ctx context.Context, agentID string) ([]interface{}, error) {
	// Сначала проверяем, что агент существует
	_, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		// Пробрасываем ErrAgentNotFound без обертки
		if errors.Is(err, model.ErrAgentNotFound) {
			return nil, model.ErrAgentNotFound
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// TODO: Реализовать получение оповещений из базы
	// Пока возвращаем пустой массив
	return []interface{}{}, nil
}
