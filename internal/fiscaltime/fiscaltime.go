// Package fiscaltime valida e normaliza issued_at face à timezone fiscal do scope (DEC-TIME-001).
package fiscaltime

import (
	"errors"
	"fmt"
	"strings"
	"time"

	_ "time/tzdata" // embed IANA TZDB for Edge/cloud parity
)

const (
	// AfricaLuanda is the Angola fiscal IANA timezone.
	AfricaLuanda = "Africa/Luanda"
	// AtlanticCapeVerde is documented for a future CV catalog only (not wired in runtime).
	AtlanticCapeVerde = "Atlantic/Cape_Verde"

	minOffsetMinutes = -840
	maxOffsetMinutes = 840

	// UTCMicroFormat is the canonical UTC issued_at / hash string (exactly 6 fractional digits).
	UTCMicroFormat = "2006-01-02T15:04:05.000000Z"
)

var (
	// ErrInvalidIssuedAt is returned for parse/offset/timezone failures.
	ErrInvalidIssuedAt = errors.New("fiscaltime: invalid issued_at")
)

// NormalizedIssued is the fiscal temporal context after validation.
type NormalizedIssued struct {
	InstantUTC        time.Time // truncated to microseconds, UTC
	UTCString         string    // UTCMicroFormat
	Timezone          string    // IANA
	OffsetMinutes     int
	LocalCivilRFC3339 string // reconstruction with original offset for tests/display
}

// NormalizeIssued parses raw RFC3339 with offset, validates against ianaTZ at that instant,
// and returns UTC truncated to microseconds plus fiscal context.
func NormalizeIssued(raw, ianaTZ string) (NormalizedIssued, error) {
	raw = strings.TrimSpace(raw)
	ianaTZ = strings.TrimSpace(ianaTZ)
	if raw == "" {
		return NormalizedIssued{}, fmt.Errorf("%w: empty", ErrInvalidIssuedAt)
	}
	if ianaTZ == "" {
		return NormalizedIssued{}, fmt.Errorf("%w: empty timezone", ErrInvalidIssuedAt)
	}
	loc, err := time.LoadLocation(ianaTZ)
	if err != nil {
		return NormalizedIssued{}, fmt.Errorf("%w: load timezone %q: %v", ErrInvalidIssuedAt, ianaTZ, err)
	}

	// Reject bare local datetime without offset (RFC3339 requires zone).
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return NormalizedIssued{}, fmt.Errorf("%w: parse: %v", ErrInvalidIssuedAt, err)
		}
	}

	// Offset from the parsed zone at that instant (seconds east of UTC).
	_, inputOffSec := t.Zone()
	inputOffMin := inputOffSec / 60

	// Expected offset from fiscal timezone at the same absolute instant.
	inFiscal := t.In(loc)
	_, fiscalOffSec := inFiscal.Zone()
	fiscalOffMin := fiscalOffSec / 60

	if inputOffMin != fiscalOffMin {
		return NormalizedIssued{}, fmt.Errorf("%w: offset %d incompatible with %s (want %d)", ErrInvalidIssuedAt, inputOffMin, ianaTZ, fiscalOffMin)
	}
	if fiscalOffMin < minOffsetMinutes || fiscalOffMin > maxOffsetMinutes {
		return NormalizedIssued{}, fmt.Errorf("%w: offset out of range", ErrInvalidIssuedAt)
	}

	utc := t.UTC().Truncate(time.Microsecond)
	localRebuilt := utc.In(loc)
	return NormalizedIssued{
		InstantUTC:        utc,
		UTCString:         utc.Format(UTCMicroFormat),
		Timezone:          ianaTZ,
		OffsetMinutes:     fiscalOffMin,
		LocalCivilRFC3339: localRebuilt.Format(time.RFC3339Nano),
	}, nil
}

// RebuildLocal formats the civil local time from UTC instant + IANA timezone.
func RebuildLocal(utc time.Time, ianaTZ string) (time.Time, error) {
	loc, err := time.LoadLocation(ianaTZ)
	if err != nil {
		return time.Time{}, err
	}
	return utc.UTC().Truncate(time.Microsecond).In(loc), nil
}

// ValidateNormalizedContext checks an already-normalized fiscal temporal triple
// (UTC micro issued_at + IANA timezone + offset minutes) before seal/hash.
// Fail-closed: unknown IANA zones, non-canonical UTC strings, and offsets that
// do not match the zone at that instant are rejected.
func ValidateNormalizedContext(utcMicro, ianaTZ string, offsetMinutes int) error {
	utcMicro = strings.TrimSpace(utcMicro)
	ianaTZ = strings.TrimSpace(ianaTZ)
	if utcMicro == "" {
		return fmt.Errorf("%w: empty issued_at", ErrInvalidIssuedAt)
	}
	if ianaTZ == "" {
		return fmt.Errorf("%w: empty timezone", ErrInvalidIssuedAt)
	}
	if offsetMinutes < minOffsetMinutes || offsetMinutes > maxOffsetMinutes {
		return fmt.Errorf("%w: offset out of range", ErrInvalidIssuedAt)
	}

	parsed, err := time.Parse(UTCMicroFormat, utcMicro)
	if err != nil {
		return fmt.Errorf("%w: issued_at must be UTC micro canonical (%s)", ErrInvalidIssuedAt, UTCMicroFormat)
	}
	utc := parsed.UTC().Truncate(time.Microsecond)
	if utc.Format(UTCMicroFormat) != utcMicro {
		return fmt.Errorf("%w: issued_at must be UTC micro canonical", ErrInvalidIssuedAt)
	}

	loc, err := time.LoadLocation(ianaTZ)
	if err != nil {
		return fmt.Errorf("%w: load timezone %q: %v", ErrInvalidIssuedAt, ianaTZ, err)
	}
	_, fiscalOffSec := utc.In(loc).Zone()
	fiscalOffMin := fiscalOffSec / 60
	if fiscalOffMin != offsetMinutes {
		return fmt.Errorf("%w: offset %d incompatible with %s at %s (want %d)",
			ErrInvalidIssuedAt, offsetMinutes, ianaTZ, utcMicro, fiscalOffMin)
	}
	return nil
}
