package secretstore

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SQL is a durable encrypted-at-rest SecretStore (AES-256-GCM ciphertext in DB).
// Master key is supplied out-of-band (env today; KMS/HSM later). Never stores plaintext.
type SQL struct {
	db      *sql.DB
	dialect Dialect
	aead    cipher.AEAD
	now     func() time.Time
}

// NewSQL builds a durable vault. key must be 32 bytes; caller should zero it after return.
func NewSQL(db *sql.DB, dialect Dialect, key []byte, now func() time.Time) (*SQL, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: db nil", ErrValidation)
	}
	switch dialect {
	case DialectPostgres, DialectSQLite:
	default:
		return nil, fmt.Errorf("%w: dialect", ErrValidation)
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &SQL{db: db, dialect: dialect, aead: aead, now: now}, nil
}

// StorageMode reports durable_encrypted (sanitized; never key material).
func (s *SQL) StorageMode() string { return StorageModeDurableEncrypted }

// Put provisions a secret write-only. Returns metadata only.
func (s *SQL) Put(ctx context.Context, ref Ref, plaintext []byte, expiresAt *time.Time) (PutResult, error) {
	if err := ctx.Err(); err != nil {
		return PutResult{}, err
	}
	if err := validateRef(ref); err != nil {
		return PutResult{}, err
	}
	if err := validatePlaintext(ref, plaintext); err != nil {
		return PutResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PutResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := s.loadTx(ctx, tx, ref)
	if err != nil {
		return PutResult{}, err
	}
	if cur != nil && cur.meta.Status == StatusPresent {
		return PutResult{}, fmt.Errorf("%w: já presente (use Rotate)", ErrValidation)
	}
	meta, err := s.writeTx(ctx, tx, ref, plaintext, expiresAt, 1, StatusPresent, cur == nil)
	if err != nil {
		return PutResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return PutResult{}, err
	}
	return PutResult{Metadata: meta}, nil
}

// Rotate replaces material; previous ciphertext is overwritten (not logged).
func (s *SQL) Rotate(ctx context.Context, ref Ref, plaintext []byte, expiresAt *time.Time) (PutResult, error) {
	if err := ctx.Err(); err != nil {
		return PutResult{}, err
	}
	if err := validateRef(ref); err != nil {
		return PutResult{}, err
	}
	if err := validatePlaintext(ref, plaintext); err != nil {
		return PutResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PutResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := s.loadTx(ctx, tx, ref)
	if err != nil {
		return PutResult{}, err
	}
	ver := 1
	insert := true
	if cur != nil {
		insert = false
		if cur.meta.Version > 0 {
			ver = cur.meta.Version + 1
		}
	}
	meta, err := s.writeTx(ctx, tx, ref, plaintext, expiresAt, ver, StatusPresent, insert)
	if err != nil {
		return PutResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return PutResult{}, err
	}
	return PutResult{Metadata: meta}, nil
}

// Revoke marks the ref revoked and wipes ciphertext/nonce/fingerprint.
func (s *SQL) Revoke(ctx context.Context, ref Ref) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if err := validateRef(ref); err != nil {
		return Metadata{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Metadata{}, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := s.loadTx(ctx, tx, ref)
	if err != nil {
		return Metadata{}, err
	}
	if cur == nil {
		return Metadata{}, ErrNotFound
	}
	now := s.now().UTC()
	meta := cur.meta
	meta.Status = StatusRevoked
	meta.Fingerprint = ""
	q := fmt.Sprintf(
		`UPDATE %s SET status=%s, fingerprint=%s, nonce=%s, ciphertext=%s, updated_at=%s WHERE ref_key=%s`,
		s.t("secret_store_entries"), s.p(1), s.p(2), s.p(3), s.p(4), s.p(5), s.p(6),
	)
	if _, err := tx.ExecContext(ctx, q, StatusRevoked, "", []byte{}, []byte{}, s.timeArg(now), ref.Key()); err != nil {
		return Metadata{}, err
	}
	if err := tx.Commit(); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

// Metadata returns sanitized fields only (never plaintext).
func (s *SQL) Metadata(ctx context.Context, ref Ref) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if err := validateRef(ref); err != nil {
		return Metadata{}, err
	}
	cur, err := s.loadTx(ctx, s.db, ref)
	if err != nil {
		return Metadata{}, err
	}
	if cur == nil {
		return Metadata{Ref: ref, Status: StatusAbsent, Environment: ref.Environment}, nil
	}
	return cur.meta, nil
}

// ListMetadata returns sanitized metadata for one environment (never plaintext/ciphertext).
func (s *SQL) ListMetadata(ctx context.Context, environment string) ([]Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	env := strings.TrimSpace(environment)
	switch env {
	case EnvHomologation, EnvProduction:
	default:
		return nil, fmt.Errorf("%w: environment deve ser homologation|production", ErrValidation)
	}
	q := fmt.Sprintf(
		`SELECT kind, environment, subject_id, name, status, fingerprint, version, expires_at, last_verified_at
		 FROM %s WHERE environment=%s`,
		s.t("secret_store_entries"), s.p(1),
	)
	rows, err := s.db.QueryContext(ctx, q, env)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Metadata, 0)
	for rows.Next() {
		var (
			kind, e, subject, name, status, fp string
			version                            int
			expiresRaw, verifiedRaw            any
		)
		if err := rows.Scan(&kind, &e, &subject, &name, &status, &fp, &version, &expiresRaw, &verifiedRaw); err != nil {
			return nil, err
		}
		meta := Metadata{
			Ref:         Ref{Kind: kind, Environment: e, SubjectID: subject, Name: name},
			Status:      status,
			Fingerprint: fp,
			Version:     version,
			Environment: e,
		}
		if t, ok, err := parseNullableTime(expiresRaw); err != nil {
			return nil, err
		} else if ok {
			meta.ExpiresAt = &t
		}
		if t, ok, err := parseNullableTime(verifiedRaw); err != nil {
			return nil, err
		} else if ok {
			meta.LastVerifiedAt = &t
		}
		out = append(out, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortMetadata(out)
	return out, nil
}

// Reveal returns plaintext for runtime only. Admin UI must not call this.
func (s *SQL) Reveal(ctx context.Context, ref Ref) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRef(ref); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := s.loadTx(ctx, tx, ref)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrNotFound
	}
	if cur.meta.Status == StatusRevoked || len(cur.ciphertext) == 0 {
		return nil, ErrRevoked
	}
	plain, err := s.aead.Open(nil, cur.nonce, cur.ciphertext, []byte(ref.Key()))
	if err != nil {
		return nil, fmt.Errorf("%w: decrypt", ErrValidation)
	}
	now := s.now().UTC()
	q := fmt.Sprintf(
		`UPDATE %s SET last_verified_at=%s, updated_at=%s WHERE ref_key=%s`,
		s.t("secret_store_entries"), s.p(1), s.p(2), s.p(3),
	)
	if _, err := tx.ExecContext(ctx, q, s.timeArg(now), s.timeArg(now), ref.Key()); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	out := make([]byte, len(plain))
	copy(out, plain)
	zeroBytes(plain)
	return out, nil
}

// CopyAcrossEnvironments is rejected (HML≠PRD).
func (s *SQL) CopyAcrossEnvironments(_ context.Context, from, to Ref) error {
	if err := validateRef(from); err != nil {
		return err
	}
	if err := validateRef(to); err != nil {
		return err
	}
	if from.Environment == to.Environment {
		return fmt.Errorf("%w: ambientes iguais", ErrValidation)
	}
	return fmt.Errorf("%w: cópia %s→%s proibida", ErrEnvIsolation, from.Environment, to.Environment)
}

type sqlEntry struct {
	nonce      []byte
	ciphertext []byte
	meta       Metadata
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *SQL) loadTx(ctx context.Context, q querier, ref Ref) (*sqlEntry, error) {
	query := fmt.Sprintf(
		`SELECT kind, environment, subject_id, name, status, fingerprint, nonce, ciphertext, version, expires_at, last_verified_at
		 FROM %s WHERE ref_key=%s`,
		s.t("secret_store_entries"), s.p(1),
	)
	var (
		kind, env, subject, name, status, fp string
		nonce, ct                            []byte
		version                              int
		expiresRaw, verifiedRaw              any
	)
	err := q.QueryRowContext(ctx, query, ref.Key()).Scan(
		&kind, &env, &subject, &name, &status, &fp, &nonce, &ct, &version, &expiresRaw, &verifiedRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	meta := Metadata{
		Ref: Ref{
			Kind: kind, Environment: env, SubjectID: subject, Name: name,
		},
		Status: status, Fingerprint: fp, Version: version, Environment: env,
	}
	if t, ok, err := parseNullableTime(expiresRaw); err != nil {
		return nil, err
	} else if ok {
		meta.ExpiresAt = &t
	}
	if t, ok, err := parseNullableTime(verifiedRaw); err != nil {
		return nil, err
	} else if ok {
		meta.LastVerifiedAt = &t
	}
	return &sqlEntry{nonce: nonce, ciphertext: ct, meta: meta}, nil
}

func (s *SQL) writeTx(
	ctx context.Context,
	tx *sql.Tx,
	ref Ref,
	plaintext []byte,
	expiresAt *time.Time,
	version int,
	status string,
	insert bool,
) (Metadata, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return Metadata{}, err
	}
	ct := s.aead.Seal(nil, nonce, plaintext, []byte(ref.Key()))
	sum := sha256.Sum256(plaintext)
	fp := hex.EncodeToString(sum[:])
	now := s.now().UTC()
	meta := Metadata{
		Ref:         ref,
		Status:      status,
		Fingerprint: fp,
		ExpiresAt:   expiresAt,
		Version:     version,
		Environment: ref.Environment,
	}
	var expiresArg any
	if expiresAt != nil {
		expiresArg = s.timeArg(expiresAt.UTC())
	}
	if insert {
		q := fmt.Sprintf(
			`INSERT INTO %s (
				ref_key, kind, environment, subject_id, name, status, fingerprint, cipher_alg,
				nonce, ciphertext, version, expires_at, last_verified_at, created_at, updated_at
			) VALUES (%s)`,
			s.t("secret_store_entries"), s.ph(15),
		)
		_, err := tx.ExecContext(ctx, q,
			ref.Key(), ref.Kind, ref.Environment, ref.SubjectID, ref.Name,
			status, fp, CipherAlgAES256GCM, nonce, ct, version,
			expiresArg, nil, s.timeArg(now), s.timeArg(now),
		)
		if err != nil {
			return Metadata{}, err
		}
		return meta, nil
	}
	q := fmt.Sprintf(
		`UPDATE %s SET status=%s, fingerprint=%s, cipher_alg=%s, nonce=%s, ciphertext=%s,
			version=%s, expires_at=%s, last_verified_at=%s, updated_at=%s
		 WHERE ref_key=%s`,
		s.t("secret_store_entries"),
		s.p(1), s.p(2), s.p(3), s.p(4), s.p(5), s.p(6), s.p(7), s.p(8), s.p(9), s.p(10),
	)
	_, err := tx.ExecContext(ctx, q,
		status, fp, CipherAlgAES256GCM, nonce, ct, version,
		expiresArg, nil, s.timeArg(now), ref.Key(),
	)
	if err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func validatePlaintext(ref Ref, plaintext []byte) error {
	if len(plaintext) == 0 {
		return fmt.Errorf("%w: plaintext vazio", ErrValidation)
	}
	if len(plaintext) > MaxBytesForKind(ref.Kind) {
		return fmt.Errorf("%w: plaintext demasiado grande", ErrValidation)
	}
	return nil
}

func (s *SQL) t(name string) string {
	if s.dialect == DialectPostgres {
		return "fiscal." + name
	}
	return name
}

func (s *SQL) ph(n int) string {
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

func (s *SQL) p(n int) string {
	if s.dialect == DialectPostgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s *SQL) timeArg(t time.Time) any {
	if s.dialect == DialectPostgres {
		return t
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

func parseNullableTime(v any) (time.Time, bool, error) {
	if v == nil {
		return time.Time{}, false, nil
	}
	switch x := v.(type) {
	case time.Time:
		return x.UTC(), true, nil
	case string:
		if strings.TrimSpace(x) == "" {
			return time.Time{}, false, nil
		}
		t, err := time.Parse(time.RFC3339Nano, x)
		return t.UTC(), err == nil, err
	case []byte:
		if len(x) == 0 {
			return time.Time{}, false, nil
		}
		t, err := time.Parse(time.RFC3339Nano, string(x))
		return t.UTC(), err == nil, err
	default:
		return time.Time{}, false, fmt.Errorf("%w: timestamp tipo", ErrValidation)
	}
}

var (
	_ AdminView     = (*SQL)(nil)
	_ RuntimeReveal = (*SQL)(nil)
	_ Vault         = (*SQL)(nil)
)
