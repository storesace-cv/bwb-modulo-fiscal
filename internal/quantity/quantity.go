// Package quantity representa quantidades em unidades de 1/10000 (int64), sem floating-point.
package quantity

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	// Scale is the fixed decimal places for quantity persistence.
	Scale = 4
	// Factor converts major units to scaled integers (10^Scale).
	Factor int64 = 10000
)

var (
	// Matches OpenAPI DecimalQuantity (strictly positive, canonical).
	canonicalPattern = regexp.MustCompile(`^(?:0\.[0-9]{0,3}[1-9]|[1-9][0-9]{0,11}(?:\.[0-9]{0,3}[1-9])?)$`)
	errInvalid       = errors.New("quantity: invalid canonical form")
	errOverflow      = errors.New("quantity: overflow")
)

// Qty is a strictly positive quantity in 1/10000 units.
type Qty struct {
	scaled int64
}

// FromScaled builds a Qty. Rejects non-positive values.
func FromScaled(scaled int64) (Qty, error) {
	if scaled <= 0 {
		return Qty{}, errInvalid
	}
	return Qty{scaled: scaled}, nil
}

// Scaled returns the fixed-point integer representation.
func (q Qty) Scaled() int64 { return q.scaled }

// ParseCanonical parses OpenAPI DecimalQuantity into scaled int64.
func ParseCanonical(s string) (Qty, error) {
	if !canonicalPattern.MatchString(s) {
		return Qty{}, errInvalid
	}
	parts := strings.SplitN(s, ".", 2)
	major, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Qty{}, errInvalid
	}
	var fracDigits string
	if len(parts) == 2 {
		fracDigits = parts[1]
	}
	if len(fracDigits) > Scale {
		return Qty{}, errInvalid
	}
	for len(fracDigits) < Scale {
		fracDigits += "0"
	}
	frac, err := strconv.ParseInt(fracDigits, 10, 64)
	if err != nil {
		return Qty{}, errInvalid
	}
	if major > math.MaxInt64/Factor {
		return Qty{}, errOverflow
	}
	scaled := major*Factor + frac
	if scaled <= 0 {
		return Qty{}, errInvalid
	}
	return Qty{scaled: scaled}, nil
}

// FormatCanonical returns a canonical DecimalQuantity string (no trailing fractional zeros).
func (q Qty) FormatCanonical() string {
	major := q.scaled / Factor
	frac := q.scaled % Factor
	if frac == 0 {
		return strconv.FormatInt(major, 10)
	}
	s := fmt.Sprintf("%d.%04d", major, frac)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}
