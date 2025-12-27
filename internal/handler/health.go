package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/notifyhub/notifyhub/internal/db"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
)

// HealthChecker определяет интерфейс для проверки здоровья
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthHandler обрабатывает эндпоинты проверки здоровья
type HealthHandler struct {
	db     HealthChecker
	logger *logger.Logger
}

// NewHealthHandler создает новый обработчик для проверки здоровья
func NewHealthHandler(database *db.DB, log *logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     database,
		logger: log.WithComponent("health_handler"),
	}
}

// RegisterRoutes регистрирует маршруты для проверки здоровья
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/live", h.Liveness)
	r.Get("/ready", h.Readiness)
}

// Liveness обрабатывает проверку живучести
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Readiness обрабатывает проверку готовности
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.HealthCheck(ctx); err != nil {
		h.logger.WithError(err).Error("database health check failed")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "not_ready",
			"database": "disconnected",
			"error":    "failed to ping database",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ready",
		"database": "connected",
	})
}
