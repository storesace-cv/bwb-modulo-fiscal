package persistence

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (s *Store) sealPostgres(ctx context.Context, req SealRequest, hash []byte) (*SealResult, error) {
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

	res, err := s.sealInQuerier(ctx, &pgQuerier{tx: tx}, req, hash, true)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("persistence: commit: %w", err)
	}
	committed = true
	return res, nil
}

type pgQuerier struct {
	tx *sql.Tx
}

func (q *pgQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.tx.ExecContext(ctx, query, args...)
}

func (q *pgQuerier) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return q.tx.QueryRowContext(ctx, query, args...)
}

func (s *Store) sealSQLite(ctx context.Context, req SealRequest, hash []byte) (*SealResult, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("persistence: conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return nil, fmt.Errorf("persistence: BEGIN IMMEDIATE: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	res, err := s.sealInQuerier(ctx, &sqliteQuerier{conn: conn}, req, hash, false)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return nil, fmt.Errorf("persistence: commit: %w", err)
	}
	committed = true
	return res, nil
}

type sqliteQuerier struct {
	conn *sql.Conn
}

func (q *sqliteQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.conn.ExecContext(ctx, query, args...)
}

func (q *sqliteQuerier) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return q.conn.QueryRowContext(ctx, query, args...)
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) sealInQuerier(ctx context.Context, q querier, req SealRequest, hash []byte, postgres bool) (*SealResult, error) {
	t := tablePrefix(postgres)
	now := s.stamp()
	nowArg := any(now)
	if !postgres {
		nowArg = now.Format(time.RFC3339Nano)
	}

	// Resolve idempotency under row lock / writer lock.
	hit, err := s.resolveIdempotency(ctx, q, t, postgres, req, hash, nowArg)
	if err != nil {
		return nil, err
	}
	if hit != nil {
		return hit, nil
	}

	// External ID uniqueness (VS-T05).
	var existingDoc string
	err = q.QueryRowContext(ctx,
		`SELECT id FROM `+t("documents")+` WHERE scope_id = `+ph(postgres, 1)+` AND external_id = `+ph(postgres, 2),
		req.Intent.ScopeID, req.Intent.ExternalID,
	).Scan(&existingDoc)
	if err == nil {
		return nil, ErrExternalIDConflict
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("persistence: external_id lookup: %w", err)
	}

	fiscalSeq, err := s.nextFiscalSeq(ctx, q, t, postgres, req.Intent.ScopeID, req.SeriesCode)
	if err != nil {
		return nil, err
	}

	docID, err := newID()
	if err != nil {
		return nil, err
	}
	ledgerID, err := newID()
	if err != nil {
		return nil, err
	}
	outboxID, err := newID()
	if err != nil {
		return nil, err
	}
	submissionID, err := newID()
	if err != nil {
		return nil, err
	}

	issuedAt := req.Intent.IssuedAtUTC // already normalized RFC3339Nano UTC
	var issuedAtArg any = issuedAt
	if postgres {
		parsed, err := time.Parse(time.RFC3339Nano, issuedAt)
		if err != nil {
			return nil, fmt.Errorf("persistence: issued_at: %w", err)
		}
		issuedAtArg = parsed.UTC()
	}

	custTax := nullIfEmpty(req.Intent.CustomerTaxID)
	custName := nullIfEmpty(req.Intent.CustomerName)
	reqSeries := nullIfEmpty(req.Intent.RequestedSeries)

	_, err = q.ExecContext(ctx, `
		INSERT INTO `+t("documents")+` (
			id, scope_id, external_id, document_type, currency, issued_at,
			requested_series, series_code, fiscal_seq,
			seller_tax_id, seller_name, customer_tax_id, customer_name,
			created_at, sealed_at
		) VALUES (`+placeholders(postgres, 15)+`)`,
		docID, req.Intent.ScopeID, req.Intent.ExternalID, req.Intent.DocumentType, req.Intent.Currency, issuedAtArg,
		reqSeries, req.SeriesCode, fiscalSeq,
		req.Intent.SellerTaxID, req.Intent.SellerName, custTax, custName,
		nowArg, nowArg,
	)
	if err != nil {
		if mapped := mapExternalIDConflict(err); errors.Is(mapped, ErrExternalIDConflict) {
			return nil, ErrExternalIDConflict
		}
		return nil, fmt.Errorf("persistence: insert document: %w", err)
	}

	for i, ln := range req.Intent.Lines {
		_, err = q.ExecContext(ctx, `
			INSERT INTO `+t("document_lines")+` (
				document_id, line_no, line_id, description, quantity_scaled, unit_price_cents, tax_code
			) VALUES (`+placeholders(postgres, 7)+`)`,
			docID, i+1, ln.LineID, ln.Description, ln.Quantity.Scaled(), ln.UnitPrice.Cents(), ln.TaxCode,
		)
		if err != nil {
			return nil, fmt.Errorf("persistence: insert line: %w", err)
		}
	}

	_, err = q.ExecContext(ctx, `
		INSERT INTO `+t("ledger_events")+` (
			id, document_id, seq, event_type, from_status, to_status, created_at
		) VALUES (`+placeholders(postgres, 7)+`)`,
		ledgerID, docID, int64(1), "status_transition", nil, "sealed_locally", nowArg,
	)
	if err != nil {
		return nil, fmt.Errorf("persistence: insert ledger: %w", err)
	}

	_, err = q.ExecContext(ctx, `
		INSERT INTO `+t("outbox_messages")+` (
			id, document_id, message_type, submission_id, state, available_at, created_at, updated_at
		) VALUES (`+placeholders(postgres, 8)+`)`,
		outboxID, docID, "authority_submission", submissionID, "pending", nowArg, nowArg, nowArg,
	)
	if err != nil {
		return nil, fmt.Errorf("persistence: insert outbox: %w", err)
	}

	_, err = q.ExecContext(ctx, `
		UPDATE `+t("idempotency_records")+`
		SET document_id = `+ph(postgres, 1)+`, state = 'completed', updated_at = `+ph(postgres, 2)+`
		WHERE scope_id = `+ph(postgres, 3)+` AND idempotency_key = `+ph(postgres, 4),
		docID, nowArg, req.Intent.ScopeID, req.IdempotencyKey,
	)
	if err != nil {
		return nil, fmt.Errorf("persistence: complete idempotency: %w", err)
	}

	return &SealResult{
		DocumentID:    docID,
		FiscalSeq:     fiscalSeq,
		SeriesCode:    req.SeriesCode,
		ScopeID:       req.Intent.ScopeID,
		ExternalID:    req.Intent.ExternalID,
		SubmissionID:  submissionID,
		IdempotentHit: false,
	}, nil
}

func (s *Store) resolveIdempotency(ctx context.Context, q querier, t func(string) string, postgres bool, req SealRequest, hash []byte, nowArg any) (*SealResult, error) {
	_, err := q.ExecContext(ctx, `
		INSERT INTO `+t("idempotency_records")+` (
			scope_id, idempotency_key, request_hash, document_id, state, created_at, updated_at
		) VALUES (`+placeholders(postgres, 7)+`)
		`+idempotencyConflictClause(postgres),
		req.Intent.ScopeID, req.IdempotencyKey, hash, nil, "in_progress", nowArg, nowArg,
	)
	if err != nil {
		return nil, fmt.Errorf("persistence: insert idempotency: %w", err)
	}

	var (
		storedHash []byte
		docID      sql.NullString
		state      string
	)
	selectSQL := `
		SELECT request_hash, document_id, state
		FROM ` + t("idempotency_records") + `
		WHERE scope_id = ` + ph(postgres, 1) + ` AND idempotency_key = ` + ph(postgres, 2)
	if postgres {
		selectSQL += ` FOR UPDATE`
	}
	err = q.QueryRowContext(ctx, selectSQL, req.Intent.ScopeID, req.IdempotencyKey).Scan(&storedHash, &docID, &state)
	if err != nil {
		return nil, fmt.Errorf("persistence: lock idempotency: %w", err)
	}

	switch state {
	case "completed":
		if !bytes.Equal(storedHash, hash) {
			return nil, ErrIdempotencyConflict
		}
		if !docID.Valid {
			return nil, fmt.Errorf("persistence: completed idempotency without document_id")
		}
		return s.loadCompletedResult(ctx, q, t, postgres, docID.String)
	case "in_progress":
		if !bytes.Equal(storedHash, hash) {
			return nil, ErrIdempotencyConflict
		}
		return nil, nil // we own the in-progress row; proceed to seal
	default:
		return nil, fmt.Errorf("persistence: unexpected idempotency state %q", state)
	}
}

func (s *Store) loadCompletedResult(ctx context.Context, q querier, t func(string) string, postgres bool, docID string) (*SealResult, error) {
	var (
		fiscalSeq    int64
		seriesCode   string
		externalID   string
		scopeID      string
		submissionID string
	)
	err := q.QueryRowContext(ctx, `
		SELECT d.scope_id, d.external_id, d.series_code, d.fiscal_seq, o.submission_id
		FROM `+t("documents")+` d
		JOIN `+t("outbox_messages")+` o ON o.document_id = d.id
		WHERE d.id = `+ph(postgres, 1),
		docID,
	).Scan(&scopeID, &externalID, &seriesCode, &fiscalSeq, &submissionID)
	if err != nil {
		return nil, fmt.Errorf("persistence: load completed: %w", err)
	}
	return &SealResult{
		DocumentID:    docID,
		FiscalSeq:     fiscalSeq,
		SeriesCode:    seriesCode,
		ScopeID:       scopeID,
		ExternalID:    externalID,
		SubmissionID:  submissionID,
		IdempotentHit: true,
	}, nil
}

func (s *Store) nextFiscalSeq(ctx context.Context, q querier, t func(string) string, postgres bool, scopeID, seriesCode string) (int64, error) {
	if postgres {
		_, err := q.ExecContext(ctx, `
			INSERT INTO `+t("series_counters")+` (scope_id, series_code, last_seq)
			VALUES ($1, $2, 0)
			ON CONFLICT (scope_id, series_code) DO NOTHING`,
			scopeID, seriesCode,
		)
		if err != nil {
			return 0, fmt.Errorf("persistence: ensure series counter: %w", err)
		}
		var last int64
		err = q.QueryRowContext(ctx, `
			SELECT last_seq FROM `+t("series_counters")+`
			WHERE scope_id = $1 AND series_code = $2
			FOR UPDATE`,
			scopeID, seriesCode,
		).Scan(&last)
		if err != nil {
			return 0, fmt.Errorf("persistence: lock series counter: %w", err)
		}
		next := last + 1
		_, err = q.ExecContext(ctx, `
			UPDATE `+t("series_counters")+`
			SET last_seq = $1
			WHERE scope_id = $2 AND series_code = $3`,
			next, scopeID, seriesCode,
		)
		if err != nil {
			return 0, fmt.Errorf("persistence: update series counter: %w", err)
		}
		return next, nil
	}

	_, err := q.ExecContext(ctx, `
		INSERT OR IGNORE INTO series_counters (scope_id, series_code, last_seq)
		VALUES (?, ?, 0)`,
		scopeID, seriesCode,
	)
	if err != nil {
		return 0, fmt.Errorf("persistence: ensure series counter: %w", err)
	}
	var last int64
	err = q.QueryRowContext(ctx, `
		SELECT last_seq FROM series_counters
		WHERE scope_id = ? AND series_code = ?`,
		scopeID, seriesCode,
	).Scan(&last)
	if err != nil {
		return 0, fmt.Errorf("persistence: read series counter: %w", err)
	}
	next := last + 1
	_, err = q.ExecContext(ctx, `
		UPDATE series_counters SET last_seq = ?
		WHERE scope_id = ? AND series_code = ?`,
		next, scopeID, seriesCode,
	)
	if err != nil {
		return 0, fmt.Errorf("persistence: update series counter: %w", err)
	}
	return next, nil
}

func tablePrefix(postgres bool) func(string) string {
	return func(name string) string {
		if postgres {
			return "fiscal." + name
		}
		return name
	}
}

func idempotencyConflictClause(postgres bool) string {
	if postgres {
		return `ON CONFLICT (scope_id, idempotency_key) DO NOTHING`
	}
	return `ON CONFLICT (scope_id, idempotency_key) DO NOTHING`
}

func ph(postgres bool, n int) string {
	if postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func placeholders(postgres bool, n int) string {
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		if postgres {
			parts[i] = fmt.Sprintf("$%d", i+1)
		} else {
			parts[i] = "?"
		}
	}
	return joinComma(parts)
}

func joinComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i]
	}
	return out
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
