// Package buildinfo holds immutable build identity injected at link time.
package buildinfo

import (
	"fmt"
	"regexp"
	"strings"
)

// Revision is the artefact identity.
// Development builds default to "dev". Release builds must inject a lowercase SHA-40 via:
//
//	-ldflags "-X github.com/storesace-cv/bwb-modulo-fiscal/internal/buildinfo.Revision=<sha40>"
var Revision = "dev"

const (
	// DevRevision is allowed only when FISCAL_ENV=development.
	DevRevision = "dev"
	// UnknownRevision is an explicit invalid sentinel (must never ship).
	UnknownRevision = "unknown"
	envDevelopment  = "development"
)

var sha40Lower = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Validate checks revision is either "dev" or a lowercase SHA-40 (format only).
func Validate(revision string) error {
	r := strings.TrimSpace(revision)
	if r == "" {
		return fmt.Errorf("buildinfo: revision is empty")
	}
	if r == UnknownRevision {
		return fmt.Errorf("buildinfo: revision is unknown")
	}
	if r == DevRevision {
		return nil
	}
	if !sha40Lower.MatchString(r) {
		return fmt.Errorf("buildinfo: revision %q is not lowercase sha40 or %q", r, DevRevision)
	}
	return nil
}

// ValidateForEnv applies environment policy: revision=dev only in development;
// homologation/production require lowercase SHA-40.
func ValidateForEnv(revision, fiscalEnv string) error {
	r := strings.TrimSpace(revision)
	if err := Validate(r); err != nil {
		return err
	}
	env := strings.TrimSpace(fiscalEnv)
	if r == DevRevision {
		if env != envDevelopment {
			return fmt.Errorf("buildinfo: revision %q requires FISCAL_ENV=%q", DevRevision, envDevelopment)
		}
		return nil
	}
	if !sha40Lower.MatchString(r) {
		return fmt.Errorf("buildinfo: revision must be lowercase sha40 outside development")
	}
	return nil
}

// ValidateRelease requires lowercase SHA-40 revision equal to commit (exact lowercase match; no case folding).
func ValidateRelease(revision, commit string) error {
	r := strings.TrimSpace(revision)
	c := strings.TrimSpace(commit)
	if err := Validate(r); err != nil {
		return err
	}
	if r == DevRevision {
		return fmt.Errorf("buildinfo: release revision must not be %q", DevRevision)
	}
	if !sha40Lower.MatchString(r) {
		return fmt.Errorf("buildinfo: revision %q is not lowercase sha40", r)
	}
	if !sha40Lower.MatchString(c) {
		return fmt.Errorf("buildinfo: commit %q is not lowercase sha40", c)
	}
	if r != c {
		return fmt.Errorf("buildinfo: revision does not match commit")
	}
	return nil
}
