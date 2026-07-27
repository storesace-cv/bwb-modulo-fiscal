package doctype_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
)

func TestComputeAvailabilityWithoutFE(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	rep := reg.ComputeAvailability(doctype.AvailabilityInput{
		FEEnrollmentStatus: "not_enrolled",
	})
	if rep.FEAderiu {
		t.Fatal("not_enrolled must not aderiu")
	}
	if len(rep.Groups) != 5 {
		t.Fatalf("groups=%d", len(rep.Groups))
	}
	byCanon := map[string]doctype.TypeAvailability{}
	for _, row := range rep.Types {
		byCanon[row.CodigoCanonico] = row
	}
	ft := byCanon[doctype.CanonicalFT]
	if !ft.Available {
		t.Fatalf("FT should be available without FE: %+v", ft)
	}
	fa := byCanon["bwb.ao.vendas.fa"]
	if fa.Available {
		t.Fatalf("FA FE-only must be unavailable without FE: %+v", fa)
	}
	gr := byCanon["bwb.ao.movimentacao.gr"]
	if gr.Available {
		t.Fatalf("GR pending_validation + off must be unavailable: %+v", gr)
	}
	foundPending := false
	for _, r := range gr.Reasons {
		if r == doctype.ReasonAGTPending || r == doctype.ReasonTypeInactive {
			foundPending = true
		}
	}
	if !foundPending {
		t.Fatalf("GR reasons: %v", gr.Reasons)
	}
}

func TestComputeAvailabilityWithFE(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	rep := reg.ComputeAvailability(doctype.AvailabilityInput{
		FEEnrollmentStatus: "active",
		Config: doctype.AvailabilityConfig{
			TypeActiveOverride: map[string]bool{
				"bwb.ao.vendas.fa": true, // still conflito → unavailable
				"bwb.ao.movimentacao.gr": true,
			},
		},
	})
	if !rep.FEAderiu {
		t.Fatal("active ⇒ aderiu")
	}
	byCanon := map[string]doctype.TypeAvailability{}
	for _, row := range rep.Types {
		byCanon[row.CodigoCanonico] = row
	}
	ft := byCanon[doctype.CanonicalFT]
	if !ft.Available {
		t.Fatalf("FT with FE: %+v", ft)
	}
	gr := byCanon["bwb.ao.movimentacao.gr"]
	if gr.Available {
		t.Fatalf("SAF-T-only GR must be FE-matrix-unsupported when aderiu: %+v", gr)
	}
	hasFEUnsupported := false
	for _, r := range gr.Reasons {
		if r == doctype.ReasonFEMatrixUnsupported {
			hasFEUnsupported = true
		}
	}
	if !hasFEUnsupported {
		t.Fatalf("want fe_matrix_unsupported, got %v", gr.Reasons)
	}
	fa := byCanon["bwb.ao.vendas.fa"]
	if fa.Available {
		t.Fatalf("FA conflito remains unavailable: %+v", fa)
	}
}

func TestGroupInactiveBlocksTypes(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	rep := reg.ComputeAvailability(doctype.AvailabilityInput{
		FEEnrollmentStatus: "not_enrolled",
		Config: doctype.AvailabilityConfig{
			GroupActive: map[string]bool{"vendas": false},
		},
	})
	for _, row := range rep.Types {
		if row.Grupo != "vendas" {
			continue
		}
		if row.Available {
			t.Fatalf("%s available with group off", row.CodigoCanonico)
		}
	}
}
