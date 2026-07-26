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
	"strings"
	"syscall"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminui"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/buildinfo"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/health"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/httpapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/httpserver"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		rev := buildinfo.Revision
		if err := buildinfo.Validate(rev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		version := os.Getenv("FISCAL_APP_VERSION")
		if version == "" {
			version = "0.0.0-dev"
		}
		fmt.Printf("version=%s revision=%s\n", version, rev)
		return 0
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := buildinfo.Validate(buildinfo.Revision); err != nil {
		logger.Error("buildinfo_invalid", "error", err.Error())
		return 1
	}

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
	if err := buildinfo.ValidateForEnv(buildinfo.Revision, docsCfg.Env); err != nil {
		logger.Error("buildinfo_env_invalid", "error", err.Error())
		return 1
	}

	ctx := context.Background()
	sqlDB, dialect, err := openStoreDB(ctx, docsCfg)
	if err != nil {
		logger.Error("database_open_failed", "error", err.Error())
		return 1
	}
	defer sqlDB.Close()

	authenticator, auditor, err := buildAuthenticator(docsCfg, sqlDB, dialect)
	if err != nil {
		logger.Error("auth_config_invalid", "error", err.Error())
		return 1
	}

	store := persistence.NewStore(sqlDB, dialect)
	docs := &httpapi.DocumentsHandler{
		Store:       store,
		Auth:        authenticator,
		Log:         logger,
		AuthAuditor: auditor,
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/health", health.NewHandler(cfg.Version, buildinfo.Revision, cfg.FiscalPackage))
	mux.Handle("/v1/documents", httpapi.WithRequestID(docs))

	adminAuthn, err := buildAdminAuthenticator(docsCfg)
	if err != nil {
		logger.Error("admin_auth_config_invalid", "error", err.Error())
		return 1
	}
	regDialect := adminregistry.DialectSQLite
	auditDialect := adminaudit.DialectSQLite
	opsDialect := adminops.DialectSQLite
	if dialect == persistence.DialectPostgres {
		regDialect = adminregistry.DialectPostgres
		auditDialect = adminaudit.DialectPostgres
		opsDialect = adminops.DialectPostgres
	}
	secretsMeta, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		logger.Error("secretstore_init_failed", "error", err.Error())
		return 1
	}
	var secGate *secadm.Gate
	if ownerID := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_OWNER_SUBJECT")); ownerID != "" {
		secGate, err = secadm.NewGate(ownerID, secretsMeta)
		if err != nil {
			logger.Error("secadm_gate_invalid", "error", err.Error())
			return 1
		}
	}
	registry := adminregistry.New(sqlDB, regDialect, nil)
	auditStore := adminaudit.New(sqlDB, auditDialect, nil)
	opsStore := adminops.New(sqlDB, opsDialect)
	adminapi.Mount(mux, adminAuthn, &adminapi.Handler{
		Registry:    registry,
		Audit:       auditStore,
		Ops:         opsStore,
		SecretsMeta: secretsMeta,
		SecAdm:      secGate,
	})

	uiHandler, err := adminui.New(registry, docsCfg.Env)
	if err != nil {
		logger.Error("adminui_init_failed", "error", err.Error())
		return 1
	}
	uiHandler.Ops = opsStore
	uiHandler.Audit = auditStore
	injectSubject := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_INJECT_SUBJECT"))
	var injectRoles []adminauth.Role
	if docsCfg.Env == config.EnvDevelopment() && strings.TrimSpace(os.Getenv("FISCAL_ADMIN_AUTH_MODE")) == "injected" {
		injectRoles, _ = adminauth.ParseRoles(os.Getenv("FISCAL_ADMIN_INJECT_ROLES"))
	}
	uiAuth := adminui.BuildUIAuthenticator(adminAuthn, docsCfg.Env, injectSubject, injectRoles)
	adminui.Mount(mux, uiAuth, uiHandler)

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

func buildAdminAuthenticator(docsCfg config.DocumentsRuntime) (adminauth.Authenticator, error) {
	mode := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_AUTH_MODE"))
	if mode == "" || mode == "fail_closed" {
		return adminauth.FailClosedAuthenticator{}, nil
	}
	if mode != "injected" {
		return nil, fmt.Errorf("FISCAL_ADMIN_AUTH_MODE=%q inválido (fail_closed|injected)", mode)
	}
	if docsCfg.Env != config.EnvDevelopment() {
		return nil, fmt.Errorf("FISCAL_ADMIN_AUTH_MODE=injected só permitido com FISCAL_ENV=development")
	}
	subject := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_INJECT_SUBJECT"))
	roles, err := adminauth.ParseRoles(os.Getenv("FISCAL_ADMIN_INJECT_ROLES"))
	if err != nil {
		return nil, fmt.Errorf("FISCAL_ADMIN_INJECT_ROLES: %w", err)
	}
	if subject == "" {
		return nil, fmt.Errorf("FISCAL_ADMIN_INJECT_SUBJECT obrigatório quando mode=injected")
	}
	return adminauth.StaticAuthenticator{Claims: adminauth.Claims{Subject: subject, Roles: roles}}, nil
}

func buildAuthenticator(docsCfg config.DocumentsRuntime, sqlDB *sql.DB, dialect persistence.Dialect) (auth.Authenticator, httpapi.AuthAuditor, error) {
	switch docsCfg.AuthMode {
	case config.AuthModeDevStatic():
		a, err := auth.NewDevStatic(auth.DevStaticConfig{
			Token:               docsCfg.AuthDevToken,
			ScopeID:             docsCfg.AuthDevScopeID,
			ForbiddenToken:      docsCfg.AuthDevForbiddenToken,
			TaxpayerNIF:         docsCfg.AuthDevTaxpayerNIF,
			IANATimezone:        docsCfg.ScopeTimezone,
			SeriesEffectiveCode: docsCfg.SeriesEffectiveCode,
			Environment:         docsCfg.Env,
		})
		return a, nil, err
	case config.AuthModeCredentialStore():
		creds := persistence.NewCredentialStore(sqlDB, dialect)
		a, err := auth.NewCredentialStoreAuthenticator(creds, docsCfg.Env)
		if err != nil {
			return nil, nil, err
		}
		return a, creds, nil
	default:
		return nil, nil, fmt.Errorf("unsupported auth mode %q", docsCfg.AuthMode)
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
