package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaljws"
)

const (
	outboxPending   = "pending"
	outboxInFlight  = "in_flight"
	outboxSucceeded = "succeeded"

	ledgerSealedLocally         = "sealed_locally"
	ledgerAuthorityProcessing   = "authority_processing"
	ledgerAuthorityAccepted     = "authority_accepted"
	ledgerAuthorityRejected     = "authority_rejected"
	ledgerAuthorityUnknown      = "authority_outcome_unknown"
	defaultUnavailableBackoff   = 5 * time.Second
	defaultInFlightReclaimAfter = 30 * time.Second
)

var (
	// ErrOutboxEmpty means no claimable authority_submission message.
	ErrOutboxEmpty = errors.New("persistence: outbox empty")
	// ErrOutboxAlreadyDone means the submission already reached a terminal outbox state.
	ErrOutboxAlreadyDone = errors.New("persistence: outbox already succeeded")
)

// AuthorityClient is the simulator/AGT transport seam (slice uses simulator only).
type AuthorityClient interface {
	Submit(ctx context.Context, req simulator.Request) (simulator.Result, error)
}

// OutboxProcessResult summarizes one worker pass.
type OutboxProcessResult struct {
	SubmissionID string
	DocumentID   string
	AttemptNo    int64
	Outcome      string // ledger to_status or "retried_unavailable"
	OutboxState  string
	JWSAttached  bool
}

// ProcessOpts configures one outbox worker pass.
type ProcessOpts struct {
	// Signer, when non-nil, attaches a technical RS256 JWS envelope (ephemeral; not FE-certified).
	Signer *fiscaljws.Signer
}

// ProcessNextAuthoritySubmission claims one outbox row, calls the authority, and records attempt/response.
// Simulator unavailable → outbox returns to pending with backoff; never authority_accepted.
// Already-succeeded submissions are skipped (at-least-once safe).
func (s *Store) ProcessNextAuthoritySubmission(ctx context.Context, client AuthorityClient, opts ProcessOpts) (*OutboxProcessResult, error) {
	if client == nil {
		return nil, fmt.Errorf("persistence: nil authority client")
	}
	postgres := s.dialect == DialectPostgres
	if postgres {
		return s.processOutboxPostgres(ctx, client, opts)
	}
	return s.processOutboxSQLite(ctx, client, opts)
}

func (s *Store) processOutboxPostgres(ctx context.Context, client AuthorityClient, opts ProcessOpts) (*OutboxProcessResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("persistence: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	q := &pgQuerier{tx: tx}
	claim, err := s.claimOutbox(ctx, q, true)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("persistence: commit claim: %w", err)
	}
	committed = true
	return s.finishOutbox(ctx, client, claim, true, opts)
}

func (s *Store) processOutboxSQLite(ctx context.Context, client AuthorityClient, opts ProcessOpts) (*OutboxProcessResult, error) {
	claim, err := func() (outboxClaim, error) {
		conn, err := s.db.Conn(ctx)
		if err != nil {
			return outboxClaim{}, fmt.Errorf("persistence: conn: %w", err)
		}
		defer conn.Close()
		if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
			return outboxClaim{}, fmt.Errorf("persistence: BEGIN IMMEDIATE: %w", err)
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			}
		}()
		q := &sqliteQuerier{conn: conn}
		claim, err := s.claimOutbox(ctx, q, false)
		if err != nil {
			return outboxClaim{}, err
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return outboxClaim{}, fmt.Errorf("persistence: commit claim: %w", err)
		}
		committed = true
		return claim, nil
	}()
	if err != nil {
		return nil, err
	}
	return s.finishOutbox(ctx, client, claim, false, opts)
}

type outboxClaim struct {
	ID           string
	DocumentID   string
	SubmissionID string
}

func (s *Store) claimOutbox(ctx context.Context, q querier, postgres bool) (outboxClaim, error) {
	t := tablePrefix(postgres)
	now := s.stamp().UTC().Truncate(time.Microsecond)
	var nowArg any = now
	if !postgres {
		nowArg = formatUTCMicro(now)
	}
	reclaimBefore := now.Add(-defaultInFlightReclaimAfter)
	var reclaimArg any = reclaimBefore
	if !postgres {
		reclaimArg = formatUTCMicro(reclaimBefore)
	}

	// Reclaim stale in_flight (worker crash) back to pending.
	_, _ = q.ExecContext(ctx,
		`UPDATE `+t("outbox_messages")+` SET state = `+ph(postgres, 1)+`, available_at = `+ph(postgres, 2)+`, updated_at = `+ph(postgres, 3)+`
		 WHERE state = `+ph(postgres, 4)+` AND updated_at <= `+ph(postgres, 5),
		outboxPending, nowArg, nowArg, outboxInFlight, reclaimArg,
	)

	var claim outboxClaim
	var selectSQL string
	if postgres {
		selectSQL = `SELECT id, document_id, submission_id FROM ` + t("outbox_messages") + `
			WHERE state = $1 AND available_at <= $2
			ORDER BY available_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1`
	} else {
		selectSQL = `SELECT id, document_id, submission_id FROM ` + t("outbox_messages") + `
			WHERE state = ? AND available_at <= ?
			ORDER BY available_at ASC
			LIMIT 1`
	}
	err := q.QueryRowContext(ctx, selectSQL, outboxPending, nowArg).Scan(&claim.ID, &claim.DocumentID, &claim.SubmissionID)
	if errors.Is(err, sql.ErrNoRows) {
		return outboxClaim{}, ErrOutboxEmpty
	}
	if err != nil {
		return outboxClaim{}, fmt.Errorf("persistence: select outbox: %w", err)
	}

	res, err := q.ExecContext(ctx,
		`UPDATE `+t("outbox_messages")+` SET state = `+ph(postgres, 1)+`, updated_at = `+ph(postgres, 2)+`
		 WHERE id = `+ph(postgres, 3)+` AND state = `+ph(postgres, 4),
		outboxInFlight, nowArg, claim.ID, outboxPending,
	)
	if err != nil {
		return outboxClaim{}, fmt.Errorf("persistence: claim outbox: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return outboxClaim{}, err
	}
	if n != 1 {
		return outboxClaim{}, ErrOutboxEmpty
	}
	return claim, nil
}

func (s *Store) finishOutbox(ctx context.Context, client AuthorityClient, claim outboxClaim, postgres bool, opts ProcessOpts) (*OutboxProcessResult, error) {
	req := simulator.Request{
		SubmissionID: claim.SubmissionID,
		DocumentID:   claim.DocumentID,
	}
	jwsAttached := false
	if opts.Signer != nil {
		compact, err := opts.Signer.SignEnvelope(claim.SubmissionID, claim.DocumentID, s.stamp().UTC())
		if err != nil {
			_ = s.releaseOutboxUnavailable(ctx, claim, postgres)
			return nil, fmt.Errorf("persistence: sign JWS: %w", err)
		}
		req.JWS = compact
		jwsAttached = true
	}

	simRes, err := client.Submit(ctx, req)
	if errors.Is(err, simulator.ErrUnavailable) {
		if err := s.releaseOutboxUnavailable(ctx, claim, postgres); err != nil {
			return nil, err
		}
		return &OutboxProcessResult{
			SubmissionID: claim.SubmissionID,
			DocumentID:   claim.DocumentID,
			Outcome:      "retried_unavailable",
			OutboxState:  outboxPending,
			JWSAttached:  jwsAttached,
		}, nil
	}
	if err != nil {
		_ = s.releaseOutboxUnavailable(ctx, claim, postgres)
		return nil, err
	}

	ledgerTo, err := mapSimulatorOutcome(simRes.Outcome)
	if err != nil {
		_ = s.releaseOutboxUnavailable(ctx, claim, postgres)
		return nil, err
	}
	out, err := s.persistAuthoritySuccess(ctx, claim, simRes, ledgerTo, postgres)
	if out != nil {
		out.JWSAttached = jwsAttached
	}
	return out, err
}

func mapSimulatorOutcome(o simulator.Outcome) (string, error) {
	switch o {
	case simulator.OutcomeAccept:
		return ledgerAuthorityAccepted, nil
	case simulator.OutcomeReject:
		return ledgerAuthorityRejected, nil
	case simulator.OutcomeUnknown:
		return ledgerAuthorityUnknown, nil
	default:
		return "", fmt.Errorf("persistence: outcome simulador inválido %q", o)
	}
}

func (s *Store) releaseOutboxUnavailable(ctx context.Context, claim outboxClaim, postgres bool) error {
	return s.withQuerier(ctx, postgres, func(q querier) error {
		t := tablePrefix(postgres)
		now := s.stamp().UTC().Truncate(time.Microsecond)
		avail := now.Add(defaultUnavailableBackoff)
		var nowArg, availArg any = now, avail
		if !postgres {
			nowArg = formatUTCMicro(now)
			availArg = formatUTCMicro(avail)
		}
		_, err := q.ExecContext(ctx,
			`UPDATE `+t("outbox_messages")+` SET state = `+ph(postgres, 1)+`, available_at = `+ph(postgres, 2)+`, updated_at = `+ph(postgres, 3)+`
			 WHERE id = `+ph(postgres, 4)+` AND state = `+ph(postgres, 5),
			outboxPending, availArg, nowArg, claim.ID, outboxInFlight,
		)
		return err
	})
}

func (s *Store) persistAuthoritySuccess(ctx context.Context, claim outboxClaim, simRes simulator.Result, ledgerTo string, postgres bool) (*OutboxProcessResult, error) {
	var out *OutboxProcessResult
	err := s.withQuerier(ctx, postgres, func(q querier) error {
		t := tablePrefix(postgres)
		now := s.stamp().UTC().Truncate(time.Microsecond)
		var nowArg any = now
		if !postgres {
			nowArg = formatUTCMicro(now)
		}

		// Idempotent: if already succeeded, do nothing.
		var state string
		err := q.QueryRowContext(ctx,
			`SELECT state FROM `+t("outbox_messages")+` WHERE id = `+ph(postgres, 1), claim.ID,
		).Scan(&state)
		if err != nil {
			return fmt.Errorf("persistence: outbox state: %w", err)
		}
		if state == outboxSucceeded {
			out = &OutboxProcessResult{
				SubmissionID: claim.SubmissionID,
				DocumentID:   claim.DocumentID,
				Outcome:      ledgerTo,
				OutboxState:  outboxSucceeded,
			}
			return ErrOutboxAlreadyDone
		}

		var latest string
		err = q.QueryRowContext(ctx,
			`SELECT to_status FROM `+t("ledger_events")+` WHERE document_id = `+ph(postgres, 1)+` ORDER BY seq DESC LIMIT 1`,
			claim.DocumentID,
		).Scan(&latest)
		if err != nil {
			return fmt.Errorf("persistence: latest ledger: %w", err)
		}
		switch latest {
		case ledgerAuthorityAccepted, ledgerAuthorityRejected, ledgerAuthorityUnknown:
			_, err := q.ExecContext(ctx,
				`UPDATE `+t("outbox_messages")+` SET state = `+ph(postgres, 1)+`, updated_at = `+ph(postgres, 2)+`
				 WHERE id = `+ph(postgres, 3),
				outboxSucceeded, nowArg, claim.ID,
			)
			if err != nil {
				return err
			}
			out = &OutboxProcessResult{
				SubmissionID: claim.SubmissionID,
				DocumentID:   claim.DocumentID,
				Outcome:      latest,
				OutboxState:  outboxSucceeded,
			}
			return nil
		case ledgerSealedLocally, ledgerAuthorityProcessing:
			// continue
		default:
			return fmt.Errorf("persistence: estado ledger inesperado %q", latest)
		}

		var maxAttempt sql.NullInt64
		err = q.QueryRowContext(ctx,
			`SELECT MAX(attempt_no) FROM `+t("authority_attempts")+` WHERE submission_id = `+ph(postgres, 1),
			claim.SubmissionID,
		).Scan(&maxAttempt)
		if err != nil {
			return fmt.Errorf("persistence: max attempt: %w", err)
		}
		attemptNo := int64(1)
		if maxAttempt.Valid {
			attemptNo = maxAttempt.Int64 + 1
		}

		attemptID, err := newID()
		if err != nil {
			return err
		}
		respID, err := newID()
		if err != nil {
			return err
		}

		var nextSeq int64
		err = q.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(seq), 0) + 1 FROM `+t("ledger_events")+` WHERE document_id = `+ph(postgres, 1),
			claim.DocumentID,
		).Scan(&nextSeq)
		if err != nil {
			return fmt.Errorf("persistence: next ledger seq: %w", err)
		}

		fromStatus := latest
		if latest == ledgerSealedLocally {
			ledgerProcID, err := newID()
			if err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx,
				`INSERT INTO `+t("ledger_events")+` (
					id, document_id, seq, event_type, from_status, to_status, created_at
				) VALUES (`+placeholders(postgres, 7)+`)`,
				ledgerProcID, claim.DocumentID, nextSeq, "status_transition", ledgerSealedLocally, ledgerAuthorityProcessing, nowArg,
			); err != nil {
				return fmt.Errorf("persistence: ledger processing: %w", err)
			}
			fromStatus = ledgerAuthorityProcessing
			nextSeq++
		}

		ledgerOutID, err := newID()
		if err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx,
			`INSERT INTO `+t("ledger_events")+` (
				id, document_id, seq, event_type, from_status, to_status, created_at
			) VALUES (`+placeholders(postgres, 7)+`)`,
			ledgerOutID, claim.DocumentID, nextSeq, "status_transition", fromStatus, ledgerTo, nowArg,
		); err != nil {
			return fmt.Errorf("persistence: ledger outcome: %w", err)
		}

		if _, err := q.ExecContext(ctx,
			`INSERT INTO `+t("authority_attempts")+` (
				id, document_id, submission_id, attempt_no, sent_at
			) VALUES (`+placeholders(postgres, 5)+`)`,
			attemptID, claim.DocumentID, claim.SubmissionID, attemptNo, nowArg,
		); err != nil {
			return fmt.Errorf("persistence: insert attempt: %w", err)
		}
		if _, err := q.ExecContext(ctx,
			`INSERT INTO `+t("authority_responses")+` (
				id, attempt_id, authority_request_id, outcome, received_at
			) VALUES (`+placeholders(postgres, 5)+`)`,
			respID, attemptID, nullIfEmpty(simRes.AuthorityRequestID), ledgerTo, nowArg,
		); err != nil {
			return fmt.Errorf("persistence: insert response: %w", err)
		}

		res, err := q.ExecContext(ctx,
			`UPDATE `+t("outbox_messages")+` SET state = `+ph(postgres, 1)+`, updated_at = `+ph(postgres, 2)+`
			 WHERE id = `+ph(postgres, 3)+` AND state IN (`+ph(postgres, 4)+`, `+ph(postgres, 5)+`)`,
			outboxSucceeded, nowArg, claim.ID, outboxInFlight, outboxPending,
		)
		if err != nil {
			return fmt.Errorf("persistence: outbox succeed: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("persistence: outbox succeed rows=%d", n)
		}

		out = &OutboxProcessResult{
			SubmissionID: claim.SubmissionID,
			DocumentID:   claim.DocumentID,
			AttemptNo:    attemptNo,
			Outcome:      ledgerTo,
			OutboxState:  outboxSucceeded,
		}
		return nil
	})
	if errors.Is(err, ErrOutboxAlreadyDone) {
		return out, nil
	}
	return out, err
}

func (s *Store) withQuerier(ctx context.Context, postgres bool, fn func(querier) error) error {
	if postgres {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		if err := fn(&pgQuerier{tx: tx}); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if err := fn(&sqliteQuerier{conn: conn}); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}
