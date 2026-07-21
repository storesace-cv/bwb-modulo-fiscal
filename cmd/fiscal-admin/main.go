// Command fiscal-admin gere scopes e credenciais sandbox (sem HTTP).
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"golang.org/x/term"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: fiscal-admin <scope|credential> ...")
		return 2
	}
	ctx := context.Background()
	switch args[0] {
	case "scope":
		return runScope(ctx, args[1:])
	case "credential":
		return runCredential(ctx, args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: fiscal-admin <scope|credential> ...")
		return 2
	}
}

func openStore(ctx context.Context) (*persistence.CredentialStore, func(), error) {
	driver := strings.TrimSpace(os.Getenv("FISCAL_DATABASE_DRIVER"))
	url := strings.TrimSpace(os.Getenv("FISCAL_DATABASE_URL"))
	if driver == "" || url == "" {
		return nil, nil, errors.New("FISCAL_DATABASE_DRIVER and FISCAL_DATABASE_URL are required")
	}
	var (
		sqlDB *sql.DB
		err   error
		dia   persistence.Dialect
	)
	switch driver {
	case db.DriverPostgres:
		sqlDB, err = db.OpenPostgres(ctx, db.PostgresConfig{URL: url})
		dia = persistence.DialectPostgres
	case db.DriverSQLite:
		sqlDB, err = db.OpenSQLite(ctx, db.SQLiteConfig{Path: url})
		dia = persistence.DialectSQLite
	default:
		return nil, nil, fmt.Errorf("unsupported driver %q", driver)
	}
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = sqlDB.Close() }
	return persistence.NewCredentialStore(sqlDB, dia), cleanup, nil
}

func runScope(ctx context.Context, args []string) int {
	if len(args) < 1 || args[0] != "create" {
		fmt.Fprintln(os.Stderr, "usage: fiscal-admin scope create --scope-id --taxpayer-nif --timezone --series --environment")
		return 2
	}
	fs := flag.NewFlagSet("scope create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scopeID := fs.String("scope-id", "", "")
	nif := fs.String("taxpayer-nif", "", "")
	tz := fs.String("timezone", "", "")
	series := fs.String("series", "", "")
	env := fs.String("environment", "", "")
	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "invalid flags")
		return 2
	}
	if strings.TrimSpace(*scopeID) == "" || strings.TrimSpace(*nif) == "" ||
		strings.TrimSpace(*tz) == "" || strings.TrimSpace(*series) == "" || strings.TrimSpace(*env) == "" {
		fmt.Fprintln(os.Stderr, "missing required flags")
		return 2
	}
	store, closeStore, err := openStore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error")
		return 1
	}
	defer closeStore()
	rec, err := store.CreateScope(ctx, persistence.CreateScopeParams{
		ScopeID:             *scopeID,
		TaxpayerNIF:         *nif,
		IANATimezone:        *tz,
		SeriesEffectiveCode: *series,
		Environment:         *env,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope create failed")
		return 1
	}
	fmt.Printf("scope_id=%s environment=%s status=%s\n", rec.ScopeID, rec.Environment, rec.Status)
	return 0
}

func runCredential(ctx context.Context, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: fiscal-admin credential <issue|rotate|revoke>")
		return 2
	}
	switch args[0] {
	case "issue":
		return runIssue(ctx, args[1:])
	case "rotate":
		return runRotate(ctx, args[1:])
	case "revoke":
		return runRevoke(ctx, args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: fiscal-admin credential <issue|rotate|revoke>")
		return 2
	}
}

func runIssue(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("credential issue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scopeID := fs.String("scope-id", "", "")
	createdBy := fs.String("created-by", "", "")
	expiresAt := fs.String("expires-at", "", "")
	outFile := fs.String("output-file", "", "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "invalid flags")
		return 2
	}
	if strings.TrimSpace(*scopeID) == "" || strings.TrimSpace(*createdBy) == "" {
		fmt.Fprintln(os.Stderr, "missing required flags")
		return 2
	}
	var exp *time.Time
	if strings.TrimSpace(*expiresAt) != "" {
		t, err := time.Parse(time.RFC3339, *expiresAt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid expires-at")
			return 2
		}
		t = t.UTC()
		exp = &t
	}

	store, closeStore, err := openStore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error")
		return 1
	}
	defer closeStore()

	sink, closer, err := openTokenSink(*outFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if closer != nil {
		defer closer()
	}

	rec, err := store.Issue(ctx, persistence.IssueParams{
		ScopeID:   *scopeID,
		CreatedBy: *createdBy,
		ExpiresAt: exp,
		RequestID: newAdminRequestID(),
		Deliver:   sink,
	})
	if err != nil {
		if strings.TrimSpace(*outFile) != "" {
			fmt.Fprintln(os.Stderr, "credential issue failed; if an output file was created, delete it manually")
		} else {
			fmt.Fprintln(os.Stderr, "credential issue failed")
		}
		return 1
	}
	fmt.Printf("credential_id=%s scope_id=%s status=%s\n", rec.CredentialID, rec.ScopeID, rec.Status)
	return 0
}

func runRotate(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("credential rotate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scopeID := fs.String("scope-id", "", "")
	createdBy := fs.String("created-by", "", "")
	graceUntil := fs.String("grace-until", "", "")
	expiresAt := fs.String("expires-at", "", "")
	outFile := fs.String("output-file", "", "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "invalid flags")
		return 2
	}
	if strings.TrimSpace(*scopeID) == "" || strings.TrimSpace(*createdBy) == "" || strings.TrimSpace(*graceUntil) == "" {
		fmt.Fprintln(os.Stderr, "missing required flags")
		return 2
	}
	gu, err := time.Parse(time.RFC3339, *graceUntil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid grace-until")
		return 2
	}
	var exp *time.Time
	if strings.TrimSpace(*expiresAt) != "" {
		t, err := time.Parse(time.RFC3339, *expiresAt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid expires-at")
			return 2
		}
		t = t.UTC()
		exp = &t
	}

	store, closeStore, err := openStore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error")
		return 1
	}
	defer closeStore()

	sink, closer, err := openTokenSink(*outFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if closer != nil {
		defer closer()
	}

	out, err := store.Rotate(ctx, persistence.RotateParams{
		ScopeID:    *scopeID,
		CreatedBy:  *createdBy,
		GraceUntil: gu.UTC(),
		ExpiresAt:  exp,
		RequestID:  newAdminRequestID(),
		Deliver:    sink,
	})
	if err != nil {
		if strings.TrimSpace(*outFile) != "" {
			fmt.Fprintln(os.Stderr, "credential rotate failed; if an output file was created, delete it manually")
		} else {
			fmt.Fprintln(os.Stderr, "credential rotate failed")
		}
		return 1
	}
	fmt.Printf("credential_id=%s previous_id=%s scope_id=%s status=%s\n",
		out.Credential.CredentialID, out.PreviousID, out.Credential.ScopeID, out.Credential.Status)
	return 0
}

func runRevoke(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("credential revoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scopeID := fs.String("scope-id", "", "")
	credID := fs.String("credential-id", "", "")
	reason := fs.String("reason-code", "", "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "invalid flags")
		return 2
	}
	if strings.TrimSpace(*scopeID) == "" || strings.TrimSpace(*credID) == "" {
		fmt.Fprintln(os.Stderr, "missing required flags")
		return 2
	}
	store, closeStore, err := openStore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error")
		return 1
	}
	defer closeStore()
	rec, err := store.Revoke(ctx, persistence.RevokeParams{
		ScopeID:      *scopeID,
		CredentialID: *credID,
		ReasonCode:   *reason,
		RequestID:    newAdminRequestID(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "credential revoke failed")
		return 1
	}
	fmt.Printf("credential_id=%s scope_id=%s status=%s\n", rec.CredentialID, rec.ScopeID, rec.Status)
	return 0
}

type syncWriter interface {
	io.Writer
	Sync() error
}

func writeAndSyncToken(w syncWriter, token string) error {
	if _, err := io.WriteString(w, token); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return w.Sync()
}

func openTokenSink(outPath string) (persistence.TokenSink, func(), error) {
	outPath = strings.TrimSpace(outPath)
	if outPath == "" {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return nil, nil, errors.New("refusing to write token to non-TTY stdout; use --output-file")
		}
		return func(token string) error {
			_, err := fmt.Fprintln(os.Stdout, token)
			return err
		}, nil, nil
	}
	f, err := openExclusiveFile(outPath)
	if err != nil {
		return nil, nil, err
	}
	return func(token string) error {
		return writeAndSyncToken(f, token)
	}, func() { _ = f.Close() }, nil
}

func newAdminRequestID() string {
	return fmt.Sprintf("admin_%d", time.Now().UnixNano())
}
