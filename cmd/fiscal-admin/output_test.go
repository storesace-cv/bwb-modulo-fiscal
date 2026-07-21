package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
