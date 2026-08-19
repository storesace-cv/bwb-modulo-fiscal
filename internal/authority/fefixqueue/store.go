package fefixqueue

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

const (
	defaultMaxAttempts        = 5
	defaultUnavailableBackoff = 5 * time.Second
	defaultInFlightReclaim    = 30 * time.Second
)

// Dialect selects SQL table qualification.
type Dialect int

const (
	DialectSQLite Dialect = iota
	DialectPostgres
)

// Store persists fixture submissions.
type Store struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

// NewStore creates a fixture queue store.
func NewStore(db *sql.DB, dialect Dialect) *Store {
	return &Store{db: db, dialect: dialect, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) table() string {
	if s.dialect == DialectPostgres {
		return "fiscal.fe_fixture_submissions"
	}
	return "fe_fixture_submissions"
}

func (s *Store) timeArg(t time.Time) any {
	if s.dialect == DialectPostgres {
		return t
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s *Store) rebind(q string) string {
	if s.dialect == DialectPostgres {
		n := 1
		var b strings.Builder
		for _, r := range q {
			if r == '?' {
				b.WriteString(fmt.Sprintf("$%d", n))
				n++
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	return q
}

// EnqueueInput describes a new fixture submission.
type EnqueueInput struct {
	Operation      string
	IdentityRef    string
	IdempotencyKey string
	Payload        Payload
}

// Enqueue inserts or returns an existing row for the idempotency key.
func (s *Store) Enqueue(ctx context.Context, in EnqueueInput) (Row, error) {
	op := strings.TrimSpace(in.Operation)
	ref := strings.TrimSpace(in.IdentityRef)
	idem := strings.TrimSpace(in.IdempotencyKey)
	if op == "" || ref == "" || idem == "" {
		return Row{}, ErrInvalidInput
	}
	if err := in.Payload.Validate(op); err != nil {
		return Row{}, err
	}
	payloadJSON, err := in.Payload.marshal()
	if err != nil {
		return Row{}, err
	}

	existing, err := s.findByIdempotency(ctx, idem, op)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Row{}, err
	}

	now := s.now()
	id := newID()
	q := `INSERT INTO ` + s.table() + ` (
		id, operation, state, identity_ref, idempotency_key, payload_json,
		attempts, mock_request_id, mock_code, source_id, note,
		next_attempt_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 0, '', '', '', ?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, s.rebind(q),
		id, op, feboundary.StateQueued, ref, idem, payloadJSON,
		"queued persistent fixture; ≠ AGT",
		s.timeArg(now), s.timeArg(now), s.timeArg(now),
	)
	if err != nil {
		if s.isUniqueViolation(err) {
			return s.findByIdempotency(ctx, idem, op)
		}
		return Row{}, fmt.Errorf("fefixqueue: insert: %w", err)
	}
	return Row{
		ID: id, Operation: op, State: feboundary.StateQueued,
		IdentityRef: ref, IdempotencyKey: idem, PayloadJSON: payloadJSON,
		Note: "queued persistent fixture; ≠ AGT",
	}, nil
}

func (s *Store) findByIdempotency(ctx context.Context, idem, op string) (Row, error) {
	q := `SELECT id, operation, state, identity_ref, idempotency_key, payload_json,
		attempts, mock_request_id, mock_code, source_id, note
		FROM ` + s.table() + ` WHERE idempotency_key = ? AND operation = ?`
	return s.scanOne(ctx, q, idem, op)
}

// Get returns a row by primary key.
func (s *Store) Get(ctx context.Context, id string) (Row, error) {
	q := `SELECT id, operation, state, identity_ref, idempotency_key, payload_json,
		attempts, mock_request_id, mock_code, source_id, note
		FROM ` + s.table() + ` WHERE id = ?`
	return s.scanOne(ctx, q, id)
}

// ListRecent returns newest rows up to limit (default 50, max 200).
func (s *Store) ListRecent(ctx context.Context, limit int) ([]Row, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := `SELECT id, operation, state, identity_ref, idempotency_key, payload_json,
		attempts, mock_request_id, mock_code, source_id, note
		FROM ` + s.table() + ` ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.QueryContext(ctx, s.rebind(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Row, 0, limit)
	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.ID, &r.Operation, &r.State, &r.IdentityRef, &r.IdempotencyKey, &r.PayloadJSON,
			&r.Attempts, &r.MockRequestID, &r.MockCode, &r.SourceID, &r.Note,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) scanOne(ctx context.Context, q string, args ...any) (Row, error) {
	var r Row
	err := s.db.QueryRowContext(ctx, s.rebind(q), args...).Scan(
		&r.ID, &r.Operation, &r.State, &r.IdentityRef, &r.IdempotencyKey, &r.PayloadJSON,
		&r.Attempts, &r.MockRequestID, &r.MockCode, &r.SourceID, &r.Note,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Row{}, ErrNotFound
	}
	if err != nil {
		return Row{}, err
	}
	return r, nil
}

type claimRow struct {
	row Row
}

func (s *Store) claimNext(ctx context.Context) (claimRow, error) {
	now := s.now()
	reclaimBefore := now.Add(-defaultInFlightReclaim)

	if s.dialect == DialectPostgres {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return claimRow{}, err
		}
		defer func() { _ = tx.Rollback() }()

		reclaimQ := `UPDATE ` + s.table() + ` SET state = ?, updated_at = ?, note = ?
			WHERE state = ? AND updated_at < ?`
		if _, err := tx.ExecContext(ctx, s.rebind(reclaimQ),
			feboundary.StateQueued, s.timeArg(now), "reclaimed stale in_flight",
			feboundary.StateInFlight, s.timeArg(reclaimBefore),
		); err != nil {
			return claimRow{}, err
		}

		selQ := `SELECT id, operation, state, identity_ref, idempotency_key, payload_json,
			attempts, mock_request_id, mock_code, source_id, note
			FROM ` + s.table() + `
			WHERE state = ? AND next_attempt_at <= ?
			ORDER BY next_attempt_at ASC, created_at ASC
			LIMIT 1 FOR UPDATE SKIP LOCKED`
		var r Row
		err = tx.QueryRowContext(ctx, s.rebind(selQ), feboundary.StateQueued, s.timeArg(now)).Scan(
			&r.ID, &r.Operation, &r.State, &r.IdentityRef, &r.IdempotencyKey, &r.PayloadJSON,
			&r.Attempts, &r.MockRequestID, &r.MockCode, &r.SourceID, &r.Note,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return claimRow{}, ErrEmpty
		}
		if err != nil {
			return claimRow{}, err
		}
		upQ := `UPDATE ` + s.table() + ` SET state = ?, attempts = attempts + 1, updated_at = ?, note = ?
			WHERE id = ? AND state = ?`
		res, err := tx.ExecContext(ctx, s.rebind(upQ),
			feboundary.StateInFlight, s.timeArg(now), "claimed for fixture boundary",
			r.ID, feboundary.StateQueued,
		)
		if err != nil {
			return claimRow{}, err
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return claimRow{}, ErrEmpty
		}
		r.State = feboundary.StateInFlight
		r.Attempts++
		if err := tx.Commit(); err != nil {
			return claimRow{}, err
		}
		return claimRow{row: r}, nil
	}

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return claimRow{}, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return claimRow{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	reclaimQ := `UPDATE ` + s.table() + ` SET state = ?, updated_at = ?, note = ?
		WHERE state = ? AND updated_at < ?`
	if _, err := conn.ExecContext(ctx, reclaimQ,
		feboundary.StateQueued, s.timeArg(now), "reclaimed stale in_flight",
		feboundary.StateInFlight, s.timeArg(reclaimBefore),
	); err != nil {
		return claimRow{}, err
	}

	selQ := `SELECT id, operation, state, identity_ref, idempotency_key, payload_json,
		attempts, mock_request_id, mock_code, source_id, note
		FROM ` + s.table() + `
		WHERE state = ? AND next_attempt_at <= ?
		ORDER BY next_attempt_at ASC, created_at ASC
		LIMIT 1`
	var r Row
	err = conn.QueryRowContext(ctx, selQ, feboundary.StateQueued, s.timeArg(now)).Scan(
		&r.ID, &r.Operation, &r.State, &r.IdentityRef, &r.IdempotencyKey, &r.PayloadJSON,
		&r.Attempts, &r.MockRequestID, &r.MockCode, &r.SourceID, &r.Note,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return claimRow{}, ErrEmpty
	}
	if err != nil {
		return claimRow{}, err
	}
	upQ := `UPDATE ` + s.table() + ` SET state = ?, attempts = attempts + 1, updated_at = ?, note = ?
		WHERE id = ? AND state = ?`
	res, err := conn.ExecContext(ctx, upQ,
		feboundary.StateInFlight, s.timeArg(now), "claimed for fixture boundary",
		r.ID, feboundary.StateQueued,
	)
	if err != nil {
		return claimRow{}, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return claimRow{}, ErrEmpty
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return claimRow{}, err
	}
	committed = true
	r.State = feboundary.StateInFlight
	r.Attempts++
	return claimRow{row: r}, nil
}

func (s *Store) finish(ctx context.Context, id, state, mockReqID, mockCode, sourceID, note string) error {
	now := s.now()
	q := `UPDATE ` + s.table() + ` SET state = ?, mock_request_id = ?, mock_code = ?, source_id = ?, note = ?, updated_at = ?
		WHERE id = ?`
	_, err := s.db.ExecContext(ctx, s.rebind(q),
		state, mockReqID, mockCode, sourceID, note, s.timeArg(now), id,
	)
	return err
}

func (s *Store) requeue(ctx context.Context, id string, attempts int, note string) error {
	now := s.now()
	next := now.Add(defaultUnavailableBackoff)
	q := `UPDATE ` + s.table() + ` SET state = ?, next_attempt_at = ?, note = ?, updated_at = ?
		WHERE id = ?`
	_, err := s.db.ExecContext(ctx, s.rebind(q),
		feboundary.StateQueued, s.timeArg(next), note, s.timeArg(now), id,
	)
	return err
}

func (s *Store) isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "ffq-" + hex.EncodeToString(b[:])
}

// ProcessResult summarizes one worker pass.
type ProcessResult struct {
	RowID       string
	Operation   string
	State       string
	Attempts    int
	Retried     bool
	MockRequest string
}

// ProcessNext claims one queued row and drives feboundary→femock.
func (s *Store) ProcessNext(ctx context.Context, eng *feboundary.Engine) (*ProcessResult, error) {
	if eng == nil {
		return nil, fmt.Errorf("fefixqueue: nil engine")
	}
	claim, err := s.claimNext(ctx)
	if err != nil {
		return nil, err
	}
	row := claim.row

	payload, err := parsePayload(row.PayloadJSON)
	if err != nil {
		_ = s.finish(ctx, row.ID, StateDead, "", "", "", "invalid payload_json")
		return &ProcessResult{RowID: row.ID, Operation: row.Operation, State: StateDead, Attempts: row.Attempts}, err
	}

	sub, err := eng.Enqueue(row.Operation)
	if err != nil {
		return s.handleTransportFailure(ctx, row, err)
	}

	in := feboundary.ProcessInput{
		SubmissionID:   sub.ID,
		IdentityRef:    row.IdentityRef,
		IdempotencyKey: row.IdempotencyKey,
		Software:       payload.Software,
		ObterEstado:    payload.ObterEstado,
		Consultar:      payload.Consultar,
	}
	result, err := eng.Process(ctx, in)
	if err != nil {
		return s.handleTransportFailure(ctx, row, err)
	}

	state := result.State
	note := result.Note
	if isRetryableEngineState(state) && row.Attempts < defaultMaxAttempts {
		if err := s.requeue(ctx, row.ID, row.Attempts, note+"; retry scheduled"); err != nil {
			return nil, err
		}
		return &ProcessResult{
			RowID: row.ID, Operation: row.Operation, State: feboundary.StateQueued,
			Attempts: row.Attempts, Retried: true,
		}, nil
	}
	if isRetryableEngineState(state) && row.Attempts >= defaultMaxAttempts {
		state = StateDead
		note = "max attempts exhausted; ≠ AGT"
	}

	if err := s.finish(ctx, row.ID, state, result.MockRequestID, result.MockCode, result.SourceID, note); err != nil {
		return nil, err
	}
	return &ProcessResult{
		RowID: row.ID, Operation: row.Operation, State: state,
		Attempts: row.Attempts, MockRequest: result.MockRequestID,
	}, nil
}

func (s *Store) handleTransportFailure(ctx context.Context, row Row, cause error) (*ProcessResult, error) {
	note := "transport/sign failure (sanitized)"
	if row.Attempts < defaultMaxAttempts {
		if err := s.requeue(ctx, row.ID, row.Attempts, note); err != nil {
			return nil, err
		}
		return &ProcessResult{
			RowID: row.ID, Operation: row.Operation, State: feboundary.StateQueued,
			Attempts: row.Attempts, Retried: true,
		}, cause
	}
	if err := s.finish(ctx, row.ID, StateDead, "", "", "", note+"; dead"); err != nil {
		return nil, err
	}
	return &ProcessResult{
		RowID: row.ID, Operation: row.Operation, State: StateDead, Attempts: row.Attempts,
	}, cause
}

// ReconcileObterEstado polls mock obterEstado for a prior mock_request_id (RM-FE-004 mock-only).
func (s *Store) ReconcileObterEstado(ctx context.Context, eng *feboundary.Engine, rowID, identityRef, requestID, taxNIF string) (*ProcessResult, error) {
	if eng == nil || rowID == "" || identityRef == "" || requestID == "" || taxNIF == "" {
		return nil, ErrInvalidInput
	}
	row, err := s.Get(ctx, rowID)
	if err != nil {
		return nil, err
	}
	if row.State != feboundary.StateOK {
		return nil, fmt.Errorf("%w: row not fixture_boundary_ok", ErrInvalidInput)
	}
	sub, err := eng.Enqueue(feboundary.OpObterEstado)
	if err != nil {
		return nil, err
	}
	in := feboundary.ProcessInput{
		SubmissionID:   sub.ID,
		IdentityRef:    identityRef,
		IdempotencyKey: row.IdempotencyKey + ":reconcile:" + requestID,
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: taxNIF,
			RequestID:             requestID,
		},
	}
	result, err := eng.Process(ctx, in)
	if err != nil {
		return nil, err
	}
	note := row.Note + "; reconcile obterEstado=" + result.State + " (≠ AGT accepted)"
	if err := s.finish(ctx, rowID, row.State, result.MockRequestID, result.MockCode, result.SourceID, note); err != nil {
		return nil, err
	}
	return &ProcessResult{
		RowID: rowID, Operation: feboundary.OpObterEstado, State: row.State,
		Attempts: row.Attempts, MockRequest: result.MockRequestID,
	}, nil
}
