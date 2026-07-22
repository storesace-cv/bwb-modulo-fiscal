package buildinfo_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/buildinfo"
)

const validSHA = "2f96fe45c0d8ad3cb2e21d8755f2988eb4a43dfd"

func TestValidate_dev(t *testing.T) {
	if err := buildinfo.Validate(buildinfo.DevRevision); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_validSHA(t *testing.T) {
	if err := buildinfo.Validate(validSHA); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_empty(t *testing.T) {
	if err := buildinfo.Validate(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_unknown(t *testing.T) {
	if err := buildinfo.Validate(buildinfo.UnknownRevision); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_invalidFormat(t *testing.T) {
	cases := []string{
		"2F96FE45C0D8AD3CB2E21D8755F2988EB4A43DFD",
		"2f96fe45c0d8ad3cb2e21d8755f2988eb4a43df",
		"not-a-sha",
	}
	for _, c := range cases {
		if err := buildinfo.Validate(c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}

func TestValidateForEnv_developmentDevOK(t *testing.T) {
	if err := buildinfo.ValidateForEnv(buildinfo.DevRevision, "development"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateForEnv_homologationDevRejected(t *testing.T) {
	if err := buildinfo.ValidateForEnv(buildinfo.DevRevision, "homologation"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateForEnv_productionDevRejected(t *testing.T) {
	if err := buildinfo.ValidateForEnv(buildinfo.DevRevision, "production"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateForEnv_productionSHAOK(t *testing.T) {
	if err := buildinfo.ValidateForEnv(validSHA, "production"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateForEnv_uppercaseRejected(t *testing.T) {
	up := "2F96FE45C0D8AD3CB2E21D8755F2988EB4A43DFD"
	if err := buildinfo.ValidateForEnv(up, "homologation"); err == nil {
		t.Fatal("expected error")
	}
	if err := buildinfo.ValidateRelease(up, up); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRelease_ok(t *testing.T) {
	if err := buildinfo.ValidateRelease(validSHA, validSHA); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRelease_mismatch(t *testing.T) {
	other := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := buildinfo.ValidateRelease(validSHA, other); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestValidateRelease_noCaseFold(t *testing.T) {
	up := "2F96FE45C0D8AD3CB2E21D8755F2988EB4A43DFD"
	if err := buildinfo.ValidateRelease(validSHA, up); err == nil {
		t.Fatal("must not fold commit case")
	}
}

func TestValidateRelease_devRejected(t *testing.T) {
	if err := buildinfo.ValidateRelease(buildinfo.DevRevision, validSHA); err == nil {
		t.Fatal("expected error")
	}
}
