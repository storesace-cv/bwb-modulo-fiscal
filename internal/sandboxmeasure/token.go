package sandboxmeasure

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

func readTokenFile(path string, production bool) ([]byte, error) {
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("sandboxmeasure: token path invalid")
	}
	if production {
		if path != FixedTokenFile {
			return nil, fmt.Errorf("sandboxmeasure: token path invalid")
		}
		// Production always validates directory ownership — no skip seam.
		if err := validateTokenDir(FixedTokenDir, true); err != nil {
			return nil, err
		}
		return readTokenOpenFstat(path, true)
	}
	// Test / non-fixed path: same open+Fstat seam; owner required (temp files are euid-owned).
	return readTokenOpenFstat(path, true)
}

// readTokenOpenFstat opens the path with O_NOFOLLOW, validates the same FD via Fstat, then reads it.
func readTokenOpenFstat(path string, requireOwner bool) ([]byte, error) {
	f, err := openTokenNoFollow(path)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	defer f.Close()

	st, err := f.Stat() // Fstat on the open descriptor (not a path re-lookup).
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("sandboxmeasure: token must be a regular file")
	}
	if st.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("sandboxmeasure: token permissions invalid")
	}
	if requireOwner {
		if err := assertOwnedByEUID(st); err != nil {
			return nil, err
		}
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	tok := bytes.TrimSpace(raw)
	if len(tok) == 0 {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	return tok, nil
}

func validateTokenDir(tokenDir string, requireOwner bool) error {
	dirInfo, err := os.Lstat(tokenDir)
	if err != nil {
		return fmt.Errorf("sandboxmeasure: token unavailable")
	}
	if dirInfo.Mode()&os.ModeSymlink != 0 || !dirInfo.IsDir() {
		return fmt.Errorf("sandboxmeasure: token dir invalid")
	}
	if dirInfo.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("sandboxmeasure: token dir permissions invalid")
	}
	if requireOwner {
		if err := assertOwnedByEUID(dirInfo); err != nil {
			return err
		}
	}
	return nil
}

func assertOwnedByEUID(info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("sandboxmeasure: token ownership unavailable")
	}
	if int(st.Uid) != os.Geteuid() {
		return fmt.Errorf("sandboxmeasure: token ownership invalid")
	}
	return nil
}

// ValidateTokenFileForTest exercises directory policy + open/Fstat token checks (tests only).
// skipOwner applies only to the directory check; the file is always validated on the open FD
// with owner=euid (temp-tree files are owned by the test process).
func ValidateTokenFileForTest(tokenDir, tokenPath string, skipOwner bool) error {
	if err := validateTokenDir(tokenDir, !skipOwner); err != nil {
		return err
	}
	_, err := readTokenOpenFstat(tokenPath, true)
	return err
}
