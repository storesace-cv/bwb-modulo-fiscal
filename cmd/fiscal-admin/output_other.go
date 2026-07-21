//go:build !unix

package main

import (
	"fmt"
	"os"
)

func openExclusiveFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("cannot create output file (exists or permission): %w", err)
	}
	return f, nil
}
