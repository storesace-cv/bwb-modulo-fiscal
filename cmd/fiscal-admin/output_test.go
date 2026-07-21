package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failSyncWriter struct {
	buf      bytes.Buffer
	syncErr  error
	writeErr error
}

func (f *failSyncWriter) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.buf.Write(p)
}

func (f *failSyncWriter) Sync() error {
	return f.syncErr
}

func TestWriteAndSyncTokenPartialSyncKeepsBytes(t *testing.T) {
	w := &failSyncWriter{syncErr: errors.New("sync failed")}
	token := "bwb_sbox_" + strings.Repeat("Z", 43)
	err := writeAndSyncToken(w, token)
	if err == nil {
		t.Fatal("expected sync failure")
	}
	got := w.buf.String()
	if !strings.Contains(got, token) {
		t.Fatal("partial write must retain token bytes in writer buffer")
	}
}

func TestOpenExclusiveFileRejectsExistingAndSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.out")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openExclusiveFile(path); err == nil {
		t.Fatal("expected existing file to fail")
	}
	link := filepath.Join(dir, "link.out")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := openExclusiveFile(link); err == nil {
		t.Fatal("expected symlink to fail")
	}
	fresh := filepath.Join(dir, "fresh.out")
	f, err := openExclusiveFile(fresh)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm=%o", st.Mode().Perm())
	}
}

func TestIssueInvalidExpiresDoesNotCreateOutputFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "should-not-exist.out")
	code := runIssue(context.Background(), []string{
		"--scope-id", "s",
		"--created-by", "admin",
		"--expires-at", "not-a-date",
		"--output-file", out,
	})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("output file must not be created on invalid expires-at")
	}
}

func TestIssueMissingFlagsDoesNotCreateOutputFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "should-not-exist.out")
	code := runIssue(context.Background(), []string{
		"--created-by", "admin",
		"--output-file", out,
	})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("output file must not be created on missing flags")
	}
}

func TestIssueDatabaseErrorDoesNotCreateOutputFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "should-not-exist.out")
	t.Setenv("FISCAL_DATABASE_DRIVER", "sqlite")
	t.Setenv("FISCAL_DATABASE_URL", filepath.Join(dir, "missing-dir", "nope.db"))
	var stderr bytes.Buffer
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	code := runIssue(context.Background(), []string{
		"--scope-id", "s",
		"--created-by", "admin",
		"--output-file", out,
	})
	_ = w.Close()
	os.Stderr = old
	_, _ = io.Copy(&stderr, r)
	if code != 1 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("output file must not be created when database open fails")
	}
	if strings.Contains(stderr.String(), "bwb_sbox_") {
		t.Fatal("stderr must not contain token prefix")
	}
}

func TestSyncFailureLeavesFileWithoutUnlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.out")
	f, err := openExclusiveFile(path)
	if err != nil {
		t.Fatal(err)
	}
	token := "bwb_sbox_" + strings.Repeat("Y", 43)
	// Write then force sync failure via helper on a wrapper after closing real fd path check.
	_ = f.Close()
	// Recreate exclusive semantics already satisfied; reopen for write of partial content + assert no unlink policy.
	// Simulate Deliver path: exclusive create, writeAndSyncToken fails Sync, file remains.
	f2, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := &failAfterWriteFile{File: f2, failSync: true}
	if err := writeAndSyncToken(wrapped, token); err == nil {
		t.Fatal("expected sync failure")
	}
	_ = f2.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed sync must not remove output path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(token)) {
		t.Fatal("partial content should remain after sync failure")
	}
}

type failAfterWriteFile struct {
	*os.File
	failSync bool
}

func (f *failAfterWriteFile) Sync() error {
	if f.failSync {
		return errors.New("sync failed")
	}
	return f.File.Sync()
}
