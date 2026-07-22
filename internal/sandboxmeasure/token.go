package sandboxmeasure

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syscall"
)

func readTokenFile(path string, production bool, skipOwner bool) ([]byte, error) {
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("sandboxmeasure: token path invalid")
	}
	if production {
		if err := validateProductionTokenTree(path, skipOwner); err != nil {
			return nil, err
		}
	} else {
		st, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("sandboxmeasure: token unavailable")
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return nil, fmt.Errorf("sandboxmeasure: token must be a regular file")
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	tok := bytes.TrimSpace(raw)
	if len(tok) == 0 {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	return tok, nil
}

func validateProductionTokenTree(path string, skipOwner bool) error {
	if path != FixedTokenFile {
		return fmt.Errorf("sandboxmeasure: token path invalid")
	}
	return validateTokenTree(FixedTokenDir, path, skipOwner)
}

func validateTokenTree(tokenDir, tokenPath string, skipOwner bool) error {
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
	fileInfo, err := os.Lstat(tokenPath)
	if err != nil {
		return fmt.Errorf("sandboxmeasure: token unavailable")
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("sandboxmeasure: token must be a regular file")
	}
	if fileInfo.Mode().Perm() != 0o600 {
		return fmt.Errorf("sandboxmeasure: token permissions invalid")
	}
	if !skipOwner {
		if err := assertOwnedByEUID(dirInfo); err != nil {
			return err
		}
		if err := assertOwnedByEUID(fileInfo); err != nil {
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

// ValidateTokenFileForTest exercises production checks against an arbitrary path tree (tests only).
func ValidateTokenFileForTest(tokenDir, tokenPath string, skipOwner bool) error {
	return validateTokenTree(tokenDir, tokenPath, skipOwner)
}
