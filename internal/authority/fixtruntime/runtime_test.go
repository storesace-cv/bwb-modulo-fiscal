package fixtruntime_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fefixqueue"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fixtruntime"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestRuntime_enqueue_and_process_obterEstado(t *testing.T) {
	ctx := context.Background()
	wbPath, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()

	dbPath := filepath.Join(t.TempDir(), "rt.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, dbPath); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	rt, err := fixtruntime.Open(wbPath, sqlDB, persistence.DialectSQLite, fixtruntime.Config{
		MockUser: "u", MockPassword: "p", WorkerInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	ref := rt.Provider.List()[0].Ref
	row, err := rt.Queue.Enqueue(ctx, fefixqueue.EnqueueInput{
		Operation: feboundary.OpObterEstado, IdentityRef: ref, IdempotencyKey: "idem-rt-1",
		Payload: fefixqueue.Payload{
			ObterEstado: &feprofile.ObterEstadoRequestClaims{
				TaxRegistrationNumber: "9100000000", RequestID: "req-rt-1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := rt.ProcessOne(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if out.State != feboundary.StateOK {
		t.Fatalf("state=%q", out.State)
	}
	got, err := rt.Queue.Get(ctx, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != feboundary.StateOK || got.IsAGTAccepted() {
		t.Fatalf("%+v", got)
	}
}
