package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/notifyhub/notifyhub/internal/config"
	"github.com/notifyhub/notifyhub/internal/db"
	"github.com/notifyhub/notifyhub/internal/handler"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
	"github.com/notifyhub/notifyhub/internal/repository"
	"github.com/notifyhub/notifyhub/internal/service"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		// Предоставляем зависимости
		fx.Provide(
			// Конфигурация
			config.LoadServerConfig,

			// Логгер
			NewLogger,

			// База данных
			NewDatabase,

			// Репозитории
			repository.NewAgentRepository,

			// Сервисы
			service.NewAgentService,

			// Обработчики
			handler.NewAgentHandler,
			handler.NewHealthHandler,
			handler.NewAuthMiddleware,

			// HTTP сервер
			NewRouter,
			NewHTTPServer,
		),

		// Вызываем для запуска сервера
		fx.Invoke(func(*http.Server) {}),
	)

	app.Run()
}

// NewLogger создает новый экземпляр логгера
func NewLogger(cfg *config.ServerConfig) (*logger.Logger, error) {
	return logger.New(logger.Config{
		Level:       cfg.LogLevel,
		Development: cfg.LogLevel == "debug",
	})
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase(cfg *config.ServerConfig, log *logger.Logger, lc fx.Lifecycle) (*db.DB, error) {
	database, err := db.NewDB(cfg.DatabaseDSN, log)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Info("closing database connection")
			return database.Close()
		},
	})

	return database, nil
}

// NewRouter создает HTTP-роутер со всеми маршрутами
func NewRouter(
	cfg *config.ServerConfig,
	agentHandler *handler.AgentHandler,
	healthHandler *handler.HealthHandler,
	authMiddleware *handler.AuthMiddleware,
	log *logger.Logger,
) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(logger.HTTPMiddleware(log))

	// Регистрируем маршруты
	healthHandler.RegisterRoutes(r)
	agentHandler.RegisterRoutes(r, authMiddleware, cfg.AgentRegistrationToken)

	log.Info("routes registered successfully")

	return r
}

// NewHTTPServer создает HTTP-сервер
func NewHTTPServer(
	cfg *config.ServerConfig,
	router chi.Router,
	log *logger.Logger,
	lc fx.Lifecycle,
) *http.Server {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Infof("starting http server on port %d", cfg.Port)
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.WithError(err).Fatal("http server failed")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("stopping http server gracefully")
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.WithError(err).Error("http server shutdown failed")
				return err
			}

			log.Info("http server stopped successfully")
			return nil
		},
	})

	return srv
}
