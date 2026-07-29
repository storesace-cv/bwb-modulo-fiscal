package adminregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fepath"
)

// Authority profile lifecycle (DEC-BO-004) — metadata only; never secrets.
const (
	AuthorityStatusDraft     = "draft"
	AuthorityStatusValidated = "validated"
	AuthorityStatusActive    = "active"
	AuthorityStatusRevoked   = "revoked"
)

// KnownAuthorityOperations are FE operation keys cited in FE-SERVICES-MATRIX
// (source_id AO-FE-SNAP-HML-2026-07-25-*, pending_validation). Do not invent paths.
var KnownAuthorityOperations = map[string]struct{}{
	"registarFactura":  {},
	"solicitarSerie":   {},
	"listarSeries":     {},
	"obterEstado":      {},
	"listarFacturas":   {},
	"consultarFactura": {},
	"validarDocumento": {},
}

// AuthorityProfile is non-secret AGT preparation config (≠ AGT connection).
type AuthorityProfile struct {
	ID                    string
	Environment           string // homologation | production
	TaxpayerID            string // optional; empty = platform-level
	ScopeID               string // optional
	DisplayName           string
	Status                string
	AllowedOperations     []string
	PendingExternal       map[string]string // extensible unknowns; never secrets
	ProducerCredentialRef string            // SecretStore ref name only
	ProducerKeyRef        string
	CertificateRef        string
	AlgorithmDeclared     string // e.g. RS256 from archived FE docs — pending_validation
	KeyIDSanitized        string
	FingerprintSanitized  string
	ExpiresAt             *time.Time
	ConfigReady           bool
	SecretsReady          bool
	OfflineValidated      bool
	ExternalVerified      bool // always false until real AGT probe authorized
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// CreateAuthorityProfileInput validates fail-closed (no secret fields accepted).
type CreateAuthorityProfileInput struct {
	Environment           string
	TaxpayerID            string
	ScopeID               string
	DisplayName           string
	Status                string
	AllowedOperations     []string
	PendingExternal       map[string]string
	ProducerCredentialRef string
	ProducerKeyRef        string
	CertificateRef        string
	AlgorithmDeclared     string
	KeyIDSanitized        string
	FingerprintSanitized  string
	ExpiresAt             *time.Time
}

// UpdateAuthorityProfileInput patches metadata (owner). ExternalVerified cannot become true.
type UpdateAuthorityProfileInput struct {
	ProfileID             string
	DisplayName           string
	Status                string
	AllowedOperations     []string
	PendingExternal       map[string]string
	ProducerCredentialRef string
	ProducerKeyRef        string
	CertificateRef        string
	AlgorithmDeclared     string
	KeyIDSanitized        string
	FingerprintSanitized  string
	ExpiresAt             *time.Time
	ConfigReady           *bool
	SecretsReady          *bool
	OfflineValidated      *bool
}

// CreateAuthorityProfile inserts a draft/validated profile. Never stores secrets.
func (r *Registry) CreateAuthorityProfile(ctx context.Context, in CreateAuthorityProfileInput) (AuthorityProfile, error) {
	p, err := normalizeCreateAuthority(in)
	if err != nil {
		return AuthorityProfile{}, err
	}
	if in.TaxpayerID != "" {
		if _, err := r.GetTaxpayer(ctx, in.TaxpayerID); err != nil {
			return AuthorityProfile{}, err
		}
	}
	if in.ScopeID != "" {
		if _, err := r.GetScopeBinding(ctx, in.ScopeID); err != nil {
			return AuthorityProfile{}, err
		}
	}
	id, err := newID()
	if err != nil {
		return AuthorityProfile{}, err
	}
	now := r.now()
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	p.ExternalVerified = false
	if err := enforceActivationGate(p); err != nil {
		return AuthorityProfile{}, err
	}
	opsJSON, err := json.Marshal(p.AllowedOperations)
	if err != nil {
		return AuthorityProfile{}, fmt.Errorf("%w: allowed_operations", ErrValidation)
	}
	pendJSON, err := json.Marshal(p.PendingExternal)
	if err != nil {
		return AuthorityProfile{}, fmt.Errorf("%w: pending_external", ErrValidation)
	}
	q := `
INSERT INTO ` + r.t("authority_profiles") + ` (
  profile_id, environment, taxpayer_id, scope_id, display_name, status,
  allowed_operations, pending_external,
  producer_credential_ref, producer_key_ref, certificate_ref,
  algorithm_declared, key_id_sanitized, fingerprint_sanitized, expires_at,
  config_ready, secrets_ready, offline_validated, external_verified,
  created_at, updated_at
) VALUES (` + r.ph(21) + `)`
	var taxID, scopeID any
	if p.TaxpayerID != "" {
		taxID = p.TaxpayerID
	}
	if p.ScopeID != "" {
		scopeID = p.ScopeID
	}
	var exp any
	if p.ExpiresAt != nil {
		exp = r.timeArg(*p.ExpiresAt)
	}
	_, err = r.db.ExecContext(ctx, q,
		p.ID, p.Environment, taxID, scopeID, p.DisplayName, p.Status,
		string(opsJSON), string(pendJSON),
		p.ProducerCredentialRef, p.ProducerKeyRef, p.CertificateRef,
		p.AlgorithmDeclared, p.KeyIDSanitized, p.FingerprintSanitized, exp,
		r.encodeBool(p.ConfigReady), r.encodeBool(p.SecretsReady), r.encodeBool(p.OfflineValidated), r.encodeBool(false),
		r.timeArg(now), r.timeArg(now),
	)
	if err != nil {
		return AuthorityProfile{}, fmt.Errorf("adminregistry: insert authority_profile: %w", err)
	}
	return p, nil
}

// GetAuthorityProfile loads one profile by id.
func (r *Registry) GetAuthorityProfile(ctx context.Context, id string) (AuthorityProfile, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return AuthorityProfile{}, fmt.Errorf("%w: profile_id", ErrValidation)
	}
	q := `
SELECT profile_id, environment, taxpayer_id, scope_id, display_name, status,
  allowed_operations, pending_external,
  producer_credential_ref, producer_key_ref, certificate_ref,
  algorithm_declared, key_id_sanitized, fingerprint_sanitized, expires_at,
  config_ready, secrets_ready, offline_validated, external_verified,
  created_at, updated_at
FROM ` + r.t("authority_profiles") + ` WHERE profile_id = ` + r.p(1)
	return r.scanAuthorityProfile(r.db.QueryRowContext(ctx, q, id))
}

// ListAuthorityProfiles returns profiles ordered by environment, created_at.
func (r *Registry) ListAuthorityProfiles(ctx context.Context) ([]AuthorityProfile, error) {
	q := `
SELECT profile_id, environment, taxpayer_id, scope_id, display_name, status,
  allowed_operations, pending_external,
  producer_credential_ref, producer_key_ref, certificate_ref,
  algorithm_declared, key_id_sanitized, fingerprint_sanitized, expires_at,
  config_ready, secrets_ready, offline_validated, external_verified,
  created_at, updated_at
FROM ` + r.t("authority_profiles") + `
ORDER BY environment ASC, created_at ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuthorityProfile
	for rows.Next() {
		p, err := r.scanAuthorityProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateAuthorityProfile patches non-secret fields. external_verified stays false.
func (r *Registry) UpdateAuthorityProfile(ctx context.Context, in UpdateAuthorityProfileInput) (AuthorityProfile, error) {
	cur, err := r.GetAuthorityProfile(ctx, in.ProfileID)
	if err != nil {
		return AuthorityProfile{}, err
	}
	if dn := strings.TrimSpace(in.DisplayName); dn != "" {
		if utf8.RuneCountInString(dn) > 120 {
			return AuthorityProfile{}, fmt.Errorf("%w: display_name", ErrValidation)
		}
		cur.DisplayName = dn
	}
	if st := strings.TrimSpace(in.Status); st != "" {
		if !validAuthorityStatus(st) {
			return AuthorityProfile{}, fmt.Errorf("%w: status", ErrValidation)
		}
		cur.Status = st
	}
	if in.AllowedOperations != nil {
		ops, err := validateAllowedOperations(in.AllowedOperations)
		if err != nil {
			return AuthorityProfile{}, err
		}
		cur.AllowedOperations = ops
	}
	if in.PendingExternal != nil {
		pe, err := validatePendingExternal(in.PendingExternal)
		if err != nil {
			return AuthorityProfile{}, err
		}
		cur.PendingExternal = pe
	}
	if in.ProducerCredentialRef != "" || in.ProducerKeyRef != "" || in.CertificateRef != "" ||
		in.AlgorithmDeclared != "" || in.KeyIDSanitized != "" || in.FingerprintSanitized != "" {
		cur.ProducerCredentialRef = sanitizeRefName(in.ProducerCredentialRef, cur.ProducerCredentialRef)
		cur.ProducerKeyRef = sanitizeRefName(in.ProducerKeyRef, cur.ProducerKeyRef)
		cur.CertificateRef = sanitizeRefName(in.CertificateRef, cur.CertificateRef)
		if ad := strings.TrimSpace(in.AlgorithmDeclared); ad != "" {
			if err := validateAlgorithmDeclared(ad); err != nil {
				return AuthorityProfile{}, err
			}
			cur.AlgorithmDeclared = ad
		}
		if k := strings.TrimSpace(in.KeyIDSanitized); k != "" {
			if utf8.RuneCountInString(k) > 64 || looksLikeSecret(k) {
				return AuthorityProfile{}, fmt.Errorf("%w: key_id_sanitized", ErrValidation)
			}
			cur.KeyIDSanitized = k
		}
		if f := strings.TrimSpace(in.FingerprintSanitized); f != "" {
			if utf8.RuneCountInString(f) > 128 || looksLikeSecret(f) {
				return AuthorityProfile{}, fmt.Errorf("%w: fingerprint_sanitized", ErrValidation)
			}
			cur.FingerprintSanitized = f
		}
	}
	if in.ExpiresAt != nil {
		cur.ExpiresAt = in.ExpiresAt
	}
	if in.ConfigReady != nil {
		cur.ConfigReady = *in.ConfigReady
	}
	if in.SecretsReady != nil {
		cur.SecretsReady = *in.SecretsReady
	}
	if in.OfflineValidated != nil {
		cur.OfflineValidated = *in.OfflineValidated
	}
	cur.ExternalVerified = false
	if err := enforceActivationGate(cur); err != nil {
		return AuthorityProfile{}, err
	}
	cur.UpdatedAt = r.now()
	opsJSON, _ := json.Marshal(cur.AllowedOperations)
	pendJSON, _ := json.Marshal(cur.PendingExternal)
	var exp any
	if cur.ExpiresAt != nil {
		exp = r.timeArg(*cur.ExpiresAt)
	}
	q := `
UPDATE ` + r.t("authority_profiles") + ` SET
  display_name=` + r.p(1) + `, status=` + r.p(2) + `,
  allowed_operations=` + r.p(3) + `, pending_external=` + r.p(4) + `,
  producer_credential_ref=` + r.p(5) + `, producer_key_ref=` + r.p(6) + `, certificate_ref=` + r.p(7) + `,
  algorithm_declared=` + r.p(8) + `, key_id_sanitized=` + r.p(9) + `, fingerprint_sanitized=` + r.p(10) + `,
  expires_at=` + r.p(11) + `,
  config_ready=` + r.p(12) + `, secrets_ready=` + r.p(13) + `, offline_validated=` + r.p(14) + `,
  external_verified=` + r.p(15) + `, updated_at=` + r.p(16) + `
WHERE profile_id=` + r.p(17)
	res, err := r.db.ExecContext(ctx, q,
		cur.DisplayName, cur.Status, string(opsJSON), string(pendJSON),
		cur.ProducerCredentialRef, cur.ProducerKeyRef, cur.CertificateRef,
		cur.AlgorithmDeclared, cur.KeyIDSanitized, cur.FingerprintSanitized, exp,
		r.encodeBool(cur.ConfigReady), r.encodeBool(cur.SecretsReady), r.encodeBool(cur.OfflineValidated),
		r.encodeBool(false), r.timeArg(cur.UpdatedAt), cur.ID,
	)
	if err != nil {
		return AuthorityProfile{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return AuthorityProfile{}, ErrNotFound
	}
	return cur, nil
}

func normalizeCreateAuthority(in CreateAuthorityProfileInput) (AuthorityProfile, error) {
	env := strings.TrimSpace(in.Environment)
	if env != EnvHomologation && env != EnvProduction {
		return AuthorityProfile{}, fmt.Errorf("%w: environment (homologation|production)", ErrValidation)
	}
	dn := strings.TrimSpace(in.DisplayName)
	if dn == "" || utf8.RuneCountInString(dn) > 120 {
		return AuthorityProfile{}, fmt.Errorf("%w: display_name", ErrValidation)
	}
	st := strings.TrimSpace(in.Status)
	if st == "" {
		st = AuthorityStatusDraft
	}
	if !validAuthorityStatus(st) {
		return AuthorityProfile{}, fmt.Errorf("%w: status", ErrValidation)
	}
	ops, err := validateAllowedOperations(in.AllowedOperations)
	if err != nil {
		return AuthorityProfile{}, err
	}
	pe, err := validatePendingExternal(in.PendingExternal)
	if err != nil {
		return AuthorityProfile{}, err
	}
	ad := strings.TrimSpace(in.AlgorithmDeclared)
	if ad != "" {
		if err := validateAlgorithmDeclared(ad); err != nil {
			return AuthorityProfile{}, err
		}
	}
	kid := strings.TrimSpace(in.KeyIDSanitized)
	if kid != "" && (utf8.RuneCountInString(kid) > 64 || looksLikeSecret(kid)) {
		return AuthorityProfile{}, fmt.Errorf("%w: key_id_sanitized", ErrValidation)
	}
	fp := strings.TrimSpace(in.FingerprintSanitized)
	if fp != "" && (utf8.RuneCountInString(fp) > 128 || looksLikeSecret(fp)) {
		return AuthorityProfile{}, fmt.Errorf("%w: fingerprint_sanitized", ErrValidation)
	}
	return AuthorityProfile{
		Environment:           env,
		TaxpayerID:            strings.TrimSpace(in.TaxpayerID),
		ScopeID:               strings.TrimSpace(in.ScopeID),
		DisplayName:           dn,
		Status:                st,
		AllowedOperations:     ops,
		PendingExternal:       pe,
		ProducerCredentialRef: sanitizeRefName(in.ProducerCredentialRef, ""),
		ProducerKeyRef:        sanitizeRefName(in.ProducerKeyRef, ""),
		CertificateRef:        sanitizeRefName(in.CertificateRef, ""),
		AlgorithmDeclared:     ad,
		KeyIDSanitized:        kid,
		FingerprintSanitized:  fp,
		ExpiresAt:             in.ExpiresAt,
		ExternalVerified:      false,
	}, nil
}

func validAuthorityStatus(s string) bool {
	switch s {
	case AuthorityStatusDraft, AuthorityStatusValidated, AuthorityStatusActive, AuthorityStatusRevoked:
		return true
	default:
		return false
	}
}

// enforceActivationGate is fail-closed: active requires local readiness triad;
// external_verified is never a gate input and must remain false (≠ AGT).
func enforceActivationGate(p AuthorityProfile) error {
	if p.ExternalVerified {
		return fmt.Errorf("%w: external_verified não permitido (≠ AGT)", ErrValidation)
	}
	if p.Status != AuthorityStatusActive {
		return nil
	}
	if !p.ConfigReady || !p.SecretsReady || !p.OfflineValidated {
		return fmt.Errorf("%w: activação exige config_ready+secrets_ready+offline_validated (fail-closed)", ErrValidation)
	}
	return nil
}

func validateAllowedOperations(ops []string) ([]string, error) {
	if ops == nil {
		return []string{}, nil
	}
	if len(ops) > 32 {
		return nil, fmt.Errorf("%w: allowed_operations excessivo", ErrValidation)
	}
	seen := map[string]struct{}{}
	var out []string
	for _, o := range ops {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if _, ok := KnownAuthorityOperations[o]; !ok {
			return nil, fmt.Errorf("%w: operation %q não catalogada (use pending_external)", ErrValidation, o)
		}
		if fepath.ConflictOpen && fepath.ServiceHasPathConflict(o) {
			return nil, fmt.Errorf("%w: operation %q conflict_open (C-FE-001)", ErrValidation, o)
		}
		if _, ok := seen[o]; ok {
			continue
		}
		seen[o] = struct{}{}
		out = append(out, o)
	}
	return out, nil
}

func validatePendingExternal(m map[string]string) (map[string]string, error) {
	if m == nil {
		return map[string]string{}, nil
	}
	if len(m) > 32 {
		return nil, fmt.Errorf("%w: pending_external excessivo", ErrValidation)
	}
	out := map[string]string{}
	for k, v := range m {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || utf8.RuneCountInString(k) > 64 {
			return nil, fmt.Errorf("%w: pending_external key", ErrValidation)
		}
		if utf8.RuneCountInString(v) > 256 || looksLikeSecret(v) || looksLikeSecret(k) {
			return nil, fmt.Errorf("%w: pending_external value", ErrValidation)
		}
		out[k] = v
	}
	return out, nil
}

func validateAlgorithmDeclared(ad string) error {
	// Only algorithms observed in archived FE docs (pending_validation) — not inventing claims.
	switch ad {
	case "RS256", "pending_external":
		return nil
	default:
		return fmt.Errorf("%w: algorithm_declared (RS256|pending_external)", ErrValidation)
	}
}

func sanitizeRefName(in, fallback string) string {
	s := strings.TrimSpace(in)
	if s == "" {
		return fallback
	}
	if utf8.RuneCountInString(s) > 64 || looksLikeSecret(s) {
		return fallback
	}
	return s
}

func looksLikeSecret(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "begin ") || strings.Contains(low, "private") ||
		strings.Contains(low, "password") || strings.Contains(low, "-----") ||
		strings.Contains(low, "bearer ") || strings.Contains(low, "basic ") {
		return true
	}
	return false
}

type scannable interface {
	Scan(dest ...any) error
}

func (r *Registry) scanAuthorityProfile(row scannable) (AuthorityProfile, error) {
	var (
		p                      AuthorityProfile
		taxID, scopeID         sql.NullString
		opsJSON, pendJSON      string
		exp                    any
		cfg, sec, off, ext     any
		createdRaw, updatedRaw any
	)
	err := row.Scan(
		&p.ID, &p.Environment, &taxID, &scopeID, &p.DisplayName, &p.Status,
		&opsJSON, &pendJSON,
		&p.ProducerCredentialRef, &p.ProducerKeyRef, &p.CertificateRef,
		&p.AlgorithmDeclared, &p.KeyIDSanitized, &p.FingerprintSanitized, &exp,
		&cfg, &sec, &off, &ext,
		&createdRaw, &updatedRaw,
	)
	if err == sql.ErrNoRows {
		return AuthorityProfile{}, ErrNotFound
	}
	if err != nil {
		return AuthorityProfile{}, err
	}
	if taxID.Valid {
		p.TaxpayerID = taxID.String
	}
	if scopeID.Valid {
		p.ScopeID = scopeID.String
	}
	_ = json.Unmarshal([]byte(opsJSON), &p.AllowedOperations)
	if p.AllowedOperations == nil {
		p.AllowedOperations = []string{}
	}
	_ = json.Unmarshal([]byte(pendJSON), &p.PendingExternal)
	if p.PendingExternal == nil {
		p.PendingExternal = map[string]string{}
	}
	p.ConfigReady = decodeBool(cfg)
	p.SecretsReady = decodeBool(sec)
	p.OfflineValidated = decodeBool(off)
	p.ExternalVerified = false
	if exp != nil {
		switch x := exp.(type) {
		case time.Time:
			t := x.UTC()
			p.ExpiresAt = &t
		case string:
			if strings.TrimSpace(x) != "" {
				if t, err := parseTime(x); err == nil {
					p.ExpiresAt = &t
				}
			}
		case []byte:
			if len(x) > 0 {
				if t, err := parseTime(x); err == nil {
					p.ExpiresAt = &t
				}
			}
		}
	}
	if t, err := parseTime(createdRaw); err == nil {
		p.CreatedAt = t
	}
	if t, err := parseTime(updatedRaw); err == nil {
		p.UpdatedAt = t
	}
	return p, nil
}

func (r *Registry) encodeBool(b bool) any {
	if r.dialect == DialectSQLite {
		if b {
			return 1
		}
		return 0
	}
	return b
}

func decodeBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case int64:
		return t != 0
	case int:
		return t != 0
	case []byte:
		return string(t) == "1" || strings.EqualFold(string(t), "true")
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}
