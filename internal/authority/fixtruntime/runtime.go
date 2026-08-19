package fixtruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fefixqueue"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fehub"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/femock"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
)

var ErrNotConfigured = errors.New("fixtruntime: workbook not configured")

const (
	defaultMockUser     = "bwb-fixture-mock"
	defaultWorkerPeriod = 2 * time.Second
)

// Config holds runtime wiring inputs.
type Config struct {
	WorkbookPath   string
	MockUser       string
	MockPassword   string
	WorkerInterval time.Duration
	Logger         *slog.Logger
}

// Status is a sanitized runtime view for admin (no credentials / PEM).
type Status struct {
	Configured       bool
	MockLoopback     string
	IdentityCount    int
	WorkerInterval   time.Duration
	ExternalVerified bool
	MockOnly         bool
	Note             string
}

// Runtime owns loopback femock, feboundary engine, and SQL fixture queue.
type Runtime struct {
	Queue    *fefixqueue.Store
	Engine   *feboundary.Engine
	Provider agttestkit.IdentityProvider

	mockHost string
	httpSrv  *http.Server
	interval time.Duration

	mu     sync.Mutex
	closed bool
	cancel context.CancelFunc
	logger *slog.Logger
}

// Open starts loopback femock + feboundary when workbook path is set.
func Open(workbookPath string, sqlDB *sql.DB, dialect persistence.Dialect, cfg Config) (*Runtime, error) {
	path := strings.TrimSpace(workbookPath)
	if path == "" {
		return nil, ErrNotConfigured
	}
	if sqlDB == nil {
		return nil, fmt.Errorf("fixtruntime: database required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	user := strings.TrimSpace(cfg.MockUser)
	if user == "" {
		user = defaultMockUser
	}
	pass := strings.TrimSpace(cfg.MockPassword)
	if pass == "" {
		var err error
		pass, err = randomPass()
		if err != nil {
			return nil, err
		}
	}
	interval := cfg.WorkerInterval
	if interval <= 0 {
		interval = defaultWorkerPeriod
	}

	provider, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		return nil, fmt.Errorf("fixtruntime: workbook: %w", err)
	}

	mock, err := femock.New(femock.Config{Username: user, Password: pass, Provider: provider})
	if err != nil {
		_ = provider.Close()
		return nil, fmt.Errorf("fixtruntime: femock: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = mock.Close()
		_ = provider.Close()
		return nil, fmt.Errorf("fixtruntime: listen: %w", err)
	}
	srv := &http.Server{
		Handler:           mock.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cfg.Logger.Error("fixture_mock_http_failed", "error", err.Error())
		}
	}()

	baseURL := "http://" + ln.Addr().String()
	eng, err := feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: provider,
		BaseURL: baseURL, Username: user, Password: pass,
		Client: &http.Client{Timeout: 30 * time.Second},
	})
	if err != nil {
		_ = srv.Close()
		_ = mock.Close()
		_ = provider.Close()
		return nil, fmt.Errorf("fixtruntime: feboundary: %w", err)
	}

	qDialect := fefixqueue.DialectSQLite
	if dialect == persistence.DialectPostgres {
		qDialect = fefixqueue.DialectPostgres
	}

	return &Runtime{
		Queue:    fefixqueue.NewStore(sqlDB, qDialect),
		Engine:   eng,
		Provider: provider,
		mockHost: ln.Addr().String(),
		httpSrv:  srv,
		interval: interval,
		logger:   cfg.Logger,
	}, nil
}

// Status returns sanitized runtime metadata.
func (r *Runtime) Status() Status {
	if r == nil {
		return Status{Configured: false, MockOnly: true, Note: "workbook path not configured"}
	}
	count := 0
	if r.Provider != nil {
		count = len(r.Provider.List())
	}
	return Status{
		Configured: true, MockLoopback: r.mockHost, IdentityCount: count,
		WorkerInterval: r.interval, ExternalVerified: false, MockOnly: true,
		Note: "workbook→fefixqueue→BWB-MOCK; ≠ AGT HML/PRD; ≠ authority_accepted",
	}
}

// StartWorker polls the SQL queue until ctx is cancelled.
func (r *Runtime) StartWorker(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.workerLoop(ctx)
}

func (r *Runtime) workerLoop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = r.ProcessOne(ctx)
		}
	}
}

// ProcessOne drains at most one queued row.
func (r *Runtime) ProcessOne(ctx context.Context) (*fefixqueue.ProcessResult, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, feboundary.ErrClosed
	}
	r.mu.Unlock()

	out, err := r.Queue.ProcessNext(ctx, r.Engine)
	if errors.Is(err, fefixqueue.ErrEmpty) {
		return nil, fefixqueue.ErrEmpty
	}
	if err != nil {
		r.logger.Info("fixture_process_error", "error", err.Error())
		return out, err
	}
	if out != nil {
		r.logger.Info("fixture_processed",
			"row_id", out.RowID, "operation", out.Operation, "state", out.State, "retried", out.Retried)
	}
	return out, nil
}

// Close stops worker, HTTP server, engine, and provider.
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	if r.cancel != nil {
		r.cancel()
	}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var err error
	if r.httpSrv != nil {
		err = r.httpSrv.Shutdown(ctx)
	}
	if r.Engine != nil {
		_ = r.Engine.Close()
	}
	if r.Provider != nil {
		_ = r.Provider.Close()
	}
	return err
}

func randomPass() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
