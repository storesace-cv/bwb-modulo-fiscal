package quantity_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

func TestParseFormatRoundTrip(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1", "1"},
		{"1.5", "1.5"},
		{"0.0001", "0.0001"},
		{"12.3456", "12.3456"},
		{"1.01", "1.01"},
	}
	for _, c := range cases {
		q, err := quantity.ParseCanonical(c.in)
		if err != nil {
			t.Fatalf("ParseCanonical(%q): %v", c.in, err)
		}
		if got := q.FormatCanonical(); got != c.want {
			t.Fatalf("FormatCanonical(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRejectsInvalid(t *testing.T) {
	for _, c := range []string{"", "0", "0.0", "0.0000", "01", "-1", "1.0000", "1.00000"} {
		if _, err := quantity.ParseCanonical(c); err == nil {
			t.Fatalf("ParseCanonical(%q) expected error", c)
		}
	}
}

func TestFromScaledRejectsNonPositive(t *testing.T) {
	if _, err := quantity.FromScaled(0); err == nil {
		t.Fatal("expected error")
	}
}
