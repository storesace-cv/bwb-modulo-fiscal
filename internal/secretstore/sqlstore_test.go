package secretstore_test

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestParseMasterKeyBase64AndHex(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	b64 := base64.StdEncoding.EncodeToString(raw)
	got, err := secretstore.ParseMasterKey(b64)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 || got[0] != 1 {
		t.Fatalf("bad key")
	}
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got2, err := secretstore.ParseMasterKey(hexKey)
	if err != nil || len(got2) != 32 {
		t.Fatalf("%v %d", err, len(got2))
	}
	_, err = secretstore.ParseMasterKey("short")
	if !errors.Is(err, secretstore.ErrValidation) {
		t.Fatalf("got %v", err)
	}
	if err != nil && (err.Error() == b64 || err.Error() == hexKey) {
		t.Fatal("master key leaked into error")
	}
}

func TestSQLStoreWriteOnlyDurableRoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "secrets.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(0xA0 + i)
	}
	store, err := secretstore.NewSQL(sqlDB, secretstore.DialectSQLite, key, func() time.Time {
		return time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.StorageMode() != secretstore.StorageModeDurableEncrypted {
		t.Fatalf("mode=%s", store.StorageMode())
	}

	ref := secretstore.Ref{
		Kind: secretstore.KindProducerCredential, Environment: secretstore.EnvHomologation,
		SubjectID: "platform", Name: "agt-basic",
	}
	secret := []byte("not-a-real-agt-secret")
	res, err := store.Put(ctx, ref, secret, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Metadata.Status != secretstore.StatusPresent || res.Metadata.Fingerprint == "" {
		t.Fatalf("%+v", res.Metadata)
	}

	// Ciphertext must not equal plaintext in the table.
	var ct []byte
	if err := sqlDB.QueryRowContext(ctx, `SELECT ciphertext FROM secret_store_entries WHERE ref_key=?`, ref.Key()).Scan(&ct); err != nil {
		t.Fatal(err)
	}
	if string(ct) == string(secret) || len(ct) == 0 {
		t.Fatal("plaintext stored or empty ciphertext")
	}

	got, err := store.Reveal(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(secret) {
		t.Fatalf("reveal mismatch")
	}
	meta, err := store.Metadata(ctx, ref)
	if err != nil || meta.LastVerifiedAt == nil {
		t.Fatalf("%+v %v", meta, err)
	}

	if _, err := store.Revoke(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reveal(ctx, ref); !errors.Is(err, secretstore.ErrRevoked) {
		t.Fatalf("got %v", err)
	}
	var nonce, ct2 []byte
	var fp, status string
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT status, fingerprint, nonce, ciphertext FROM secret_store_entries WHERE ref_key=?`, ref.Key(),
	).Scan(&status, &fp, &nonce, &ct2); err != nil {
		t.Fatal(err)
	}
	if status != secretstore.StatusRevoked || fp != "" || len(nonce) != 0 || len(ct2) != 0 {
		t.Fatalf("revoke wipe incomplete: %s %q %d %d", status, fp, len(nonce), len(ct2))
	}
}

func TestOpenFromEnvFailClosedHomologation(t *testing.T) {
	t.Setenv("FISCAL_SECRETSTORE_BACKEND", "")
	t.Setenv("FISCAL_SECRETSTORE_MASTER_KEY", "")
	_, err := secretstore.OpenFromEnv(nil, secretstore.DialectSQLite, "homologation")
	if !errors.Is(err, secretstore.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestOpenFromEnvMemoryOnlyDevelopment(t *testing.T) {
	t.Setenv("FISCAL_SECRETSTORE_BACKEND", "memory")
	t.Setenv("FISCAL_SECRETSTORE_MASTER_KEY", "")
	v, err := secretstore.OpenFromEnv(nil, secretstore.DialectSQLite, "development")
	if err != nil {
		t.Fatal(err)
	}
	if v.StorageMode() != secretstore.StorageModeEphemeralMemory {
		t.Fatalf("%s", v.StorageMode())
	}
	_, err = secretstore.OpenFromEnv(nil, secretstore.DialectSQLite, "production")
	if !errors.Is(err, secretstore.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestOpenFromEnvDurableWithKey(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "secrets2.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	t.Setenv("FISCAL_SECRETSTORE_BACKEND", "sql")
	t.Setenv("FISCAL_SECRETSTORE_MASTER_KEY", base64.StdEncoding.EncodeToString(raw))
	v, err := secretstore.OpenFromEnv(sqlDB, secretstore.DialectSQLite, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	if v.StorageMode() != secretstore.StorageModeDurableEncrypted {
		t.Fatalf("%s", v.StorageMode())
	}
}
