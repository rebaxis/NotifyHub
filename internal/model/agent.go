package model

import "time"

// Agent представляет сущность агента в системе
type Agent struct {
	ID         string                 `json:"id"`
	Namespace  string                 `json:"namespace"`
	Metadata   map[string]interface{} `json:"metadata"`
	SecretHash []byte                 `json:"-"`
	LastSeen   time.Time              `json:"last_seen"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// GetID возвращает ID агента (реализация интерфейса Entity)
func (a *Agent) GetID() string {
	return a.ID
}

// SetID устанавливает ID агента (реализация интерфейса Entity)
func (a *Agent) SetID(id string) {
	a.ID = id
}

// Константы статусов агента
const (
	AgentStatusActive  = "active"
	AgentStatusStale   = "stale"
	AgentStatusRevoked = "revoked"
)

// RegisterAgentRequest представляет запрос на регистрацию нового агента
type RegisterAgentRequest struct {
	AgentID   string                 `json:"agent_id" validate:"omitempty,min=1,max=255"`
	Namespace string                 `json:"namespace" validate:"required,min=1,max=255"`
	Metadata  map[string]interface{} `json:"metadata" validate:"required"`
}

// RegisterAgentResponse представляет ответ после успешной регистрации
type RegisterAgentResponse struct {
	AgentID string `json:"agent_id"`
	Secret  string `json:"secret"`
}

// UpdateAgentRequest представляет запрос на обновление метаданных агента
type UpdateAgentRequest struct {
	Metadata map[string]interface{} `json:"metadata" validate:"required"`
	Status   string                 `json:"status" validate:"omitempty,oneof=active offline"`
}
