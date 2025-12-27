package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
	"github.com/notifyhub/notifyhub/internal/model"
	"github.com/notifyhub/notifyhub/internal/service"
)

// AgentHandler обрабатывает HTTP-запросы для работы с агентами
type AgentHandler struct {
	service  *service.AgentService
	logger   *logger.Logger
	validate *validator.Validate
}

// NewAgentHandler создает новый обработчик для агентов
func NewAgentHandler(service *service.AgentService, log *logger.Logger) *AgentHandler {
	return &AgentHandler{
		service:  service,
		logger:   log.WithComponent("agent_handler"),
		validate: validator.New(),
	}
}

// RegisterRoutes регистрирует маршруты для работы с агентами
func (h *AgentHandler) RegisterRoutes(r chi.Router, authMiddleware *AuthMiddleware, registrationToken string) {
	// Эндпоинт регистрации - требует токен регистрации
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RegistrationAuth(registrationToken))
		r.Post("/api/v1/agents/register", h.Register)
	})

	// Защищенные эндпоинты - требуют аутентификации агента
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.AgentAuth)
		r.Put("/api/v1/agents/{agent_id}", h.Update)
		r.Get("/api/v1/agents/{agent_id}/notifications", h.GetNotifications)
	})
}

// Register обрабатывает регистрацию нового агента
func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Декодируем запрос
	var req model.RegisterAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		h.logger.WithError(err).Warn("failed to decode request")
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	// Валидируем запрос
	if err := h.validate.Struct(&req); err != nil {
		h.logger.WithError(err).Warn("request validation failed")

		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			details := make(map[string]interface{})
			for _, fieldErr := range validationErrors {
				details[fieldErr.Field()] = fieldErr.Tag()
			}
			writeErrorWithDetails(w, http.StatusBadRequest, "validation failed", "VALIDATION_ERROR", details)
			return
		}

		writeError(w, http.StatusBadRequest, "validation failed", "VALIDATION_ERROR")
		return
	}

	// Регистрируем агента
	resp, err := h.service.RegisterAgent(r.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrAgentAlreadyExists) {
			h.logger.WithString("agent_id", req.AgentID).Warn("agent already exists")
			writeError(w, http.StatusConflict, "agent already exists", "AGENT_EXISTS")
			return
		}

		h.logger.WithError(err).Error("failed to register agent")
		writeError(w, http.StatusInternalServerError, "failed to register agent", "INTERNAL_ERROR")
		return
	}

	h.logger.WithString("agent_id", resp.AgentID).
		WithString("namespace", req.Namespace).
		Info("agent registered")

	writeJSON(w, http.StatusCreated, resp)
}

// Update обрабатывает обновление метаданных агента
func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Достаем ID агента из URL
	agentID := chi.URLParam(r, "agent_id")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "missing agent_id", "INVALID_REQUEST")
		return
	}

	// Проверяем, что агент в контексте соответствует URL
	ctxAgentID := GetAgentIDFromContext(r.Context())
	if ctxAgentID != agentID {
		h.logger.WithString("context_agent_id", ctxAgentID).
			WithString("url_agent_id", agentID).
			Warn("agent ID mismatch")
		writeError(w, http.StatusForbidden, "forbidden", "FORBIDDEN")
		return
	}

	// Декодируем запрос
	var req model.UpdateAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		h.logger.WithError(err).Warn("failed to decode request")
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	// Валидируем запрос
	if err := h.validate.Struct(&req); err != nil {
		h.logger.WithError(err).Warn("request validation failed")
		writeError(w, http.StatusBadRequest, "validation failed", "VALIDATION_ERROR")
		return
	}

	// Обновляем агента
	if err := h.service.UpdateAgent(r.Context(), agentID, &req); err != nil {
		if errors.Is(err, model.ErrAgentNotFound) {
			writeError(w, http.StatusNotFound, "agent not found", "AGENT_NOT_FOUND")
			return
		}

		h.logger.WithError(err).Error("failed to update agent")
		writeError(w, http.StatusInternalServerError, "failed to update agent", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// GetNotifications обрабатывает получение уведомлений для агента
func (h *AgentHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	// Достаем ID агента из URL
	agentID := chi.URLParam(r, "agent_id")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "missing agent_id", "INVALID_REQUEST")
		return
	}

	// Проверяем, что агент в контексте соответствует URL
	ctxAgentID := GetAgentIDFromContext(r.Context())
	if ctxAgentID != agentID {
		h.logger.WithString("context_agent_id", ctxAgentID).
			WithString("url_agent_id", agentID).
			Warn("agent ID mismatch")
		writeError(w, http.StatusForbidden, "forbidden", "FORBIDDEN")
		return
	}

	// Получаем оповещения через сервис
	notifications, err := h.service.GetNotifications(r.Context(), agentID)
	if err != nil {
		if errors.Is(err, model.ErrAgentNotFound) {
			writeError(w, http.StatusNotFound, "agent not found", "AGENT_NOT_FOUND")
			return
		}

		h.logger.WithError(err).Error("failed to get notifications")
		writeError(w, http.StatusInternalServerError, "failed to get notifications", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, notifications)
}
