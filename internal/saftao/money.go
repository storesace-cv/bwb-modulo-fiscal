package saftao

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrValidation is fail-closed structural/XSD-shape validation (≠ AGT acceptance).
var ErrValidation = errors.New("saftao: validação estrutural")

// Money2 is SAFMonetaryType2DecimalPlaces: non-negative with exactly two fractional digits.
// Never float64. Canonical string "N.NN" (source_id AO-SAFT-XSD-1.01_01).
type Money2 string

var money2Pattern = regexp.MustCompile(`^\d+\.\d{2}$`)

// NewMoney2 validates and returns Money2.
func NewMoney2(s string) (Money2, error) {
	s = strings.TrimSpace(s)
	if !money2Pattern.MatchString(s) {
		return "", fmt.Errorf("%w: Money2 deve ser N.NN", ErrValidation)
	}
	return Money2(s), nil
}

// MustMoney2 panics on invalid input (fixtures/tests).
func MustMoney2(s string) Money2 {
	m, err := NewMoney2(s)
	if err != nil {
		panic(err)
	}
	return m
}

func (m Money2) String() string { return string(m) }

func (m Money2) Validate() error {
	if !money2Pattern.MatchString(string(m)) {
		return fmt.Errorf("%w: Money2", ErrValidation)
	}
	return nil
}

// DecimalNonNeg is SAFdecimalType / SAFmonetaryType (≥ 0). Never float64.
type DecimalNonNeg string

var decimalPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)

// NewDecimalNonNeg validates a non-negative decimal string.
func NewDecimalNonNeg(s string) (DecimalNonNeg, error) {
	s = strings.TrimSpace(s)
	if !decimalPattern.MatchString(s) {
		return "", fmt.Errorf("%w: decimal", ErrValidation)
	}
	return DecimalNonNeg(s), nil
}

func MustDecimal(s string) DecimalNonNeg {
	d, err := NewDecimalNonNeg(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d DecimalNonNeg) Validate() error {
	if !decimalPattern.MatchString(string(d)) {
		return fmt.Errorf("%w: decimal", ErrValidation)
	}
	return nil
}

func (d DecimalNonNeg) String() string { return string(d) }
