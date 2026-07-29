package prep

import (
	"context"
	"fmt"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fepath"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

// Binding issue codes (sanitized; never secrets).
const (
	BindingOK                = "ok"
	BindingConflictOpen      = "conflict_open"
	BindingRefAbsent         = "ref_absent"
	BindingRefWrongEnv       = "ref_wrong_env"
	BindingRefRevoked        = "ref_revoked"
	BindingSecretsReadyUnmet = "secrets_ready_unmet"
	BindingOpsConflict       = "ops_conflict_open"
)

// BindingIssue is one fail-closed finding for owner prep (≠ AGT).
type BindingIssue struct {
	Code   string `json:"code"`
	Field  string `json:"field"`
	Detail string `json:"detail"`
}

// BindingValidation is the owner-visible binding check for an AuthorityProfile.
type BindingValidation struct {
	ProfileID        string            `json:"profile_id"`
	Environment      string            `json:"environment"`
	Status           string            `json:"status"`
	SecretsReady     bool              `json:"secrets_ready"`
	ExternalVerified bool              `json:"external_verified"`
	Valid            bool              `json:"valid"`
	Issues           []BindingIssue    `json:"issues"`
	OpsPathStatuses  map[string]string `json:"ops_path_statuses"`
	Note             string            `json:"note"`
}

// ValidateProfileBindings checks FE op path conflicts + SecretStore ref presence when secrets_ready.
// Never reads plaintext. external_verified always reported false.
func ValidateProfileBindings(ctx context.Context, p adminregistry.AuthorityProfile, lookup MetadataLookup) (BindingValidation, error) {
	if err := ctx.Err(); err != nil {
		return BindingValidation{}, err
	}
	if lookup == nil {
		return BindingValidation{}, fmt.Errorf("prep: metadata lookup obrigatório")
	}
	out := BindingValidation{
		ProfileID:        p.ID,
		Environment:      p.Environment,
		Status:           p.Status,
		SecretsReady:     p.SecretsReady,
		ExternalVerified: false,
		Valid:            true,
		Issues:           make([]BindingIssue, 0),
		OpsPathStatuses:  map[string]string{},
		Note:             "Validação local de preparação; ≠ AGT / external_verified",
	}

	for _, op := range p.AllowedOperations {
		op = strings.TrimSpace(op)
		if op == "" {
			continue
		}
		switch {
		case fepath.ServiceHasPathConflict(op):
			out.OpsPathStatuses[op] = PathStatusConflictOpen
			out.Valid = false
			out.Issues = append(out.Issues, BindingIssue{
				Code: BindingOpsConflict, Field: "allowed_operations",
				Detail: "C-FE-001 aberto — operação conflituosa não pode ser bindável: " + op,
			})
		case fepath.ServiceIsAligned(op):
			out.OpsPathStatuses[op] = PathStatusAligned
		default:
			out.OpsPathStatuses[op] = PathStatusPendingExternal
		}
	}

	roles := []MaterialRefRole{RoleCredential, RoleKey, RoleCert}
	present := 0
	for _, role := range roles {
		ref, ok := RefForRole(p, role)
		field := string(role) + "_ref"
		if !ok {
			if p.SecretsReady {
				out.Valid = false
				out.Issues = append(out.Issues, BindingIssue{
					Code: BindingRefAbsent, Field: field,
					Detail: "secrets_ready exige ref lógica não vazia",
				})
			}
			continue
		}
		meta, err := lookup(ref)
		if err != nil {
			return BindingValidation{}, err
		}
		if meta.Environment != "" && meta.Environment != p.Environment && meta.Status == secretstore.StatusPresent {
			out.Valid = false
			out.Issues = append(out.Issues, BindingIssue{
				Code: BindingRefWrongEnv, Field: field,
				Detail: "ref presente noutro ambiente (HML≠PRD)",
			})
			continue
		}
		switch meta.Status {
		case secretstore.StatusPresent:
			present++
		case secretstore.StatusRevoked:
			out.Valid = false
			out.Issues = append(out.Issues, BindingIssue{
				Code: BindingRefRevoked, Field: field,
				Detail: "ref revogada",
			})
		default:
			if p.SecretsReady {
				out.Valid = false
				out.Issues = append(out.Issues, BindingIssue{
					Code: BindingRefAbsent, Field: field,
					Detail: "ref ausente ou sem material presente no SecretStore",
				})
			}
		}
	}
	if p.SecretsReady && present < 3 {
		out.Valid = false
		// Ensure a summary issue if individual ones already added.
		hasUnmet := false
		for _, i := range out.Issues {
			if i.Code == BindingRefAbsent || i.Code == BindingRefRevoked || i.Code == BindingSecretsReadyUnmet {
				hasUnmet = true
				break
			}
		}
		if !hasUnmet {
			out.Issues = append(out.Issues, BindingIssue{
				Code: BindingSecretsReadyUnmet, Field: "secrets_ready",
				Detail: "secrets_ready exige credential+key+certificate presentes",
			})
		}
	}
	return out, nil
}

// AssertBindingsForWrite fails closed when secrets_ready or active and bindings invalid.
func AssertBindingsForWrite(ctx context.Context, p adminregistry.AuthorityProfile, lookup MetadataLookup) error {
	if lookup == nil {
		return fmt.Errorf("prep: metadata lookup obrigatório")
	}
	if !p.SecretsReady && p.Status != adminregistry.AuthorityStatusActive {
		// Still refuse conflict ops even in draft.
		for _, op := range p.AllowedOperations {
			if fepath.ConflictOpen && fepath.ServiceHasPathConflict(op) {
				return fmt.Errorf("%w: %s", adminregistry.ErrValidation, BindingOpsConflict+": "+op)
			}
		}
		return nil
	}
	v, err := ValidateProfileBindings(ctx, p, lookup)
	if err != nil {
		return err
	}
	if !v.Valid {
		code := BindingSecretsReadyUnmet
		if len(v.Issues) > 0 {
			code = v.Issues[0].Code
		}
		return fmt.Errorf("%w: binding %s", adminregistry.ErrValidation, code)
	}
	return nil
}
