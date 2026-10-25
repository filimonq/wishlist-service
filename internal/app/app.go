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
	servers := []*http.Server{
		{Addr: a.cfg.HTTPAddr, Handler: a.deps.Router},
		{Addr: a.cfg.MetricsAddr, Handler: a.deps.Metrics},
	}
	listeners, err := listenServers(servers)
	if err != nil {
		return err
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, len(servers))
	for i, srv := range servers {
		go func() {
			serveErr := srv.Serve(listeners[i])
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				serverErrors <- fmt.Errorf("serve %s: %w", srv.Addr, serveErr)
			} else {
				serverErrors <- nil
			}
		}()
	}
	slog.Info("server started", "addr", a.cfg.HTTPAddr, "metrics_addr", a.cfg.MetricsAddr)
	select {
	case err = <-serverErrors:
	case <-signalCtx.Done():
		stop()
	}
	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err = errors.Join(err, shutdownServers(ctx, servers))
	if err != nil {
		return err
	}
	slog.Info("server stopped")
	return nil
}

func listenServers(servers []*http.Server) ([]net.Listener, error) {
	var config net.ListenConfig
	listeners := make([]net.Listener, 0, len(servers))
	for _, srv := range servers {
		listener, err := config.Listen(context.Background(), "tcp", srv.Addr)
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			return nil, fmt.Errorf("listen %s: %w", srv.Addr, err)
		}
		listeners = append(listeners, listener)
	}
	return listeners, nil
}

func shutdownServers(ctx context.Context, servers []*http.Server) error {
	var result error
	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			_ = srv.Close()
			result = errors.Join(result, fmt.Errorf("shutdown %s: %w", srv.Addr, err))
		}
	}
	return result
}
