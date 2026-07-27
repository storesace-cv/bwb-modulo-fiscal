package adminregistry_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

func TestFEAderiuAndEffectiveStatus(t *testing.T) {
	if adminregistry.FEAderiu(adminregistry.FEEnrollmentActive) != true {
		t.Fatal("active deve aderir")
	}
	if adminregistry.FEAderiu(adminregistry.FEEnrollmentPending) {
		t.Fatal("pending não adere")
	}
	rows := []adminregistry.FEEnrollment{
		{Environment: adminregistry.EnvHomologation, Status: adminregistry.FEEnrollmentActive},
	}
	if got := adminregistry.EffectiveFEStatus(rows, adminregistry.EnvHomologation); got != adminregistry.FEEnrollmentActive {
		t.Fatalf("got %s", got)
	}
	if got := adminregistry.EffectiveFEStatus(rows, adminregistry.EnvProduction); got != adminregistry.FEEnrollmentNotEnrolled {
		t.Fatalf("missing env want not_enrolled got %s", got)
	}
}
