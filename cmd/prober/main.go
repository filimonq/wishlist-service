package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/filimonq/wishlist-service/internal/prober"
)

const shutdownTimeout = 5 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				return slog.Time("timestamp", attr.Value.Time().UTC())
			case slog.MessageKey:
				attr.Key = "message"
			}
			return attr
		},
	})).With("service.name", "wishlist-prober"))
	if err := run(); err != nil {
		slog.Error("prober stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := prober.NewConfig()
	if err != nil {
		return fmt.Errorf("configure prober: %w", err)
	}
	p := prober.New(cfg)
	server := &http.Server{Addr: cfg.ListenAddr, Handler: p.Handler()}
	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen metrics: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.Serve(listener) }()
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		p.Run(ctx)
	}()
	slog.Info("prober started", "metrics_addr", cfg.ListenAddr, "interval", cfg.Interval, "timeout", cfg.Timeout)
	select {
	case err = <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	case <-ctx.Done():
	}
	stop()
	<-workerDone
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown metrics: %w", errors.Join(err, shutdownErr))
	}
	if err != nil {
		return fmt.Errorf("serve metrics: %w", err)
	}
	slog.Info("prober stopped")
	return nil
}
