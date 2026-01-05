package service

import (
	"context"
	"fmt"
	"time"

	"github.com/notifyhub/notifyhub/internal/config"
	"github.com/notifyhub/notifyhub/internal/db"
	"github.com/notifyhub/notifyhub/internal/lib/logger"
	"golang.org/x/sync/errgroup"
)

// CleanupService управляет фоновыми задачами очистки
type CleanupService struct {
	db                    *db.DB
	logger                *logger.Logger
	interval              time.Duration
	notificationRetention time.Duration
	agentStaleTimeout     time.Duration
	stopChan              chan struct{}
}

// NewCleanupService создает новый сервис очистки
func NewCleanupService(database *db.DB, log *logger.Logger, cfg *config.ServerConfig) *CleanupService {
	return &CleanupService{
		db:                    database,
		logger:                log.WithComponent("cleanup_service"),
		interval:              cfg.CleanupInterval,
		notificationRetention: cfg.NotificationRetentionPeriod,
		agentStaleTimeout:     cfg.AgentStaleTimeout,
		stopChan:              make(chan struct{}),
	}
}

// Start запускает фоновый воркер очистки
func (s *CleanupService) Start(ctx context.Context) error {
	s.logger.Info("starting cleanup worker")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Запускаем первую очистку сразу
	if err := s.runCleanup(ctx); err != nil {
		s.logger.WithError(err).Error("initial cleanup failed")
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("cleanup worker stopped by context")
			return ctx.Err()
		case <-s.stopChan:
			s.logger.Info("cleanup worker stopped")
			return nil
		case <-ticker.C:
			if err := s.runCleanup(ctx); err != nil {
				s.logger.WithError(err).Error("cleanup cycle failed")
			}
		}
	}
}

// Stop останавливает фоновый воркер
func (s *CleanupService) Stop() {
	close(s.stopChan)
}

// runCleanup выполняет цикл очистки
func (s *CleanupService) runCleanup(ctx context.Context) error {
	startTime := time.Now()
	s.logger.Info("starting cleanup cycle")

	g, ctx := errgroup.WithContext(ctx)

	var deletedNotifications int64
	var staleAgents int64

	// Параллельно выполняем задачи очистки
	g.Go(func() error {
		deleted, err := s.cleanupExpiredNotifications(ctx)
		if err != nil {
			return fmt.Errorf("failed to cleanup notifications: %w", err)
		}
		deletedNotifications = deleted
		return nil
	})

	g.Go(func() error {
		updated, err := s.markStaleAgents(ctx)
		if err != nil {
			return fmt.Errorf("failed to mark stale agents: %w", err)
		}
		staleAgents = updated
		return nil
	})

	// Ждем завершения всех задач
	if err := g.Wait(); err != nil {
		return err
	}

	duration := time.Since(startTime)
	s.logger.
		WithInt("deleted_notifications", int(deletedNotifications)).
		WithInt("stale_agents", int(staleAgents)).
		WithString("duration", duration.String()).
		Info("cleanup cycle completed")

	return nil
}

// cleanupExpiredNotifications удаляет истекшие уведомления
func (s *CleanupService) cleanupExpiredNotifications(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM notifications
		WHERE expires_at < NOW() - $1::interval
	`

	// Форматируем duration в PostgreSQL interval
	retentionInterval := fmt.Sprintf("%d seconds", int(s.notificationRetention.Seconds()))

	result, err := s.db.ExecContext(ctx, query, retentionInterval)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired notifications: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if deleted > 0 {
		s.logger.WithInt("count", int(deleted)).Info("deleted expired notifications")
	}

	return deleted, nil
}

// markStaleAgents помечает агентов как устаревшие
func (s *CleanupService) markStaleAgents(ctx context.Context) (int64, error) {
	// Используем параметризованный запрос с настраиваемым временем устаревания
	query := `
		UPDATE agents
		SET status = 'stale', updated_at = NOW()
		WHERE last_seen < NOW() - $1::interval
		  AND status != 'stale'
	`

	// Форматируем duration в PostgreSQL interval
	staleInterval := fmt.Sprintf("%d seconds", int(s.agentStaleTimeout.Seconds()))

	result, err := s.db.ExecContext(ctx, query, staleInterval)
	if err != nil {
		return 0, fmt.Errorf("failed to mark stale agents: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if updated > 0 {
		s.logger.WithInt("count", int(updated)).Info("marked agents as stale")
	}

	return updated, nil
}
