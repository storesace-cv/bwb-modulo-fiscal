// Package money representa valores monetários AOA em centavos (int64), sem floating-point.
package money

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	// Scale is the number of decimal places for AOA MVP money (OpenAPI Money).
	Scale = 2
	// Factor converts major units to cents (10^Scale).
	Factor int64 = 100
)

var (
	canonicalPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})\.[0-9]{2}$`)
	errInvalid       = errors.New("money: invalid canonical form")
	errOverflow      = errors.New("money: overflow")
)

// Amount is a non-negative monetary value in cents.
type Amount struct {
	cents int64
}

// FromCents builds an Amount from cents. Rejects negative values.
func FromCents(cents int64) (Amount, error) {
	if cents < 0 {
		return Amount{}, errInvalid
	}
	return Amount{cents: cents}, nil
}

// Cents returns the fixed-point integer representation.
func (a Amount) Cents() int64 { return a.cents }

// ParseCanonical parses OpenAPI Money (scale 2, no sign).
func ParseCanonical(s string) (Amount, error) {
	if !canonicalPattern.MatchString(s) {
		return Amount{}, errInvalid
	}
	parts := strings.Split(s, ".")
	major, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Amount{}, errInvalid
	}
	frac, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Amount{}, errInvalid
	}
	if major > math.MaxInt64/Factor {
		return Amount{}, errOverflow
	}
	cents := major*Factor + frac
	if cents < 0 {
		return Amount{}, errOverflow
	}
	return Amount{cents: cents}, nil
}

// FormatCanonical returns the OpenAPI Money string.
func (a Amount) FormatCanonical() string {
	major := a.cents / Factor
	frac := a.cents % Factor
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%d.%02d", major, frac)
}
