package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/notifyhub/notifyhub/internal/lib/logger"
	"github.com/notifyhub/notifyhub/internal/model"
	"github.com/notifyhub/notifyhub/internal/service"
)

type contextKey string

const (
	contextKeyAgentID contextKey = "agent_id"
)

// AuthMiddleware управляет аутентификацией агентов
type AuthMiddleware struct {
	agentService *service.AgentService
	logger       *logger.Logger
}

// NewAuthMiddleware создает новый middleware для аутентификации
func NewAuthMiddleware(agentService *service.AgentService, log *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		agentService: agentService,
		logger:       log.WithComponent("auth_middleware"),
	}
}

// AgentAuth проверяет учётные данные агента
func (m *AuthMiddleware) AgentAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Достаем учётки из заголовков
		agentID := r.Header.Get("X-Agent-ID")
		secret := r.Header.Get("X-Agent-Secret")

		// Если нет в кастомных заголовках, пробуем Bearer токен
		if agentID == "" || secret == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				secret = strings.TrimPrefix(authHeader, "Bearer ")
				// В продакшене тут надо парсить JWT и доставать agent_id
				// Пока требуем заголовок X-Agent-ID
				if agentID == "" {
					m.logger.Warn("missing agent ID in request")
					writeError(w, http.StatusUnauthorized, "missing agent credentials", "UNAUTHORIZED")
					return
				}
			}
		}

		if agentID == "" || secret == "" {
			m.logger.Warn("missing agent credentials")
			writeError(w, http.StatusUnauthorized, "missing agent credentials", "UNAUTHORIZED")
			return
		}

		// Проверяем учётки
		if err := m.agentService.VerifyAgent(r.Context(), agentID, secret); err != nil {
			if err == model.ErrAgentNotFound {
				m.logger.WithString("agent_id", agentID).Warn("agent not found")
				writeError(w, http.StatusNotFound, "agent not found", "AGENT_NOT_FOUND")
				return
			}
			if err == model.ErrUnauthorized {
				m.logger.WithString("agent_id", agentID).Warn("invalid agent credentials")
				writeError(w, http.StatusUnauthorized, "invalid credentials", "UNAUTHORIZED")
				return
			}
			m.logger.WithError(err).Error("failed to verify agent")
			writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
			return
		}

		// Добавляем ID агента в контекст
		ctx := context.WithValue(r.Context(), contextKeyAgentID, agentID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminAuth проверяет токен администратора
func (m *AuthMiddleware) AdminAuth(adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				m.logger.Warn("missing admin token")
				writeError(w, http.StatusUnauthorized, "missing admin token", "UNAUTHORIZED")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != adminToken {
				m.logger.Warn("invalid admin token")
				writeError(w, http.StatusUnauthorized, "invalid admin token", "UNAUTHORIZED")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RegistrationAuth проверяет токен регистрации агентов
func (m *AuthMiddleware) RegistrationAuth(registrationToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				m.logger.Warn("missing registration token")
				writeError(w, http.StatusUnauthorized, "missing registration token", "UNAUTHORIZED")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != registrationToken {
				m.logger.Warn("invalid registration token")
				writeError(w, http.StatusUnauthorized, "invalid registration token", "UNAUTHORIZED")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAgentIDFromContext достает ID агента из контекста запроса
func GetAgentIDFromContext(ctx context.Context) string {
	if agentID, ok := ctx.Value(contextKeyAgentID).(string); ok {
		return agentID
	}
	return ""
}
