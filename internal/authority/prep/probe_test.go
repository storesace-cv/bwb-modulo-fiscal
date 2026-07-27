package prep_test

import (
	"context"
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
)

func TestFailClosedProductionSimulator(t *testing.T) {
	if err := prep.FailClosedProduction("production", config.AuthoritySimulator); !errors.Is(err, prep.ErrFailClosed) {
		t.Fatalf("got %v", err)
	}
	if err := prep.FailClosedProduction("development", config.AuthoritySimulator); err != nil {
		t.Fatal(err)
	}
	if err := prep.FailClosedProduction("homologation", config.AuthorityAGTHML); !errors.Is(err, prep.ErrFailClosed) {
		t.Fatalf("got %v", err)
	}
}

func TestRequirePrepStillBlocksAGT(t *testing.T) {
	err := prep.RequirePrepForReservedModes(config.AuthorityAGTHML, prep.Readiness{
		ConfigReady: true, SecretsReady: true, OfflineValidated: true,
	})
	if !errors.Is(err, prep.ErrFailClosed) {
		t.Fatalf("got %v", err)
	}
}

func TestProbeSimulator(t *testing.T) {
	c := simulator.New(simulator.OutcomeAccept)
	rep, err := prep.ProbeSimulator(context.Background(), config.AuthoritySimulator, c)
	if err != nil || !rep.OK || !rep.SimulatorReachable || rep.ExternalVerified {
		t.Fatalf("%+v %v", rep, err)
	}
	_, err = prep.ProbeSimulator(context.Background(), config.AuthorityAGTHML, c)
	if !errors.Is(err, prep.ErrNotSimulator) {
		t.Fatalf("got %v", err)
	}
}
