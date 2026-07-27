package adminregistry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
)

// Series lifecycle (RM-BO-014). Never reuse codes; never move backwards.
const (
	SeriesDraft  = "draft"
	SeriesActive = "active"
	SeriesClosed = "closed"
)

// EstablishmentSeries is non-secret series metadata for an establishment+environment+type.
// This is NOT the fiscal numbering authority (DEC-PROD-008); it configures which series codes exist.
type EstablishmentSeries struct {
	ID              string
	EstablishmentID string
	Environment     string
	CodigoCanonico  string
	Code            string
	Status          string
	ValidFrom       time.Time
	ValidTo         *time.Time
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateSeriesInput creates a draft series (fail-closed).
type CreateSeriesInput struct {
	EstablishmentID string
	Environment     string
	CodigoCanonico  string
	Code            string
	ValidFrom       time.Time // zero ⇒ now
}

// TransitionSeriesInput advances lifecycle with optimistic concurrency.
type TransitionSeriesInput struct {
	SeriesID        string
	ExpectedVersion int
	ToStatus        string // active | closed
}

// CreateSeries inserts a draft series. Code is unique per establishment+environment forever.
func (r *Registry) CreateSeries(ctx context.Context, in CreateSeriesInput) (EstablishmentSeries, error) {
	estID := strings.TrimSpace(in.EstablishmentID)
	env := strings.TrimSpace(in.Environment)
	canon := strings.TrimSpace(in.CodigoCanonico)
	code := strings.TrimSpace(in.Code)
	if estID == "" || env == "" || canon == "" || code == "" {
		return EstablishmentSeries{}, fmt.Errorf("%w: campos obrigatórios", ErrValidation)
	}
	if err := validateEnvironment(env); err != nil {
		return EstablishmentSeries{}, err
	}
	reg, err := doctype.Default()
	if err != nil {
		return EstablishmentSeries{}, err
	}
	if _, ok := reg.Lookup(canon); !ok {
		return EstablishmentSeries{}, fmt.Errorf("%w: codigo_canonico fora do catálogo", ErrValidation)
	}
	if _, err := r.GetEstablishment(ctx, estID); err != nil {
		return EstablishmentSeries{}, err
	}
	id, err := newID()
	if err != nil {
		return EstablishmentSeries{}, err
	}
	now := r.now().UTC().Truncate(time.Microsecond)
	validFrom := in.ValidFrom.UTC().Truncate(time.Microsecond)
	if validFrom.IsZero() {
		validFrom = now
	}
	q := `INSERT INTO ` + r.t("establishment_series") + ` (
		series_id, establishment_id, environment, codigo_canonico, code, status,
		valid_from, valid_to, version, created_at, updated_at
	) VALUES (` + r.ph(11) + `)`
	_, err = r.db.ExecContext(ctx, q,
		id, estID, env, canon, code, SeriesDraft,
		r.timeArg(validFrom), nil, 1, r.timeArg(now), r.timeArg(now),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return EstablishmentSeries{}, fmt.Errorf("%w: código de série já utilizado neste ambiente", ErrConflict)
		}
		return EstablishmentSeries{}, fmt.Errorf("adminregistry: insert series: %w", err)
	}
	return EstablishmentSeries{
		ID: id, EstablishmentID: estID, Environment: env, CodigoCanonico: canon,
		Code: code, Status: SeriesDraft, ValidFrom: validFrom, Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// TransitionSeries advances draft→active or active→closed under a transaction + version check.
func (r *Registry) TransitionSeries(ctx context.Context, in TransitionSeriesInput) (EstablishmentSeries, error) {
	id := strings.TrimSpace(in.SeriesID)
	to := strings.TrimSpace(in.ToStatus)
	if id == "" || in.ExpectedVersion < 1 {
		return EstablishmentSeries{}, fmt.Errorf("%w: series_id/expected_version obrigatórios", ErrValidation)
	}
	if to != SeriesActive && to != SeriesClosed {
		return EstablishmentSeries{}, fmt.Errorf("%w: transição inválida", ErrValidation)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return EstablishmentSeries{}, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := r.getSeriesTx(ctx, tx, id)
	if err != nil {
		return EstablishmentSeries{}, err
	}
	if cur.Version != in.ExpectedVersion {
		return EstablishmentSeries{}, fmt.Errorf("%w: versão desactualizada", ErrConflict)
	}
	if err := validateSeriesTransition(cur.Status, to); err != nil {
		return EstablishmentSeries{}, err
	}

	now := r.now().UTC().Truncate(time.Microsecond)
	newVersion := cur.Version + 1
	var validTo any
	if to == SeriesClosed {
		validTo = r.timeArg(now)
		t := now
		cur.ValidTo = &t
	}

	q := `UPDATE ` + r.t("establishment_series") + ` SET
		status = ` + r.p(1) + `,
		valid_to = ` + r.p(2) + `,
		version = ` + r.p(3) + `,
		updated_at = ` + r.p(4) + `
	WHERE series_id = ` + r.p(5) + ` AND version = ` + r.p(6)
	res, err := tx.ExecContext(ctx, q, to, validTo, newVersion, r.timeArg(now), id, in.ExpectedVersion)
	if err != nil {
		if isUniqueViolation(err) {
			return EstablishmentSeries{}, fmt.Errorf("%w: já existe série active para este tipo", ErrConflict)
		}
		return EstablishmentSeries{}, fmt.Errorf("adminregistry: transition series: %w", err)
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return EstablishmentSeries{}, fmt.Errorf("%w: versão desactualizada", ErrConflict)
	}
	if err := tx.Commit(); err != nil {
		return EstablishmentSeries{}, err
	}
	cur.Status = to
	cur.Version = newVersion
	cur.UpdatedAt = now
	return cur, nil
}

func validateSeriesTransition(from, to string) error {
	switch {
	case from == SeriesDraft && to == SeriesActive:
		return nil
	case from == SeriesActive && to == SeriesClosed:
		return nil
	case from == SeriesClosed:
		return fmt.Errorf("%w: série closed é terminal (sem reutilizar/retroceder)", ErrValidation)
	case from == SeriesActive && to == SeriesDraft:
		return fmt.Errorf("%w: proibido retroceder active→draft", ErrValidation)
	case from == SeriesDraft && to == SeriesClosed:
		// allow closing unused draft without activating
		return nil
	default:
		return fmt.Errorf("%w: transição %s→%s inválida", ErrValidation, from, to)
	}
}

// ListSeries returns series for an establishment (optional environment filter).
func (r *Registry) ListSeries(ctx context.Context, establishmentID, environment string, limit int) ([]EstablishmentSeries, error) {
	estID := strings.TrimSpace(establishmentID)
	env := strings.TrimSpace(environment)
	if estID == "" {
		return nil, fmt.Errorf("%w: establishment_id obrigatório", ErrValidation)
	}
	limit = clampList(limit)
	var (
		rows *sql.Rows
		err  error
	)
	if env != "" {
		if err := validateEnvironment(env); err != nil {
			return nil, err
		}
		q := `SELECT series_id, establishment_id, environment, codigo_canonico, code, status,
			valid_from, valid_to, version, created_at, updated_at
			FROM ` + r.t("establishment_series") + `
			WHERE establishment_id = ` + r.p(1) + ` AND environment = ` + r.p(2) + `
			ORDER BY created_at DESC LIMIT ` + r.p(3)
		rows, err = r.db.QueryContext(ctx, q, estID, env, limit)
	} else {
		q := `SELECT series_id, establishment_id, environment, codigo_canonico, code, status,
			valid_from, valid_to, version, created_at, updated_at
			FROM ` + r.t("establishment_series") + `
			WHERE establishment_id = ` + r.p(1) + `
			ORDER BY created_at DESC LIMIT ` + r.p(2)
		rows, err = r.db.QueryContext(ctx, q, estID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EstablishmentSeries, 0)
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSeries returns one series by id.
func (r *Registry) GetSeries(ctx context.Context, seriesID string) (EstablishmentSeries, error) {
	return r.getSeriesTx(ctx, r.db, strings.TrimSpace(seriesID))
}

func (r *Registry) getSeriesTx(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, seriesID string) (EstablishmentSeries, error) {
	if seriesID == "" {
		return EstablishmentSeries{}, fmt.Errorf("%w: series_id obrigatório", ErrValidation)
	}
	row := q.QueryRowContext(ctx, `SELECT series_id, establishment_id, environment, codigo_canonico, code, status,
		valid_from, valid_to, version, created_at, updated_at
		FROM `+r.t("establishment_series")+` WHERE series_id = `+r.ph(1), seriesID)
	var s EstablishmentSeries
	var validFrom, validTo, created, updated any
	err := row.Scan(
		&s.ID, &s.EstablishmentID, &s.Environment, &s.CodigoCanonico, &s.Code, &s.Status,
		&validFrom, &validTo, &s.Version, &created, &updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return EstablishmentSeries{}, ErrNotFound
	}
	if err != nil {
		return EstablishmentSeries{}, err
	}
	s.ValidFrom, err = parseTime(validFrom)
	if err != nil {
		return EstablishmentSeries{}, err
	}
	if validTo != nil {
		if t, err := parseTime(validTo); err == nil && !t.IsZero() {
			s.ValidTo = &t
		}
	}
	s.CreatedAt, err = parseTime(created)
	if err != nil {
		return EstablishmentSeries{}, err
	}
	s.UpdatedAt, err = parseTime(updated)
	return s, err
}

func scanSeries(rows *sql.Rows) (EstablishmentSeries, error) {
	var s EstablishmentSeries
	var validFrom, validTo, created, updated any
	if err := rows.Scan(
		&s.ID, &s.EstablishmentID, &s.Environment, &s.CodigoCanonico, &s.Code, &s.Status,
		&validFrom, &validTo, &s.Version, &created, &updated,
	); err != nil {
		return EstablishmentSeries{}, err
	}
	var err error
	s.ValidFrom, err = parseTime(validFrom)
	if err != nil {
		return EstablishmentSeries{}, err
	}
	if validTo != nil {
		if t, err := parseTime(validTo); err == nil && !t.IsZero() {
			s.ValidTo = &t
		}
	}
	s.CreatedAt, err = parseTime(created)
	if err != nil {
		return EstablishmentSeries{}, err
	}
	s.UpdatedAt, err = parseTime(updated)
	return s, err
}
