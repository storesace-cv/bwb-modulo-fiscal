package saftao

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Date is xs:date / SAFdateType as YYYY-MM-DD (calendar date; no timezone).
type Date string

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// NewDate validates YYYY-MM-DD.
func NewDate(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if !datePattern.MatchString(s) {
		return "", fmt.Errorf("%w: Date YYYY-MM-DD", ErrValidation)
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return "", fmt.Errorf("%w: Date inválida", ErrValidation)
	}
	return Date(s), nil
}

func MustDate(s string) Date {
	d, err := NewDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d Date) Validate() error {
	_, err := NewDate(string(d))
	return err
}

func (d Date) String() string { return string(d) }

// DateTime is xs:dateTime / SAFdateTimeType stored as a canonical string.
// Accepts RFC3339 or local "YYYY-MM-DDTHH:MM:SS".
type DateTime string

// NewDateTime validates and normalizes dateTime strings.
func NewDateTime(s string) (DateTime, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("%w: DateTime vazio", ErrValidation)
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return DateTime(t.UTC().Format("2006-01-02T15:04:05Z")), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return DateTime(t.Format("2006-01-02T15:04:05")), nil
	}
	return "", fmt.Errorf("%w: DateTime", ErrValidation)
}

func MustDateTime(s string) DateTime {
	dt, err := NewDateTime(s)
	if err != nil {
		panic(err)
	}
	return dt
}

func (dt DateTime) Validate() error {
	_, err := NewDateTime(string(dt))
	return err
}

func (dt DateTime) String() string { return string(dt) }
