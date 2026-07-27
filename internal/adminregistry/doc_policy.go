package adminregistry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
)

// DocPolicyConfig is establishment+environment activation for document groups/types (DEC-PROD-003).
type DocPolicyConfig struct {
	EstablishmentID string
	Environment     string
	GroupActive     map[string]bool
	TypeActive      map[string]bool
}

// UpsertDocGroupInput updates one of the five groups.
type UpsertDocGroupInput struct {
	EstablishmentID string
	Environment     string
	Grupo           string
	Active          bool
}

// UpsertDocTypeInput overrides seed activo for a known canonical code.
type UpsertDocTypeInput struct {
	EstablishmentID string
	Environment     string
	CodigoCanonico  string
	Active          bool
}

// LoadDocPolicyConfig returns stored overrides (missing keys ⇒ defaults in doctype policy).
func (r *Registry) LoadDocPolicyConfig(ctx context.Context, establishmentID, environment string) (DocPolicyConfig, error) {
	estID := strings.TrimSpace(establishmentID)
	env := strings.TrimSpace(environment)
	if estID == "" || env == "" {
		return DocPolicyConfig{}, fmt.Errorf("%w: establishment_id/environment obrigatórios", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return DocPolicyConfig{}, err
	}
	if _, err := r.GetEstablishment(ctx, estID); err != nil {
		return DocPolicyConfig{}, err
	}
	out := DocPolicyConfig{
		EstablishmentID: estID,
		Environment:     env,
		GroupActive:     map[string]bool{},
		TypeActive:      map[string]bool{},
	}
	gq := `SELECT grupo, active FROM ` + r.t("establishment_doc_group_config") +
		` WHERE establishment_id = ` + r.p(1) + ` AND environment = ` + r.p(2)
	rows, err := r.db.QueryContext(ctx, gq, estID, env)
	if err != nil {
		return DocPolicyConfig{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var grupo string
		var active any
		if err := rows.Scan(&grupo, &active); err != nil {
			return DocPolicyConfig{}, err
		}
		out.GroupActive[grupo] = decodeBool(active)
	}
	if err := rows.Err(); err != nil {
		return DocPolicyConfig{}, err
	}
	tq := `SELECT codigo_canonico, active FROM ` + r.t("establishment_doc_type_config") +
		` WHERE establishment_id = ` + r.p(1) + ` AND environment = ` + r.p(2)
	trows, err := r.db.QueryContext(ctx, tq, estID, env)
	if err != nil {
		return DocPolicyConfig{}, err
	}
	defer trows.Close()
	for trows.Next() {
		var canon string
		var active any
		if err := trows.Scan(&canon, &active); err != nil {
			return DocPolicyConfig{}, err
		}
		out.TypeActive[canon] = decodeBool(active)
	}
	return out, trows.Err()
}

// UpsertDocGroupConfig persists group activation (five groups only).
func (r *Registry) UpsertDocGroupConfig(ctx context.Context, in UpsertDocGroupInput) error {
	estID := strings.TrimSpace(in.EstablishmentID)
	env := strings.TrimSpace(in.Environment)
	grupo := strings.TrimSpace(in.Grupo)
	if estID == "" || env == "" || grupo == "" {
		return fmt.Errorf("%w: campos obrigatórios", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return err
	}
	if !doctype.ValidGroup(grupo) {
		return fmt.Errorf("%w: grupo inválido", ErrValidation)
	}
	if _, err := r.GetEstablishment(ctx, estID); err != nil {
		return err
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + r.t("establishment_doc_group_config") + ` (
		establishment_id, environment, grupo, active, updated_at
	) VALUES (` + r.ph(5) + `)
	ON CONFLICT (establishment_id, environment, grupo) DO UPDATE SET
		active = excluded.active,
		updated_at = excluded.updated_at`
	_, err := r.db.ExecContext(ctx, q, estID, env, grupo, r.encodeBool(in.Active), r.timeArg(now))
	if err != nil {
		return fmt.Errorf("adminregistry: upsert doc group: %w", err)
	}
	return nil
}

// UpsertDocTypeConfig persists type activation override for a catalog canonical.
func (r *Registry) UpsertDocTypeConfig(ctx context.Context, in UpsertDocTypeInput) error {
	estID := strings.TrimSpace(in.EstablishmentID)
	env := strings.TrimSpace(in.Environment)
	canon := strings.TrimSpace(in.CodigoCanonico)
	if estID == "" || env == "" || canon == "" {
		return fmt.Errorf("%w: campos obrigatórios", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return err
	}
	reg, err := doctype.Default()
	if err != nil {
		return err
	}
	if _, ok := reg.Lookup(canon); !ok {
		return fmt.Errorf("%w: codigo_canonico fora do catálogo", ErrValidation)
	}
	if _, err := r.GetEstablishment(ctx, estID); err != nil {
		return err
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + r.t("establishment_doc_type_config") + ` (
		establishment_id, environment, codigo_canonico, active, updated_at
	) VALUES (` + r.ph(5) + `)
	ON CONFLICT (establishment_id, environment, codigo_canonico) DO UPDATE SET
		active = excluded.active,
		updated_at = excluded.updated_at`
	_, err = r.db.ExecContext(ctx, q, estID, env, canon, r.encodeBool(in.Active), r.timeArg(now))
	if err != nil {
		return fmt.Errorf("adminregistry: upsert doc type: %w", err)
	}
	return nil
}

// AvailabilityConfig converts stored overrides to doctype.AvailabilityConfig.
func (c DocPolicyConfig) AvailabilityConfig() doctype.AvailabilityConfig {
	return doctype.AvailabilityConfig{
		GroupActive:        c.GroupActive,
		TypeActiveOverride: c.TypeActive,
	}
}
