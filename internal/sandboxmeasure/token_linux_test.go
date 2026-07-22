//go:build linux

package sandboxmeasure_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/sandboxmeasure"
)

// TestTokenSymlinkRejectedByONoFollow proves Linux open(O_NOFOLLOW) rejects a path that is a symlink
// (including replacement of a previously regular token file).
func TestTokenSymlinkRejectedByONoFollow(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(dir, "real.token")
	if err := os.WriteFile(real, []byte("secret-token-value"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "measure.token")
	if err := os.WriteFile(path, []byte("secret-token-value"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Replace regular file with symlink (TOCTOU-style presentation).
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, path); err != nil {
		t.Fatal(err)
	}

	fix := filepath.Join(dir, "create-document.min.json")
	if err := os.WriteFile(fix, []byte(`{"external_id":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          path,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("must not call HTTP after symlink token")
			return nil, nil
		})},
	}
	_, err := sandboxmeasure.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected O_NOFOLLOW rejection")
	}
	msg := err.Error()
	if strings.Contains(msg, "secret-token-value") || strings.Contains(msg, path) || strings.Contains(msg, dir) {
		t.Fatalf("sensitive data in error: %v", err)
	}
	if strings.Contains(strings.ToLower(msg), "bearer") {
		t.Fatalf("auth leaked: %v", err)
	}
}
