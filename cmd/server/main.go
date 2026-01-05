package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/notifyhub/notifyhub/internal/config"
	"github.com/notifyhub/notifyhub/internal/db"
	"github.com/notifyhub/notifyhub/internal/handler"
	"github.com/notifyhub/notifyhub/internal/lib"
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

			// Криптография
			NewCrypto,

			// База данных
			NewDatabase,

			// Репозитории
			fx.Annotate(
				repository.NewAgentRepository,
				fx.As(new(repository.AgentRepositoryInterface)),
			),

			// Сервисы
			service.NewAgentService,
			service.NewCleanupService,

			// Обработчики
			handler.NewAgentHandler,
			handler.NewHealthHandler,
			handler.NewAuthMiddleware,

			// HTTP сервер
			NewRouter,
			NewHTTPServer,
		),

		// Вызываем для запуска сервера и воркеров
		fx.Invoke(func(*http.Server) {}, StartCleanupWorker),
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

// NewCrypto создает новый экземпляр Crypto
func NewCrypto(cfg *config.ServerConfig) *lib.Crypto {
	return lib.NewCrypto(lib.CryptoConfig{
		BcryptCost:       cfg.BcryptCost,
		SecretByteLength: cfg.SecretByteLength,
	})
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase(cfg *config.ServerConfig, log *logger.Logger, lc fx.Lifecycle) (*db.DB, error) {
	dbConfig := db.DBConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		PingTimeout:     cfg.DBPingTimeout,
		MigrationsPath:  cfg.DBMigrationsPath,
	}

	database, err := db.NewDB(cfg.DatabaseDSN, log, dbConfig)
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
	r.Use(middleware.Timeout(cfg.HTTPRequestTimeout))
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
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
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
			shutdownCtx, cancel := context.WithTimeout(ctx, cfg.HTTPShutdownTimeout)
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

// StartCleanupWorker запускает фоновый воркер очистки
func StartCleanupWorker(
	cleanupService *service.CleanupService,
	log *logger.Logger,
	lc fx.Lifecycle,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("starting cleanup worker")
			go func() {
				if err := cleanupService.Start(context.Background()); err != nil {
					log.WithError(err).Error("cleanup worker failed")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("stopping cleanup worker")
			cleanupService.Stop()
			return nil
		},
	})
}
