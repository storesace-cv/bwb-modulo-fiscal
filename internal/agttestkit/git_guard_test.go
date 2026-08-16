package agttestkit_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
)

func TestContainsPrivatePEMBlockRequiresFullEnvelope(t *testing.T) {
	doc := []byte("Documentation may mention BEGIN PRIVATE KEY without a full block.")
	if agttestkit.ContainsPrivatePEMBlock(doc) {
		t.Fatal("documentary phrase must not match")
	}
	full := []byte("-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC7\n-----END PRIVATE KEY-----\n")
	if !agttestkit.ContainsPrivatePEMBlock(full) {
		t.Fatal("full block must match")
	}
}

func TestGitTrackedFilesHaveNoPrivatePEMBlocks(t *testing.T) {
	root := repoRoot(t)
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range bytes.Split(out, []byte{0}) {
		if len(rel) == 0 {
			continue
		}
		path := filepath.Join(root, string(rel))
		// Skip obviously binary / lockfiles by extension where needed.
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".xlsx", ".zip", ".gz":
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if agttestkit.ContainsPrivatePEMBlock(b) {
			t.Fatalf("tracked file contains private PEM block: %s", string(rel))
		}
	}
}

func TestRealWorkbookNotTrackedByGit(t *testing.T) {
	root := repoRoot(t)
	rel := "local/7cb4c654-0e77-4831-a826-7306aef08524.xlsx"
	out, err := exec.Command("git", "-C", root, "ls-files", "--", rel).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		t.Fatalf("real workbook must not be tracked: %s", string(out))
	}
	// Confirm ignore of local/
	st, err := exec.Command("git", "-C", root, "check-ignore", "-v", "local/").CombinedOutput()
	if err != nil {
		t.Fatalf("local/ should be ignored: %v %s", err, st)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
