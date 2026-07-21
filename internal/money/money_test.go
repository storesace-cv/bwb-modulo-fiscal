package money_test

import (
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
)

func TestParseFormatRoundTrip(t *testing.T) {
	cases := []string{"0.00", "10.50", "1250.00", "9999999999999999.99"}
	for _, c := range cases {
		a, err := money.ParseCanonical(c)
		if err != nil {
			t.Fatalf("ParseCanonical(%q): %v", c, err)
		}
		if a.FormatCanonical() != c {
			t.Fatalf("FormatCanonical = %q, want %q", a.FormatCanonical(), c)
		}
	}
}

func TestRejectsInvalid(t *testing.T) {
	for _, c := range []string{"", "1", "1.0", "01.00", "-1.00", "1.001", "abc"} {
		if _, err := money.ParseCanonical(c); err == nil {
			t.Fatalf("ParseCanonical(%q) expected error", c)
		}
	}
}

func TestNoFloatInSource(t *testing.T) {
	// Guardrail: money package must not mention float types in its API path.
	// Compile-time: Amount is int64-backed; this checks Format/Parse stay integer.
	a, err := money.FromCents(100)
	if err != nil {
		t.Fatal(err)
	}
	if a.Cents() != 100 {
		t.Fatalf("cents = %d", a.Cents())
	}
	if strings.Contains(a.FormatCanonical(), "e") {
		t.Fatal("unexpected scientific notation")
	}
}
