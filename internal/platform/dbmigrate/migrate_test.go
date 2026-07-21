package dbmigrate_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestMigrationVersionParity(t *testing.T) {
	root := repoRoot(t)
	pg, err := filepath.Glob(filepath.Join(root, "migrations", "postgres", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sq, err := filepath.Glob(filepath.Join(root, "migrations", "sqlite", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	pgVers := versions(pg)
	sqVers := versions(sq)
	if len(pgVers) == 0 || len(sqVers) == 0 {
		t.Fatal("expected migrations in both dialects")
	}
	if len(pgVers) != len(sqVers) {
		t.Fatalf("version count mismatch postgres=%v sqlite=%v", pgVers, sqVers)
	}
	for i := range pgVers {
		if pgVers[i] != sqVers[i] {
			t.Fatalf("version mismatch at %d: postgres=%s sqlite=%s", i, pgVers[i], sqVers[i])
		}
	}
	if pgVers[len(pgVers)-1] != "0002" {
		t.Fatalf("latest version = %s, want 0002", pgVers[len(pgVers)-1])
	}
	if dbmigrate.ExpectedVersion != 2 {
		t.Fatalf("ExpectedVersion = %d", dbmigrate.ExpectedVersion)
	}
}

func TestFiscalMigrateCLIOnlyUpVersion(t *testing.T) {
	root := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "fiscal-migrate")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/fiscal-migrate")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	path := filepath.Join(t.TempDir(), "cli.db")
	env := append(os.Environ(),
		"FISCAL_DATABASE_DRIVER=sqlite",
		"FISCAL_DATABASE_URL="+path,
	)

	up := exec.Command(bin, "up")
	up.Env = env
	if out, err := up.CombinedOutput(); err != nil {
		t.Fatalf("up: %v\n%s", err, out)
	}

	ver := exec.Command(bin, "version")
	ver.Env = env
	out, err := ver.CombinedOutput()
	if err != nil {
		t.Fatalf("version: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "version=2") {
		t.Fatalf("unexpected version output: %s", out)
	}

	for _, bad := range []string{"down", "drop", "force"} {
		c := exec.Command(bin, bad)
		c.Env = env
		out, err := c.CombinedOutput()
		if err == nil {
			t.Fatalf("command %q should fail", bad)
		}
		if !strings.Contains(string(out), "only up and version") {
			t.Fatalf("command %q output: %s", bad, out)
		}
	}
}

func versions(files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		base := filepath.Base(f)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) > 0 {
			out = append(out, parts[0])
		}
	}
	return out
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/persistence or internal/platform/dbmigrate -> repo root
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		wd = filepath.Dir(wd)
	}
	t.Fatal("go.mod not found")
	return ""
}
