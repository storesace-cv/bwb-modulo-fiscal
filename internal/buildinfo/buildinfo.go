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
	// DevRevision is allowed only for non-release / local development builds.
	DevRevision = "dev"
	// UnknownRevision is an explicit invalid sentinel (must never ship).
	UnknownRevision = "unknown"
)

var sha40Lower = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Validate checks revision is either "dev" or a lowercase SHA-40.
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

// ValidateRelease requires a lowercase SHA-40 that equals commit (also lowercase SHA-40).
func ValidateRelease(revision, commit string) error {
	if err := Validate(revision); err != nil {
		return err
	}
	if revision == DevRevision {
		return fmt.Errorf("buildinfo: release revision must not be %q", DevRevision)
	}
	c := strings.TrimSpace(strings.ToLower(commit))
	if !sha40Lower.MatchString(c) {
		return fmt.Errorf("buildinfo: commit %q is not lowercase sha40", commit)
	}
	if revision != c {
		return fmt.Errorf("buildinfo: revision does not match commit")
	}
	return nil
}

// MustRevision returns Revision after Validate, or panics (startup fail-closed).
func MustRevision() string {
	if err := Validate(Revision); err != nil {
		panic(err)
	}
	return Revision
}
