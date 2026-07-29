package prep

import (
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

// MaterialRefRole is a logical role on AuthorityProfile (never a secret).
type MaterialRefRole string

const (
	RoleCredential MaterialRefRole = "producer_credential"
	RoleKey        MaterialRefRole = "producer_key"
	RoleCert       MaterialRefRole = "certificate"
)

// MaterialRefView is sanitized SecretStore metadata for one profile ref.
type MaterialRefView struct {
	Role        MaterialRefRole `json:"role"`
	RefName     string          `json:"ref_name"`
	Kind        string          `json:"kind"`
	Environment string          `json:"environment"`
	SubjectID   string          `json:"subject_id"`
	Status      string          `json:"status"`
	Version     int             `json:"version"`
	Fingerprint string          `json:"fingerprint,omitempty"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
}

// MaterialLifecycle is the owner-visible certificate/material lifecycle for a profile.
// Never includes plaintext. external_verified is always false.
type MaterialLifecycle struct {
	ProfileID        string            `json:"profile_id"`
	Environment      string            `json:"environment"`
	Status           string            `json:"status"`
	OfflineValidated bool              `json:"offline_validated"`
	SecretsReady     bool              `json:"secrets_ready"`
	ExternalVerified bool              `json:"external_verified"`
	Refs             []MaterialRefView `json:"refs"`
	Note             string            `json:"note"`
}

// MetadataLookup fetches sanitized metadata for a SecretStore ref.
type MetadataLookup func(ref secretstore.Ref) (secretstore.Metadata, error)

// SubjectForProfile chooses SecretStore subject_id (taxpayer or platform).
func SubjectForProfile(p adminregistry.AuthorityProfile) string {
	if s := strings.TrimSpace(p.TaxpayerID); s != "" {
		return s
	}
	return "platform"
}

// RefForRole builds a SecretStore Ref from profile logical name + role.
func RefForRole(p adminregistry.AuthorityProfile, role MaterialRefRole) (secretstore.Ref, bool) {
	var name, kind string
	switch role {
	case RoleCredential:
		name, kind = p.ProducerCredentialRef, secretstore.KindProducerCredential
	case RoleKey:
		name, kind = p.ProducerKeyRef, secretstore.KindProducerKey
	case RoleCert:
		name, kind = p.CertificateRef, secretstore.KindCertificate
	default:
		return secretstore.Ref{}, false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return secretstore.Ref{}, false
	}
	return secretstore.Ref{
		Kind: kind, Environment: p.Environment,
		SubjectID: SubjectForProfile(p), Name: name,
	}, true
}

// BuildMaterialLifecycle returns sanitized lifecycle snapshot (≠ AGT).
func BuildMaterialLifecycle(p adminregistry.AuthorityProfile, lookup MetadataLookup) (MaterialLifecycle, error) {
	if lookup == nil {
		return MaterialLifecycle{}, fmt.Errorf("prep: metadata lookup obrigatório")
	}
	out := MaterialLifecycle{
		ProfileID:        p.ID,
		Environment:      p.Environment,
		Status:           p.Status,
		OfflineValidated: p.OfflineValidated,
		SecretsReady:     p.SecretsReady,
		ExternalVerified: false,
		Note:             "Após rotate/revoke, offline_validated é invalidado até nova validação offline. ≠ AGT / external_verified.",
		Refs:             make([]MaterialRefView, 0, 3),
	}
	for _, role := range []MaterialRefRole{RoleCredential, RoleKey, RoleCert} {
		ref, ok := RefForRole(p, role)
		if !ok {
			out.Refs = append(out.Refs, MaterialRefView{
				Role: role, RefName: "", Status: secretstore.StatusAbsent,
				Kind: string(role), Environment: p.Environment, SubjectID: SubjectForProfile(p),
			})
			continue
		}
		meta, err := lookup(ref)
		if err != nil {
			return MaterialLifecycle{}, err
		}
		v := MaterialRefView{
			Role: role, RefName: ref.Name, Kind: ref.Kind,
			Environment: ref.Environment, SubjectID: ref.SubjectID,
			Status: meta.Status, Version: meta.Version, Fingerprint: meta.Fingerprint,
			ExpiresAt: meta.ExpiresAt,
		}
		if v.Status == "" {
			v.Status = secretstore.StatusAbsent
		}
		out.Refs = append(out.Refs, v)
	}
	return out, nil
}

// ProfilePatchAfterMaterialChange invalidates offline validation after put/rotate/revoke.
// When kind is certificate and meta is present, updates sanitized fingerprint/expiry.
// Never sets external_verified. Never stores plaintext.
func ProfilePatchAfterMaterialChange(
	p adminregistry.AuthorityProfile,
	kind string,
	refName string,
	meta secretstore.Metadata,
	revoked bool,
) adminregistry.UpdateAuthorityProfileInput {
	falseVal := false
	in := adminregistry.UpdateAuthorityProfileInput{
		ProfileID:        p.ID,
		OfflineValidated: &falseVal,
	}
	refName = strings.TrimSpace(refName)
	kind = strings.TrimSpace(kind)

	if revoked {
		sec := false
		in.SecretsReady = &sec
		return in
	}

	// Fail-closed: material change never auto-claims secrets_ready (RM-AGTPREP-018).
	// Owner must re-assert after ValidateProfileBindings confirms credential+key+cert present.
	sec := false
	in.SecretsReady = &sec
	if kind == secretstore.KindCertificate && refName != "" && (p.CertificateRef == "" || p.CertificateRef == refName) {
		if meta.Fingerprint != "" {
			fp := meta.Fingerprint
			if !strings.HasPrefix(fp, "sha256:") {
				fp = "sha256:" + fp
			}
			in.FingerprintSanitized = fp
		}
		if meta.ExpiresAt != nil {
			exp := meta.ExpiresAt.UTC()
			in.ExpiresAt = &exp
		}
		if p.CertificateRef == "" {
			in.CertificateRef = refName
		}
	}
	return in
}
