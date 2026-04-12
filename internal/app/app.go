package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
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

func (a *App) Run() {
	srv := &http.Server{
		Addr:    a.cfg.HTTPAddr,
		Handler: a.deps.Router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server started", "addr", a.cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	a.deps.DB.Close()
	slog.Info("server stopped")
}
