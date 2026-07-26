// Package adminregistry is the DEC-BO-001 plano A foundation: taxpayers, establishments, scope bindings.
//
// No secrets, AGT credentials, private keys, tokens, or private URLs are stored or returned.
package adminregistry

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Dialect selects SQL placeholders / timestamp encoding.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"

	EnvHomologation = "homologation"
	EnvProduction   = "production"
	EnvDevelopment  = "development"
)

var (
	// ErrValidation is returned for fail-closed input errors.
	ErrValidation = errors.New("adminregistry: validação")
	// ErrConflict is returned on unique constraint / business conflicts.
	ErrConflict = errors.New("adminregistry: conflito")
	// ErrNotFound is returned when a referenced entity is missing.
	ErrNotFound = errors.New("adminregistry: não encontrado")
)

// Registry persists non-secret admin cadastros.
type Registry struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

// New returns a Registry. now may be nil (uses time.Now UTC).
func New(db *sql.DB, dialect Dialect, now func() time.Time) *Registry {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Registry{db: db, dialect: dialect, now: now}
}

// Taxpayer is a non-secret contribuinte record.
type Taxpayer struct {
	ID        string
	NIF       string
	LegalName string
	Status    string
	CreatedAt time.Time
}

// Establishment belongs to a taxpayer (no secrets).
type Establishment struct {
	ID         string
	TaxpayerID string
	Code       string
	Name       string
	Status     string
	CreatedAt  time.Time
}

// ScopeBinding links POS scope metadata to taxpayer + establishment (no secrets).
type ScopeBinding struct {
	ScopeID             string
	TaxpayerID          string
	EstablishmentID     string
	Environment         string
	IANATimezone        string
	SeriesEffectiveCode string
	Status              string
	CreatedAt           time.Time
}

// CreateTaxpayerInput is validated fail-closed.
type CreateTaxpayerInput struct {
	NIF       string
	LegalName string
	Status    string // default active
}

// CreateEstablishmentInput requires an existing taxpayer.
type CreateEstablishmentInput struct {
	TaxpayerID string
	Code       string
	Name       string
	Status     string
}

// CreateScopeBindingInput requires matching taxpayer↔establishment ownership.
type CreateScopeBindingInput struct {
	ScopeID             string
	TaxpayerID          string
	EstablishmentID     string
	Environment         string
	IANATimezone        string
	SeriesEffectiveCode string
	Status              string
}

// UpdateScopeConfigInput updates non-secret series/timezone/environment/status (RM-BO-002).
// TaxpayerID and EstablishmentID are immutable after create.
type UpdateScopeConfigInput struct {
	ScopeID             string
	Environment         string
	IANATimezone        string
	SeriesEffectiveCode string
	Status              string
}

// CreateTaxpayer inserts a taxpayer. Never accepts secret fields.
func (r *Registry) CreateTaxpayer(ctx context.Context, in CreateTaxpayerInput) (Taxpayer, error) {
	nif := strings.TrimSpace(in.NIF)
	name := strings.TrimSpace(in.LegalName)
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = StatusActive
	}
	if nif == "" || name == "" {
		return Taxpayer{}, fmt.Errorf("%w: nif/legal_name obrigatórios", ErrValidation)
	}
	if status != StatusActive && status != StatusInactive {
		return Taxpayer{}, fmt.Errorf("%w: status inválido", ErrValidation)
	}
	id, err := newID()
	if err != nil {
		return Taxpayer{}, err
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + r.t("taxpayers") + ` (taxpayer_id, nif, legal_name, status, created_at) VALUES (` + r.ph(5) + `)`
	_, err = r.db.ExecContext(ctx, q, id, nif, name, status, r.timeArg(now))
	if err != nil {
		if isUniqueViolation(err) {
			return Taxpayer{}, fmt.Errorf("%w: nif duplicado", ErrConflict)
		}
		return Taxpayer{}, fmt.Errorf("adminregistry: insert taxpayer: %w", err)
	}
	return Taxpayer{ID: id, NIF: nif, LegalName: name, Status: status, CreatedAt: now}, nil
}

// CreateEstablishment inserts an establishment under a taxpayer.
func (r *Registry) CreateEstablishment(ctx context.Context, in CreateEstablishmentInput) (Establishment, error) {
	tp := strings.TrimSpace(in.TaxpayerID)
	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = StatusActive
	}
	if tp == "" || code == "" || name == "" {
		return Establishment{}, fmt.Errorf("%w: taxpayer_id/code/name obrigatórios", ErrValidation)
	}
	if status != StatusActive && status != StatusInactive {
		return Establishment{}, fmt.Errorf("%w: status inválido", ErrValidation)
	}
	if err := r.requireTaxpayer(ctx, tp); err != nil {
		return Establishment{}, err
	}
	id, err := newID()
	if err != nil {
		return Establishment{}, err
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + r.t("establishments") + ` (
		establishment_id, taxpayer_id, code, name, status, created_at
	) VALUES (` + r.ph(6) + `)`
	_, err = r.db.ExecContext(ctx, q, id, tp, code, name, status, r.timeArg(now))
	if err != nil {
		if isUniqueViolation(err) {
			return Establishment{}, fmt.Errorf("%w: código duplicado no contribuinte", ErrConflict)
		}
		return Establishment{}, fmt.Errorf("adminregistry: insert establishment: %w", err)
	}
	return Establishment{ID: id, TaxpayerID: tp, Code: code, Name: name, Status: status, CreatedAt: now}, nil
}

// CreateScopeBinding inserts a non-secret scope binding; establishment must belong to taxpayer.
func (r *Registry) CreateScopeBinding(ctx context.Context, in CreateScopeBindingInput) (ScopeBinding, error) {
	scopeID := strings.TrimSpace(in.ScopeID)
	tp := strings.TrimSpace(in.TaxpayerID)
	est := strings.TrimSpace(in.EstablishmentID)
	env := strings.TrimSpace(in.Environment)
	tz := strings.TrimSpace(in.IANATimezone)
	series := strings.TrimSpace(in.SeriesEffectiveCode)
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = StatusActive
	}
	if scopeID == "" || tp == "" || est == "" || env == "" || tz == "" || series == "" {
		return ScopeBinding{}, fmt.Errorf("%w: campos obrigatórios em falta", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return ScopeBinding{}, err
	}
	if err := validateStatus(status); err != nil {
		return ScopeBinding{}, err
	}
	if err := validateIANATimezone(tz); err != nil {
		return ScopeBinding{}, err
	}
	owner, err := r.establishmentTaxpayer(ctx, est)
	if err != nil {
		return ScopeBinding{}, err
	}
	if owner != tp {
		return ScopeBinding{}, fmt.Errorf("%w: estabelecimento não pertence ao contribuinte", ErrValidation)
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + r.t("scope_bindings") + ` (
		scope_id, taxpayer_id, establishment_id, environment, iana_timezone, series_effective_code, status, created_at
	) VALUES (` + r.ph(8) + `)`
	_, err = r.db.ExecContext(ctx, q, scopeID, tp, est, env, tz, series, status, r.timeArg(now))
	if err != nil {
		if isUniqueViolation(err) {
			return ScopeBinding{}, fmt.Errorf("%w: scope_id duplicado", ErrConflict)
		}
		return ScopeBinding{}, fmt.Errorf("adminregistry: insert scope_binding: %w", err)
	}
	return ScopeBinding{
		ScopeID: scopeID, TaxpayerID: tp, EstablishmentID: est,
		Environment: env, IANATimezone: tz, SeriesEffectiveCode: series,
		Status: status, CreatedAt: now,
	}, nil
}

// UpdateScopeConfig patches non-secret configuration on an existing binding (RM-BO-002).
// Never accepts or returns secrets. Environment is metadata only (HML≠PRD isolation is logical).
func (r *Registry) UpdateScopeConfig(ctx context.Context, in UpdateScopeConfigInput) (ScopeBinding, error) {
	scopeID := strings.TrimSpace(in.ScopeID)
	env := strings.TrimSpace(in.Environment)
	tz := strings.TrimSpace(in.IANATimezone)
	series := strings.TrimSpace(in.SeriesEffectiveCode)
	status := strings.TrimSpace(in.Status)
	if scopeID == "" || env == "" || tz == "" || series == "" || status == "" {
		return ScopeBinding{}, fmt.Errorf("%w: campos obrigatórios em falta", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return ScopeBinding{}, err
	}
	if err := validateStatus(status); err != nil {
		return ScopeBinding{}, err
	}
	if err := validateIANATimezone(tz); err != nil {
		return ScopeBinding{}, err
	}
	cur, err := r.GetScopeBinding(ctx, scopeID)
	if err != nil {
		return ScopeBinding{}, err
	}
	q := `UPDATE ` + r.t("scope_bindings") + ` SET environment = ` + r.p(1) +
		`, iana_timezone = ` + r.p(2) +
		`, series_effective_code = ` + r.p(3) +
		`, status = ` + r.p(4) +
		` WHERE scope_id = ` + r.p(5)
	res, err := r.db.ExecContext(ctx, q, env, tz, series, status, scopeID)
	if err != nil {
		return ScopeBinding{}, fmt.Errorf("adminregistry: update scope_binding: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ScopeBinding{}, ErrNotFound
	}
	return ScopeBinding{
		ScopeID: cur.ScopeID, TaxpayerID: cur.TaxpayerID, EstablishmentID: cur.EstablishmentID,
		Environment: env, IANATimezone: tz, SeriesEffectiveCode: series,
		Status: status, CreatedAt: cur.CreatedAt,
	}, nil
}

// GetTaxpayer returns a taxpayer by id.
func (r *Registry) GetTaxpayer(ctx context.Context, id string) (Taxpayer, error) {
	id = strings.TrimSpace(id)
	var out Taxpayer
	var created any
	q := `SELECT taxpayer_id, nif, legal_name, status, created_at FROM ` + r.t("taxpayers") + ` WHERE taxpayer_id = ` + r.ph(1)
	err := r.db.QueryRowContext(ctx, q, id).Scan(&out.ID, &out.NIF, &out.LegalName, &out.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Taxpayer{}, ErrNotFound
	}
	if err != nil {
		return Taxpayer{}, err
	}
	out.CreatedAt, err = parseTime(created)
	return out, err
}

// GetEstablishment returns an establishment by id.
func (r *Registry) GetEstablishment(ctx context.Context, id string) (Establishment, error) {
	id = strings.TrimSpace(id)
	var out Establishment
	var created any
	q := `SELECT establishment_id, taxpayer_id, code, name, status, created_at FROM ` + r.t("establishments") + ` WHERE establishment_id = ` + r.ph(1)
	err := r.db.QueryRowContext(ctx, q, id).Scan(&out.ID, &out.TaxpayerID, &out.Code, &out.Name, &out.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Establishment{}, ErrNotFound
	}
	if err != nil {
		return Establishment{}, err
	}
	out.CreatedAt, err = parseTime(created)
	return out, err
}

// GetScopeBinding returns a scope binding by scope_id.
func (r *Registry) GetScopeBinding(ctx context.Context, scopeID string) (ScopeBinding, error) {
	scopeID = strings.TrimSpace(scopeID)
	var out ScopeBinding
	var created any
	q := `SELECT scope_id, taxpayer_id, establishment_id, environment, iana_timezone, series_effective_code, status, created_at FROM ` + r.t("scope_bindings") + ` WHERE scope_id = ` + r.ph(1)
	err := r.db.QueryRowContext(ctx, q, scopeID).Scan(
		&out.ScopeID, &out.TaxpayerID, &out.EstablishmentID, &out.Environment,
		&out.IANATimezone, &out.SeriesEffectiveCode, &out.Status, &created,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ScopeBinding{}, ErrNotFound
	}
	if err != nil {
		return ScopeBinding{}, err
	}
	out.CreatedAt, err = parseTime(created)
	return out, err
}

func (r *Registry) requireTaxpayer(ctx context.Context, id string) error {
	_, err := r.GetTaxpayer(ctx, id)
	return err
}

func (r *Registry) establishmentTaxpayer(ctx context.Context, establishmentID string) (string, error) {
	var tp string
	q := `SELECT taxpayer_id FROM ` + r.t("establishments") + ` WHERE establishment_id = ` + r.ph(1)
	err := r.db.QueryRowContext(ctx, q, establishmentID).Scan(&tp)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return tp, err
}

func (r *Registry) t(name string) string {
	if r.dialect == DialectPostgres {
		return "fiscal." + name
	}
	return name
}

func (r *Registry) ph(n int) string {
	if r.dialect == DialectPostgres {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf("$%d", i+1)
		}
		return strings.Join(parts, ", ")
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

// p returns a single placeholder ($n or ?).
func (r *Registry) p(n int) string {
	if r.dialect == DialectPostgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (r *Registry) timeArg(t time.Time) any {
	if r.dialect == DialectPostgres {
		return t
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

func parseTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x.UTC(), nil
	case string:
		return time.Parse(time.RFC3339Nano, x)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(x))
	default:
		return time.Time{}, fmt.Errorf("adminregistry: created_at tipo %T", v)
	}
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "constraint")
}

func validateEnvironment(env string) error {
	switch env {
	case EnvHomologation, EnvProduction, EnvDevelopment:
		return nil
	default:
		return fmt.Errorf("%w: environment inválido", ErrValidation)
	}
}

func validateStatus(status string) error {
	if status != StatusActive && status != StatusInactive {
		return fmt.Errorf("%w: status inválido", ErrValidation)
	}
	return nil
}

func validateIANATimezone(tz string) error {
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("%w: iana_timezone inválido", ErrValidation)
	}
	return nil
}
