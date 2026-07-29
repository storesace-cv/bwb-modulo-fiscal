package secretstore

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

const (
	// StorageModeEphemeralMemory is process-local AES-GCM (dev/tests only).
	StorageModeEphemeralMemory = "ephemeral_memory"
	// StorageModeDurableEncrypted is SQL ciphertext + external master key.
	StorageModeDurableEncrypted = "durable_encrypted"

	envSecretBackend   = "FISCAL_SECRETSTORE_BACKEND"
	envSecretMasterKey = "FISCAL_SECRETSTORE_MASTER_KEY"
)

// Vault is the combined SecAdm + runtime surface with a sanitized storage mode label.
type Vault interface {
	AdminView
	RuntimeReveal
	StorageMode() string
}

// Dialect selects SQL placeholders / table qualification.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// OpenFromEnv selects durable SQL (preferred) or ephemeral memory under fail-closed rules.
// Never logs master key material. Homologation/production require durable_encrypted.
func OpenFromEnv(db *sql.DB, dialect Dialect, fiscalEnv string) (Vault, error) {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv(envSecretBackend)))
	keyRaw := strings.TrimSpace(os.Getenv(envSecretMasterKey))
	env := strings.ToLower(strings.TrimSpace(fiscalEnv))

	wantMemory := backend == "memory"
	wantSQL := backend == "sql" || backend == "durable" || backend == "encrypted"
	if backend != "" && !wantMemory && !wantSQL {
		return nil, fmt.Errorf("%w: %s deve ser sql|memory", ErrValidation, envSecretBackend)
	}

	switch {
	case wantMemory:
		if env != "" && env != "development" {
			return nil, fmt.Errorf("%w: backend memory só em development", ErrValidation)
		}
		return NewMemorySimulator(nil)
	case wantSQL || keyRaw != "":
		if db == nil {
			return nil, fmt.Errorf("%w: db obrigatório para durable_encrypted", ErrValidation)
		}
		key, err := ParseMasterKey(keyRaw)
		if err != nil {
			return nil, err
		}
		defer zeroBytes(key)
		return NewSQL(db, dialect, key, nil)
	default:
		// Auto: development may use ephemeral memory; other envs fail-closed.
		if env == "development" || env == "" {
			return NewMemorySimulator(nil)
		}
		return nil, fmt.Errorf("%w: %s obrigatória em %s (durable encrypted-at-rest)", ErrValidation, envSecretMasterKey, env)
	}
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Ensure Memory implements Vault.
var _ Vault = (*Memory)(nil)
