// Command fiscal-api é o binário do serviço fiscal.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaltz"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/health"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/httpapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/httpserver"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/series"
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
	docsCfg, err := config.LoadDocumentsRuntime()
	if err != nil {
		logger.Error("documents_config_invalid", "error", err.Error())
		return 1
	}

	ctx := context.Background()
	sqlDB, dialect, err := openStoreDB(ctx, docsCfg)
	if err != nil {
		logger.Error("database_open_failed", "error", err.Error())
		return 1
	}
	defer sqlDB.Close()

	authenticator, err := auth.NewDevStatic(auth.DevStaticConfig{
		Token:          docsCfg.AuthDevToken,
		ScopeID:        docsCfg.AuthDevScopeID,
		ForbiddenToken: docsCfg.AuthDevForbiddenToken,
	})
	if err != nil {
		logger.Error("auth_config_invalid", "error", err.Error())
		return 1
	}
	resolver, err := series.NewStatic(series.StaticConfig{EffectiveCode: docsCfg.SeriesEffectiveCode})
	if err != nil {
		logger.Error("series_config_invalid", "error", err.Error())
		return 1
	}
	tzResolver, err := fiscaltz.NewStatic(fiscaltz.StaticConfig{
		ScopeID:  docsCfg.AuthDevScopeID,
		Timezone: docsCfg.ScopeTimezone,
	})
	if err != nil {
		logger.Error("fiscal_timezone_config_invalid", "error", err.Error())
		return 1
	}

	store := persistence.NewStore(sqlDB, dialect)
	docs := &httpapi.DocumentsHandler{
		Store:    store,
		Auth:     authenticator,
		Series:   resolver,
		FiscalTZ: tzResolver,
		Log:      logger,
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/health", health.NewHandler(cfg.Version, cfg.FiscalPackage))
	mux.Handle("/v1/documents", httpapi.WithRequestID(docs))

	srv := httpserver.New(httpserver.Config{
		Addr:              cfg.HTTPAddr,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ShutdownTimeout:   cfg.ShutdownTimeout,
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

func openStoreDB(ctx context.Context, cfg config.DocumentsRuntime) (*sql.DB, persistence.Dialect, error) {
	switch cfg.DatabaseDriver {
	case db.DriverPostgres:
		sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: cfg.DatabaseURL})
		return sqlDB, persistence.DialectPostgres, err
	case db.DriverSQLite:
		sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: cfg.DatabaseURL})
		return sqlDB, persistence.DialectSQLite, err
	default:
		return nil, "", fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
}
