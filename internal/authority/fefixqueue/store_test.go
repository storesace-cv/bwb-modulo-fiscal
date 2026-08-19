package fefixqueue_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fefixqueue"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fehub"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/femock"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
)

const mockUser = "bwb-mock-user-queue"
const mockPass = "bwb-mock-pass-queue"

func TestFixtureQueueSQLiteE2E(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fefixqueue.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runFixtureQueueSuite(t, ctx, fefixqueue.NewStore(sqlDB, fefixqueue.DialectSQLite), sqlDB)
}

func TestFixtureQueuePostgresE2E(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runFixtureQueueSuite(t, ctx, fefixqueue.NewStore(sqlDB, fefixqueue.DialectPostgres), sqlDB)
}

func runFixtureQueueSuite(t *testing.T, ctx context.Context, store *fefixqueue.Store, _ *sql.DB) {
	t.Helper()
	eng, provider, _, cleanup := setupEngine(t)
	defer cleanup()

	refs := provider.List()
	if len(refs) == 0 {
		t.Fatal("no identities")
	}
	ref := refs[0].Ref

	t.Run("E2E_obterEstado_ok", func(t *testing.T) {
		idem := fmt.Sprintf("idem-obter-%d", time.Now().UnixNano())
		row, err := store.Enqueue(ctx, fefixqueue.EnqueueInput{
			Operation: feboundary.OpObterEstado, IdentityRef: ref, IdempotencyKey: idem,
			Payload: fefixqueue.Payload{
				ObterEstado: &feprofile.ObterEstadoRequestClaims{
					TaxRegistrationNumber: "9100000000",
					RequestID:             "req-e2e-001",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if row.State != feboundary.StateQueued {
			t.Fatalf("state=%q", row.State)
		}
		out, err := store.ProcessNext(ctx, eng)
		if err != nil {
			t.Fatal(err)
		}
		if out.State != feboundary.StateOK {
			t.Fatalf("out=%+v", out)
		}
		if out.MockRequest == "" {
			t.Fatal("missing mock request id")
		}
		got, err := store.Get(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.State != feboundary.StateOK || got.IsAGTAccepted() {
			t.Fatalf("%+v", got)
		}
	})

	t.Run("idempotency_returns_same_row", func(t *testing.T) {
		idem := fmt.Sprintf("idem-dup-%d", time.Now().UnixNano())
		in := fefixqueue.EnqueueInput{
			Operation: feboundary.OpConsultarFactura, IdentityRef: ref, IdempotencyKey: idem,
			Payload: fefixqueue.Payload{
				Consultar: &feprofile.ConsultarFacturaRequestClaims{
					TaxRegistrationNumber: "9100000000",
					DocumentNo:            "FT A/1",
				},
			},
		}
		a, err := store.Enqueue(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		b, err := store.Enqueue(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if a.ID != b.ID {
			t.Fatalf("ids differ %s vs %s", a.ID, b.ID)
		}
	})

	t.Run("transport_retry_then_ok", func(t *testing.T) {
		failEng, _, _, cleanupFail := setupEngineWithHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(503)
		}))
		defer cleanupFail()

		idem := fmt.Sprintf("idem-retry-%d", time.Now().UnixNano())
		row, err := store.Enqueue(ctx, fefixqueue.EnqueueInput{
			Operation: feboundary.OpObterEstado, IdentityRef: ref, IdempotencyKey: idem,
			Payload: fefixqueue.Payload{
				ObterEstado: &feprofile.ObterEstadoRequestClaims{
					TaxRegistrationNumber: "9100000000",
					RequestID:             "req-retry",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		out, err := store.ProcessNext(ctx, failEng)
		if err == nil || !out.Retried {
			t.Fatalf("want retry, out=%+v err=%v", out, err)
		}
		got, _ := store.Get(ctx, row.ID)
		if got.State != feboundary.StateQueued {
			t.Fatalf("state=%q want queued", got.State)
		}
		out2, err := store.ProcessNext(ctx, eng)
		if err != nil {
			t.Fatal(err)
		}
		if out2.State != feboundary.StateOK {
			t.Fatalf("%+v", out2)
		}
	})

	t.Run("empty_queue", func(t *testing.T) {
		_, err := store.ProcessNext(ctx, eng)
		if !errors.Is(err, fefixqueue.ErrEmpty) {
			t.Fatalf("%v", err)
		}
	})
}

func setupEngine(t *testing.T) (*feboundary.Engine, agttestkit.IdentityProvider, *femock.Server, func()) {
	return setupEngineWithHandler(t, nil)
}

func setupEngineWithHandler(t *testing.T, override http.Handler) (*feboundary.Engine, agttestkit.IdentityProvider, *femock.Server, func()) {
	t.Helper()
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		cleanupWB()
		t.Fatal(err)
	}
	mock, err := femock.New(femock.Config{Username: mockUser, Password: mockPass, Provider: provider})
	if err != nil {
		t.Fatal(err)
	}
	var ts *httptest.Server
	if override != nil {
		ts = httptest.NewServer(override)
	} else {
		ts = httptest.NewServer(mock.Handler())
	}
	eng, err := feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: provider,
		BaseURL: ts.URL, Username: mockUser, Password: mockPass, Client: ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		_ = eng.Close()
		ts.Close()
		_ = mock.Close()
		_ = provider.Close()
		cleanupWB()
	}
	return eng, provider, mock, cleanup
}
