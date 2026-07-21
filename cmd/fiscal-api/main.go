// Command fiscal-api é o binário mínimo do serviço fiscal (scaffold Fase 1).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/health"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/httpserver"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config_invalid", "error", err.Error())
		return 1
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/health", health.NewHandler(cfg.Version, cfg.FiscalPackage))

	srv := httpserver.New(httpserver.Config{
		Addr:            cfg.HTTPAddr,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		IdleTimeout:     cfg.IdleTimeout,
		ShutdownTimeout: cfg.ShutdownTimeout,
	}, mux, logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("http_listen_failed", "error", err.Error())
			return 1
		}
		return 0
	case sig := <-sigCh:
		logger.Info("shutdown_signal", "signal", sig.String())
		if err := srv.Shutdown(context.Background()); err != nil {
			return 1
		}
		if err := <-errCh; err != nil {
			logger.Error("http_listen_failed", "error", err.Error())
			return 1
		}
		return 0
	}
}
