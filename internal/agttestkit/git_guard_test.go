package agttestkit_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func TestNoTrackedXLSXWorkbooks(t *testing.T) {
	root := repoRoot(t)
	out, err := exec.Command("git", "-C", root, "ls-files", "-z", "--", "*.xlsx").Output()
	if err != nil {
		t.Fatal(err)
	}
	var tracked []string
	for _, rel := range bytes.Split(out, []byte{0}) {
		if len(rel) == 0 {
			continue
		}
		tracked = append(tracked, string(rel))
	}
	if len(tracked) != 0 {
		t.Fatalf("unexpected tracked xlsx (none allowed in this repo): %v", tracked)
	}
}

func TestPackageSourcesHaveNoHardcodedWorkbookPaths(t *testing.T) {
	root := repoRoot(t)
	pkg := filepath.Join(root, "internal", "agttestkit")
	entries, err := os.ReadDir(pkg)
	if err != nil {
		t.Fatal(err)
	}
	xlsxName := regexp.MustCompile(`(?i)[A-Za-z0-9_.-]+\.xlsx`)
	// Build markers without embedding contiguous absolute-path literals in this file.
	unixUsers := "/" + "Users" + "/"
	unixHome := "/" + "home" + "/"
	winDrive := regexp.MustCompile(`(?i)[A-Za-z]:\\`)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(pkg, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if strings.Contains(s, unixUsers) || strings.Contains(s, unixHome) || winDrive.Match(b) {
			t.Fatalf("%s: absolute filesystem path literal forbidden", e.Name())
		}
		for _, m := range xlsxName.FindAll(b, -1) {
			name := string(m)
			if name == "synthetic-agt-test-identities.xlsx" || name == "custom.xlsx" {
				continue
			}
			t.Fatalf("%s: unexpected xlsx name literal %q (CI must use synthetic temps only)", e.Name(), name)
		}
	}
}

func TestSyntheticWorkbookLivesUnderTempDir(t *testing.T) {
	dir := t.TempDir()
	path, cleanup, err := agttestkit.WriteSyntheticWorkbook(dir, agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("synthetic workbook must live under temp dir")
	}
	if _, err := agttestkit.LoadAndValidate(path); err != nil {
		t.Fatal(err)
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
