package adminops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrValidation is fail-closed input validation.
	ErrValidation = errors.New("adminops: validação")
	// ErrNotFound means submission missing.
	ErrNotFound = errors.New("adminops: não encontrado")
	// ErrConflict means concurrency or idempotency conflict.
	ErrConflict = errors.New("adminops: conflito")
	// ErrForbiddenEnv blocks simulator-driven ops mutations in production.
	ErrForbiddenEnv = errors.New("adminops: ambiente proibido para acção")
)

const (
	ActionRetry        = "retry"
	ActionCancel       = "cancel"
	ActionManualReview = "manual_review"

	DispositionCancelled    = "cancelled"
	DispositionManualReview = "manual_review"

	QueueCancelled = "cancelled"
)

// ActionInput is a secure ops mutation (RM-BO-016).
type ActionInput struct {
	SubmissionID      string
	Action            string
	IdempotencyKey    string
	ExpectedUpdatedAt time.Time // optimistic concurrency
	AuthorityMode     string    // simulator|…
	FiscalEnv         string    // development|homologation|production
	Now               time.Time
}

// ActionResult is the sanitized post-action snapshot.
type ActionResult struct {
	SubmissionID     string
	QueueStatus      string
	OutboxState      string
	Disposition      string
	IdempotentReplay bool
	UpdatedAt        time.Time
}

// ApplyQueueAction performs retry|cancel|manual_review with idempotency + concurrency.
// Never touches payload/JWS/secrets. Simulator + production is fail-closed.
func (s *Store) ApplyQueueAction(ctx context.Context, in ActionInput) (ActionResult, error) {
	subID := strings.TrimSpace(in.SubmissionID)
	action := strings.TrimSpace(in.Action)
	idem := strings.TrimSpace(in.IdempotencyKey)
	if subID == "" || action == "" || idem == "" {
		return ActionResult{}, fmt.Errorf("%w: submission_id/action/Idempotency-Key obrigatórios", ErrValidation)
	}
	if action != ActionRetry && action != ActionCancel && action != ActionManualReview {
		return ActionResult{}, fmt.Errorf("%w: acção inválida", ErrValidation)
	}
	if in.ExpectedUpdatedAt.IsZero() {
		return ActionResult{}, fmt.Errorf("%w: expected_updated_at obrigatório", ErrValidation)
	}
	if err := GuardSimulatorNotProduction(in.AuthorityMode, in.FiscalEnv); err != nil {
		return ActionResult{}, err
	}
	now := in.Now.UTC().Truncate(time.Microsecond)
	if now.IsZero() {
		now = time.Now().UTC().Truncate(time.Microsecond)
	}

	if prev, ok, err := s.lookupIdempotency(ctx, idem); err != nil {
		return ActionResult{}, err
	} else if ok {
		if prev.action != action || prev.submissionID != subID {
			return ActionResult{}, fmt.Errorf("%w: Idempotency-Key reutilizada com contexto diferente", ErrConflict)
		}
		cur, err := s.getOutboxRow(ctx, subID)
		if err != nil {
			return ActionResult{}, err
		}
		return ActionResult{
			SubmissionID: subID, QueueStatus: prev.resultStatus,
			OutboxState: cur.state, Disposition: cur.disposition,
			IdempotentReplay: true, UpdatedAt: cur.updatedAt,
		}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ActionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := s.getOutboxRowTx(ctx, tx, subID)
	if err != nil {
		return ActionResult{}, err
	}
	if !sameUpdatedAt(cur.updatedAt, in.ExpectedUpdatedAt) {
		return ActionResult{}, fmt.Errorf("%w: updated_at desactualizado", ErrConflict)
	}

	newState := cur.state
	newDisp := cur.disposition
	switch action {
	case ActionRetry:
		if cur.state == "succeeded" {
			return ActionResult{}, fmt.Errorf("%w: não é possível retry de succeeded", ErrValidation)
		}
		if cur.disposition == DispositionCancelled {
			return ActionResult{}, fmt.Errorf("%w: cancelled é terminal sem reopen", ErrValidation)
		}
		newState = "pending"
		newDisp = ""
	case ActionCancel:
		if cur.state == "succeeded" {
			return ActionResult{}, fmt.Errorf("%w: não é possível cancel de succeeded", ErrValidation)
		}
		newState = "dead"
		newDisp = DispositionCancelled
	case ActionManualReview:
		if cur.state == "succeeded" {
			return ActionResult{}, fmt.Errorf("%w: não é possível manual_review de succeeded aceite", ErrValidation)
		}
		if cur.disposition == DispositionCancelled {
			return ActionResult{}, fmt.Errorf("%w: cancelled é terminal", ErrValidation)
		}
		newState = "dead"
		newDisp = DispositionManualReview
	}

	var avail any
	if newState == "pending" {
		avail = s.timeArg(now)
	} else {
		avail = s.timeArg(cur.availableAt)
	}
	var dispArg any
	if newDisp == "" {
		dispArg = nil
	} else {
		dispArg = newDisp
	}

	q := `UPDATE ` + s.t("outbox_messages") + ` SET
		state = ` + s.p(1) + `,
		available_at = ` + s.p(2) + `,
		ops_disposition = ` + s.p(3) + `,
		updated_at = ` + s.p(4) + `
	WHERE submission_id = ` + s.p(5) + ` AND updated_at = ` + s.p(6)
	res, err := tx.ExecContext(ctx, q, newState, avail, dispArg, s.timeArg(now), subID, cur.updatedRaw)
	if err != nil {
		return ActionResult{}, fmt.Errorf("adminops: update outbox: %w", err)
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ActionResult{}, fmt.Errorf("%w: updated_at desactualizado", ErrConflict)
	}

	queueStatus := DeriveQueueStatusWithDisposition(newState, "", 0, newDisp)
	iq := `INSERT INTO ` + s.t("admin_ops_action_idempotency") + ` (
		idempotency_key, action, submission_id, result_queue_status, created_at
	) VALUES (` + s.ph(5) + `)`
	if _, err := tx.ExecContext(ctx, iq, idem, action, subID, queueStatus, s.timeArg(now)); err != nil {
		if isUnique(err) {
			return ActionResult{}, fmt.Errorf("%w: Idempotency-Key em corrida", ErrConflict)
		}
		return ActionResult{}, fmt.Errorf("adminops: idempotency insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{
		SubmissionID: subID, QueueStatus: queueStatus, OutboxState: newState,
		Disposition: newDisp, UpdatedAt: now,
	}, nil
}

// GuardSimulatorNotProduction is fail-closed for ops mutations (RM-BO-016).
func GuardSimulatorNotProduction(authorityMode, fiscalEnv string) error {
	mode := strings.ToLower(strings.TrimSpace(authorityMode))
	env := strings.ToLower(strings.TrimSpace(fiscalEnv))
	if mode == "" {
		mode = "simulator"
	}
	if mode == "simulator" && env == "production" {
		return fmt.Errorf("%w: simulator proibido em production", ErrForbiddenEnv)
	}
	return nil
}

func sameUpdatedAt(a, b time.Time) bool {
	a = a.UTC().Truncate(time.Microsecond)
	b = b.UTC().Truncate(time.Microsecond)
	return a.Equal(b)
}

type outboxRow struct {
	state       string
	disposition string
	availableAt time.Time
	updatedAt   time.Time
	updatedRaw  any // exact DB value for optimistic WHERE
}

func (s *Store) getOutboxRow(ctx context.Context, submissionID string) (outboxRow, error) {
	return s.getOutboxRowTx(ctx, s.db, submissionID)
}

func (s *Store) getOutboxRowTx(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, submissionID string) (outboxRow, error) {
	row := q.QueryRowContext(ctx, `SELECT state, COALESCE(ops_disposition, ''), available_at, updated_at
		FROM `+s.t("outbox_messages")+` WHERE submission_id = `+s.ph(1), submissionID)
	var o outboxRow
	var avail any
	err := row.Scan(&o.state, &o.disposition, &avail, &o.updatedRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return outboxRow{}, ErrNotFound
	}
	if err != nil {
		return outboxRow{}, err
	}
	o.availableAt, err = parseTime(avail)
	if err != nil {
		return outboxRow{}, err
	}
	o.updatedAt, err = parseTime(o.updatedRaw)
	return o, err
}

type idemRow struct {
	action       string
	submissionID string
	resultStatus string
}

func (s *Store) lookupIdempotency(ctx context.Context, key string) (idemRow, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT action, submission_id, result_queue_status
		FROM `+s.t("admin_ops_action_idempotency")+` WHERE idempotency_key = `+s.ph(1), key)
	var out idemRow
	err := row.Scan(&out.action, &out.submissionID, &out.resultStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return idemRow{}, false, nil
	}
	if err != nil {
		return idemRow{}, false, err
	}
	return out, true, nil
}

func (s *Store) timeArg(t time.Time) any {
	if s.dialect == DialectPostgres {
		return t
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

func (s *Store) ph(n int) string {
	if s.dialect == DialectPostgres {
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

func isUnique(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}

// DeriveQueueStatusWithDisposition extends DeriveQueueStatus with ops disposition (RM-BO-016).
func DeriveQueueStatusWithDisposition(outboxState, outcome string, attempts int64, disposition string) string {
	switch strings.TrimSpace(disposition) {
	case DispositionCancelled:
		return QueueCancelled
	case DispositionManualReview:
		return QueueManualReview
	}
	return DeriveQueueStatus(outboxState, outcome, attempts)
}
