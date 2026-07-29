package prep_test

import (
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
)

func TestBuildConnectivityStatusSimulator(t *testing.T) {
	ts := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	s := prep.BuildConnectivityStatus("homologation", "simulator", "success", "simulator", &ts)
	if s.Status != prep.ConnectivitySimulator || s.ExternalVerified || !s.SimulatorAllowed {
		t.Fatalf("%+v", s)
	}
	if s.LastProbeResult != "success" || s.LastProbeAt == nil {
		t.Fatalf("%+v", s)
	}
}

func TestBuildConnectivityStatusProductionFailClosed(t *testing.T) {
	s := prep.BuildConnectivityStatus("production", "simulator", "", "", nil)
	if s.Status != prep.ConnectivityFailClosed || s.SimulatorAllowed {
		t.Fatalf("%+v", s)
	}
}

func TestBuildConnectivityStatusAGTBlocked(t *testing.T) {
	s := prep.BuildConnectivityStatus("homologation", "agt-hml", "", "", nil)
	if s.Status != prep.ConnectivityBlockedAGT || s.ExternalVerified {
		t.Fatalf("%+v", s)
	}
}
