package series_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/series"
)

func TestStaticIgnoresRequestedSeries(t *testing.T) {
	r, err := series.NewStatic(series.StaticConfig{EffectiveCode: "EFF-A"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Resolve("scope", "HACK-FROM-POS")
	if err != nil {
		t.Fatal(err)
	}
	if got != "EFF-A" {
		t.Fatalf("got %q want EFF-A", got)
	}
	got2, err := r.Resolve("scope", "")
	if err != nil || got2 != "EFF-A" {
		t.Fatalf("got %q %v", got2, err)
	}
}

func TestNewStaticRequiresCode(t *testing.T) {
	_, err := series.NewStatic(series.StaticConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}
