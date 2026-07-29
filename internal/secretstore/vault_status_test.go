package secretstore_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestBuildVaultStatusSanitized(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	b64 := base64.StdEncoding.EncodeToString(key)

	st := secretstore.BuildVaultStatus("homologation", "sql", b64, secretstore.StorageModeDurableEncrypted)
	if !st.MasterKeyConfigured || !st.MasterKeyParseOK || st.MasterKeyFingerprint == "" {
		t.Fatalf("want parse ok+fingerprint: %+v", st)
	}
	if strings.Contains(st.MasterKeyFingerprint, b64) || len(st.MasterKeyFingerprint) != 64 {
		t.Fatalf("fingerprint leak/shape: %q", st.MasterKeyFingerprint)
	}
	if !st.ReadyForHomologation || !st.DurableRequired {
		t.Fatalf("want ready durable: %+v", st)
	}
	for _, n := range st.Notes {
		if strings.Contains(n, b64) || strings.Contains(strings.ToLower(n), "begin") {
			t.Fatalf("note leak: %q", n)
		}
	}

	bad := secretstore.BuildVaultStatus("production", "memory", "", "")
	if bad.ReadyForHomologation || bad.MasterKeyConfigured {
		t.Fatalf("fail-closed memory/prod: %+v", bad)
	}
	if bad.BackendDeclared != "memory" {
		t.Fatalf("backend=%s", bad.BackendDeclared)
	}
}
