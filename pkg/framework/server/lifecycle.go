package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Reconciler ready, manual reconciliation available via API")
	s.reconciler.SetReady(true)

	go s.startReconciliationHandler(ctx)

	go s.startLogCleanup(ctx)

	go func() {
		s.logger.Info("Starting HTTP server", "port", s.config.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error(err, "HTTP server error")
		}
	}()

	return nil
}

func (s *Server) WaitForShutdown(ctx context.Context) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case <-sigChan:
		s.logger.Info("Shutting down...")
		return s.Shutdown(context.Background())
	case <-ctx.Done():
		s.logger.Info("Shutting down due to context cancellation...")
		return s.Shutdown(ctx)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, DefaultShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info("Shutdown complete")
	return nil
}

func (s *Server) startReconciliationHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case key := <-s.reconcileCh:
			if err := s.reconciler.ReconcileKey(ctx, key); err != nil {
				s.logger.Error(err, "failed to reconcile key from API", "key", key)
			}
		}
	}
}

func (s *Server) startLogCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.config.LogCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			before := time.Now().AddDate(0, 0, -s.config.LogRetentionDays)
			if err := s.eventStore.CleanupOldEvents(before); err != nil {
				s.logger.Error(err, "failed to cleanup old events")
			}
		}
	}
}

