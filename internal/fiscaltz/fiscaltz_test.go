package fiscaltz_test

import (
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaltime"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaltz"
)

func TestStaticResolveOK(t *testing.T) {
	r, err := fiscaltz.NewStaticAfricaLuanda("scope-dev")
	if err != nil {
		t.Fatal(err)
	}
	tz, err := r.Resolve("scope-dev")
	if err != nil || tz != fiscaltime.AfricaLuanda {
		t.Fatalf("tz=%q err=%v", tz, err)
	}
}

func TestStaticFailClosedUnknownScope(t *testing.T) {
	r, err := fiscaltz.NewStaticAfricaLuanda("scope-dev")
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Resolve("other")
	if !errors.Is(err, fiscaltz.ErrUnresolved) {
		t.Fatalf("err=%v", err)
	}
}

func TestStaticRejectsEmptyAndInvalid(t *testing.T) {
	if _, err := fiscaltz.NewStatic(fiscaltz.StaticConfig{ScopeID: "", Timezone: fiscaltime.AfricaLuanda}); err == nil {
		t.Fatal("empty scope")
	}
	if _, err := fiscaltz.NewStatic(fiscaltz.StaticConfig{ScopeID: "s", Timezone: ""}); err == nil {
		t.Fatal("empty tz")
	}
	if _, err := fiscaltz.NewStatic(fiscaltz.StaticConfig{ScopeID: "s", Timezone: "Not/AZone"}); err == nil {
		t.Fatal("invalid tz")
	}
}

func TestCapeVerdeNotWiredAsAngolaHelper(t *testing.T) {
	// Constant exists for documentation only; Angola helper must not return CV zone.
	r, err := fiscaltz.NewStaticAfricaLuanda("scope")
	if err != nil {
		t.Fatal(err)
	}
	tz, _ := r.Resolve("scope")
	if tz == fiscaltime.AtlanticCapeVerde {
		t.Fatal("Cabo Verde must not be wired via Angola helper")
	}
}
