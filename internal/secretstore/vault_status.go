package secretstore

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// VaultStatus is owner-visible SecretStore readiness. Never includes key material or env raw values.
type VaultStatus struct {
	FiscalEnv            string   `json:"fiscal_env"`
	BackendDeclared      string   `json:"backend_declared"` // unset|auto|memory|sql|invalid
	MasterKeyConfigured  bool     `json:"master_key_configured"`
	MasterKeyParseOK     bool     `json:"master_key_parse_ok"`
	MasterKeyFingerprint string   `json:"master_key_fingerprint"` // sha256 hex of key bytes; empty if unset/invalid
	StorageModeRuntime   string   `json:"storage_mode_runtime"`   // from open vault; may be empty
	DurableRequired      bool     `json:"durable_required"`
	ReadyForHomologation bool     `json:"ready_for_homologation"` // local prep; ≠ AGT
	CipherAlgorithm      string   `json:"cipher_algorithm"`
	Notes                []string `json:"notes"`
}

// BuildVaultStatusFromEnv inspects process env without logging secrets.
// runtimeMode is the open Vault.StorageMode() when available (else empty).
func BuildVaultStatusFromEnv(fiscalEnv, runtimeMode string) VaultStatus {
	return BuildVaultStatus(fiscalEnv, os.Getenv(envSecretBackend), os.Getenv(envSecretMasterKey), runtimeMode)
}

// BuildVaultStatus is the pure builder (tests inject env values).
func BuildVaultStatus(fiscalEnv, backendRaw, masterKeyRaw, runtimeMode string) VaultStatus {
	env := strings.ToLower(strings.TrimSpace(fiscalEnv))
	backend := strings.ToLower(strings.TrimSpace(backendRaw))
	keyRaw := strings.TrimSpace(masterKeyRaw)
	out := VaultStatus{
		FiscalEnv:          env,
		CipherAlgorithm:    CipherAlgAES256GCM,
		Notes:              []string{"metadados só; ≠ plaintext; ≠ AGT/KMS", "HML≠PRD; master key fora do Git"},
		StorageModeRuntime: strings.TrimSpace(runtimeMode),
	}

	switch backend {
	case "":
		out.BackendDeclared = "auto"
	case "memory":
		out.BackendDeclared = "memory"
	case "sql", "durable", "encrypted":
		out.BackendDeclared = "sql"
	default:
		out.BackendDeclared = "invalid"
		out.Notes = append(out.Notes, "FISCAL_SECRETSTORE_BACKEND inválido (sql|memory)")
	}

	out.DurableRequired = env == "homologation" || env == "production"
	if keyRaw != "" {
		out.MasterKeyConfigured = true
		key, err := ParseMasterKey(keyRaw)
		if err == nil {
			out.MasterKeyParseOK = true
			sum := sha256.Sum256(key)
			out.MasterKeyFingerprint = hex.EncodeToString(sum[:])
			zeroBytes(key)
		} else {
			out.Notes = append(out.Notes, "master key presente mas encoding/tamanho inválido")
		}
	} else {
		out.Notes = append(out.Notes, "FISCAL_SECRETSTORE_MASTER_KEY ausente")
	}

	if out.BackendDeclared == "memory" && out.DurableRequired {
		out.Notes = append(out.Notes, "fail-closed: memory proibido fora de development")
	}
	if out.DurableRequired && !out.MasterKeyParseOK {
		out.Notes = append(out.Notes, "fail-closed: homologation/production exigem master key válida")
	}

	durableRuntime := out.StorageModeRuntime == StorageModeDurableEncrypted
	out.ReadyForHomologation = out.BackendDeclared != "invalid" &&
		out.BackendDeclared != "memory" &&
		out.MasterKeyParseOK &&
		(out.StorageModeRuntime == "" || durableRuntime) &&
		out.DurableRequired
	// Also allow ready when env is development but durable is configured (prep).
	if !out.DurableRequired && out.MasterKeyParseOK && out.BackendDeclared != "invalid" && out.BackendDeclared != "memory" {
		out.ReadyForHomologation = false // not an HML/PRD env; local prep only
		out.Notes = append(out.Notes, "ambiente não exige durable; ready_for_homologation=false")
	}
	if out.ReadyForHomologation {
		out.Notes = append(out.Notes, "ready_for_homologation=true (prep local; ≠ homologação AGT)")
	}
	return out
}
