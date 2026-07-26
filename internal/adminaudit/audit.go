// Package adminaudit persists append-only admin mutation events (DEC-BO-002).
// Never stores secrets.
package adminaudit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

// Dialect selects SQL dialect.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

const (
	ResultSuccess = "success"
	ResultDenied  = "denied"
	ResultError   = "error"
)

// Event is one append-only admin audit row.
type Event struct {
	ID           string
	OccurredAt   time.Time
	ActorSubject string
	ActorRoles   string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	RequestID    string
}

// Store writes admin audit events.
type Store struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

// New returns a Store.
func New(db *sql.DB, dialect Dialect, now func() time.Time) *Store {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Store{db: db, dialect: dialect, now: now}
}

// Record appends an audit event. Fail-closed on empty required fields.
func (s *Store) Record(ctx context.Context, claims adminauth.Claims, action, resourceType, resourceID, result, requestID string) error {
	action = strings.TrimSpace(action)
	resourceType = strings.TrimSpace(resourceType)
	resourceID = strings.TrimSpace(resourceID)
	result = strings.TrimSpace(result)
	if claims.Subject == "" || action == "" || resourceType == "" || resourceID == "" {
		return fmt.Errorf("adminaudit: campos obrigatórios")
	}
	switch result {
	case ResultSuccess, ResultDenied, ResultError:
	default:
		return fmt.Errorf("adminaudit: result inválido")
	}
	roles := make([]string, 0, len(claims.Roles))
	for _, r := range claims.Roles {
		roles = append(roles, string(r))
	}
	id, err := newID()
	if err != nil {
		return err
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	q := `INSERT INTO ` + s.t("admin_audit_events") + ` (
		event_id, occurred_at, actor_subject, actor_roles, action, resource_type, resource_id, result, request_id
	) VALUES (` + s.ph(9) + `)`
	_, err = s.db.ExecContext(ctx, q,
		id, s.timeArg(now), claims.Subject, strings.Join(roles, ","),
		action, resourceType, resourceID, result, nullIfEmpty(requestID),
	)
	if err != nil {
		return fmt.Errorf("adminaudit: insert: %w", err)
	}
	return nil
}

// CountForTests returns row count (tests only).
func (s *Store) CountForTests(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+s.t("admin_audit_events")).Scan(&n)
	return n, err
}

func (s *Store) t(name string) string {
	if s.dialect == DialectPostgres {
		return "fiscal." + name
	}
	return name
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

func (s *Store) timeArg(t time.Time) any {
	if s.dialect == DialectPostgres {
		return t
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
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
