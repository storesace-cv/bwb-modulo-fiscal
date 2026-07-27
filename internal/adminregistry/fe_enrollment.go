package adminregistry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// FE enrollment states (DEC-PROD-004). Canonical model is this enum — never a boolean.
const (
	FEEnrollmentNotEnrolled = "not_enrolled"
	FEEnrollmentPending     = "pending"
	FEEnrollmentActive      = "active"
	FEEnrollmentSuspended   = "suspended"
)

// FEEnrollment is adesão FE do contribuinte num ambiente (plano A; sem segredos).
// "aderiu_facturacao_electronica" no sentido operacional ≡ Status == FEEnrollmentActive.
type FEEnrollment struct {
	TaxpayerID  string
	Environment string
	Status      string
	UpdatedAt   time.Time
}

// UpsertFEEnrollmentInput validates fail-closed.
type UpsertFEEnrollmentInput struct {
	TaxpayerID  string
	Environment string
	Status      string
}

// FEAderiu reports whether FE channel may be considered enrolled for emission gates.
// Equivalent to the product notion "aderiu facturação electrónica" for that environment.
func FEAderiu(status string) bool {
	return strings.TrimSpace(status) == FEEnrollmentActive
}

// EffectiveFEStatus returns persisted status or not_enrolled when absent (DEC-PROD-004).
func EffectiveFEStatus(enrollments []FEEnrollment, environment string) string {
	environment = strings.TrimSpace(environment)
	for _, e := range enrollments {
		if e.Environment == environment {
			return e.Status
		}
	}
	return FEEnrollmentNotEnrolled
}

func validateFEEnrollmentStatus(status string) error {
	switch status {
	case FEEnrollmentNotEnrolled, FEEnrollmentPending, FEEnrollmentActive, FEEnrollmentSuspended:
		return nil
	default:
		return fmt.Errorf("%w: fe_enrollment_status inválido", ErrValidation)
	}
}

// UpsertFEEnrollment creates or updates FE enrollment for taxpayer+environment.
// Never stores NIF; audit callers must use taxpayer_id only.
func (r *Registry) UpsertFEEnrollment(ctx context.Context, in UpsertFEEnrollmentInput) (FEEnrollment, error) {
	tp := strings.TrimSpace(in.TaxpayerID)
	env := strings.TrimSpace(in.Environment)
	status := strings.TrimSpace(in.Status)
	if tp == "" || env == "" || status == "" {
		return FEEnrollment{}, fmt.Errorf("%w: taxpayer_id/environment/status obrigatórios", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return FEEnrollment{}, err
	}
	if err := validateFEEnrollmentStatus(status); err != nil {
		return FEEnrollment{}, err
	}
	if err := r.requireTaxpayer(ctx, tp); err != nil {
		return FEEnrollment{}, err
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + r.t("taxpayer_fe_enrollments") + ` (
		taxpayer_id, environment, status, updated_at
	) VALUES (` + r.ph(4) + `)
	ON CONFLICT (taxpayer_id, environment) DO UPDATE SET
		status = excluded.status,
		updated_at = excluded.updated_at`
	if _, err := r.db.ExecContext(ctx, q, tp, env, status, r.timeArg(now)); err != nil {
		return FEEnrollment{}, fmt.Errorf("adminregistry: upsert fe enrollment: %w", err)
	}
	return FEEnrollment{TaxpayerID: tp, Environment: env, Status: status, UpdatedAt: now}, nil
}

// ListFEEnrollments returns all FE enrollment rows for a taxpayer (may be empty).
func (r *Registry) ListFEEnrollments(ctx context.Context, taxpayerID string) ([]FEEnrollment, error) {
	taxpayerID = strings.TrimSpace(taxpayerID)
	if taxpayerID == "" {
		return nil, fmt.Errorf("%w: taxpayer_id obrigatório", ErrValidation)
	}
	q := `SELECT taxpayer_id, environment, status, updated_at FROM ` + r.t("taxpayer_fe_enrollments") +
		` WHERE taxpayer_id = ` + r.ph(1) + ` ORDER BY environment ASC`
	rows, err := r.db.QueryContext(ctx, q, taxpayerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]FEEnrollment, 0)
	for rows.Next() {
		var e FEEnrollment
		var updated any
		if err := rows.Scan(&e.TaxpayerID, &e.Environment, &e.Status, &updated); err != nil {
			return nil, err
		}
		e.UpdatedAt, err = parseTime(updated)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetFEEnrollment returns one enrollment or ErrNotFound (caller may treat as not_enrolled).
func (r *Registry) GetFEEnrollment(ctx context.Context, taxpayerID, environment string) (FEEnrollment, error) {
	taxpayerID = strings.TrimSpace(taxpayerID)
	environment = strings.TrimSpace(environment)
	var out FEEnrollment
	var updated any
	q := `SELECT taxpayer_id, environment, status, updated_at FROM ` + r.t("taxpayer_fe_enrollments") +
		` WHERE taxpayer_id = ` + r.ph(1) + ` AND environment = ` + r.ph(2)
	err := r.db.QueryRowContext(ctx, q, taxpayerID, environment).Scan(
		&out.TaxpayerID, &out.Environment, &out.Status, &updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return FEEnrollment{}, ErrNotFound
	}
	if err != nil {
		return FEEnrollment{}, err
	}
	out.UpdatedAt, err = parseTime(updated)
	return out, err
}
