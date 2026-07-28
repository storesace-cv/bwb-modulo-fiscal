package signsep_test

import (
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaljws"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/signsep"
)

func TestCSign001PackageInvariants(t *testing.T) {
	v := signsep.CheckInvariants()
	if len(v) != 0 {
		t.Fatalf("C-SIGN-001 invariants: %v", v)
	}
}

func TestCSign001MechanismsDistinct(t *testing.T) {
	if signsep.MechanismSAFTHash == signsep.MechanismFEJWS {
		t.Fatal("mechanisms must differ")
	}
	if signsep.FETechnicalAlgorithm != fiscaljws.Algorithm || fiscaljws.Algorithm != "RS256" {
		t.Fatalf("FE technical alg: %q", signsep.FETechnicalAlgorithm)
	}
	if signsep.SAFTCitedHashDigest != "SHA-1" {
		t.Fatalf("cited SAF-T digest: %q", signsep.SAFTCitedHashDigest)
	}
}

func TestCSign001ExportMarksPendingHash(t *testing.T) {
	base := saftao.MinimalSalesInvoiceFixture()
	res, err := saftao.BuildIncrementalExport(saftao.ExportRequest{
		Header:              *base.Header,
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		AllowedInvoiceTypes: []saftao.InvoiceType{saftao.InvoiceTypeFT},
		Customers:           base.MasterFiles.Customer,
		Products:            base.MasterFiles.Product,
		Invoices:            append([]saftao.Invoice{}, base.SourceDocuments.SalesInvoices.Invoice...),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingHashAlgorithm {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("export must include PendingHashAlgorithm, got %v", res.PendingRegulatory)
	}
	// Artifact SHA-256 integrity ≠ Invoice.Hash algorithm (C-SIGN-001).
	if res.SHA256 == "" || len(res.SHA256) != 64 {
		t.Fatalf("artifact sha: %q", res.SHA256)
	}
	if strings.EqualFold(res.SHA256, string(saftao.PendingHashAlgorithm)) {
		t.Fatal("artifact SHA must not equal pending hash marker")
	}
}

func TestRejectConflatedAlgorithm(t *testing.T) {
	if err := signsep.RejectConflatedAlgorithm("RS256", signsep.MechanismFEJWS); err != nil {
		t.Fatal(err)
	}
	if err := signsep.RejectConflatedAlgorithm("SHA-1", signsep.MechanismFEJWS); err == nil {
		t.Fatal("SHA-1 must not be accepted as FE JWS")
	}
	if err := signsep.RejectConflatedAlgorithm("RS256", signsep.MechanismSAFTHash); err == nil {
		t.Fatal("RS256 must not be accepted as SAF-T Hash mechanism")
	}
	if err := signsep.RejectConflatedAlgorithm("pending_ao", signsep.MechanismSAFTHash); err != nil {
		t.Fatal(err)
	}
}
