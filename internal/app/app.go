package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 5 * time.Second

type App struct {
	cfg  *Config
	deps *Dependencies
}

func New(cfg *Config, deps *Dependencies) *App {
	return &App{cfg: cfg, deps: deps}
}

func (a *App) Run() error {
	defer a.deps.DB.Close()

	srv := &http.Server{
		Addr:    a.cfg.HTTPAddr,
		Handler: a.deps.Router,
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- srv.Serve(listener)
	}()
	slog.Info("server started", "addr", a.cfg.HTTPAddr)

	select {
	case err = <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-signalCtx.Done():
		stop()
	}
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("shutdown HTTP: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
