package doctype_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestCDOC003SeedInvariants(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	v := reg.CheckCDOC003Invariants()
	if len(v) != 0 {
		t.Fatalf("C-DOC-003 seed violations: %v", v)
	}
}

func TestCDOC004ARDualL3Seed(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	if v := reg.CheckCDOC004Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-004 seed violations: %v", v)
	}
	vendas, ok1 := reg.Lookup("bwb.ao.vendas.ar")
	pag, ok2 := reg.Lookup("bwb.ao.pagamentos.ar")
	if !ok1 || !ok2 {
		t.Fatal("AR dual seeds missing")
	}
	if vendas.CodigoCanonico == pag.CodigoCanonico {
		t.Fatal("AR canonicals must remain distinct")
	}
	vLayer, vCode := doctype.ParseSAFTTypeAdapter(vendas.ChannelAdapters.SAFTType)
	pLayer, pCode := doctype.ParseSAFTTypeAdapter(pag.ChannelAdapters.SAFTType)
	if vLayer != doctype.SAFTLayerInvoice || vCode != "AR" || vendas.ChannelAdapters.SAFTStructure != "SalesInvoices" {
		t.Fatalf("vendas.ar: layer=%q code=%q l3=%q", vLayer, vCode, vendas.ChannelAdapters.SAFTStructure)
	}
	if pLayer != doctype.SAFTLayerPayment || pCode != "AR" || pag.ChannelAdapters.SAFTStructure != "Payments" {
		t.Fatalf("pagamentos.ar: layer=%q code=%q l3=%q", pLayer, pCode, pag.ChannelAdapters.SAFTStructure)
	}
	if !saftao.ValidInvoiceType(saftao.InvoiceTypeAR) {
		t.Fatal("XSD InvoiceType must accept AR")
	}
	if !saftao.ValidPaymentType(saftao.PaymentTypeAR) {
		t.Fatal("XSD PaymentType must accept AR")
	}
	// Dual membership in XSD must not collapse product routing.
	if vendas.Activo != doctype.ActiveOff || pag.Activo != doctype.ActiveOff {
		t.Fatalf("AR seeds must stay off: vendas=%q pag=%q", vendas.Activo, pag.Activo)
	}
}


func TestCDOC003FAIsFEOnly(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	fa, ok := reg.Lookup("bwb.ao.vendas.fa")
	if !ok {
		t.Fatal("FA missing from catalog")
	}
	if fa.ChannelAdapters.FECode != "FA" {
		t.Fatalf("FE FA: %q", fa.ChannelAdapters.FECode)
	}
	layer, code := doctype.ParseSAFTTypeAdapter(fa.ChannelAdapters.SAFTType)
	if layer != doctype.SAFTLayerNone || code != "" {
		t.Fatalf("FA must have empty SAF-T adapter: layer=%q code=%q raw=%q", layer, code, fa.ChannelAdapters.SAFTType)
	}
	if fa.ChannelAdapters.Eligibility != "FE" {
		t.Fatalf("FA eligibility: %q", fa.ChannelAdapters.Eligibility)
	}
	if saftao.ValidInvoiceType(saftao.InvoiceType("FA")) {
		t.Fatal("XSD InvoiceType must not accept FA")
	}
	if saftao.ValidPaymentType(saftao.PaymentType("FA")) {
		t.Fatal("XSD PaymentType must not accept FA")
	}
}

func TestCDOC003RCPaymentNotInvoice(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	rc, ok := reg.Lookup("bwb.ao.pagamentos.rc")
	if !ok {
		t.Fatal("RC missing")
	}
	layer, code := doctype.ParseSAFTTypeAdapter(rc.ChannelAdapters.SAFTType)
	if layer != doctype.SAFTLayerPayment || code != "RC" {
		t.Fatalf("RC SAF-T: layer=%q code=%q raw=%q", layer, code, rc.ChannelAdapters.SAFTType)
	}
	if saftao.ValidInvoiceType(saftao.InvoiceType("RC")) {
		t.Fatal("InvoiceType must not accept RC")
	}
	if !saftao.ValidPaymentType(saftao.PaymentTypeRC) {
		t.Fatal("PaymentType RC must be valid")
	}
}

func TestCDOC003RGPaymentNotInvoice(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	rg, ok := reg.Lookup("bwb.ao.pagamentos.rg")
	if !ok {
		t.Fatal("RG missing")
	}
	layer, code := doctype.ParseSAFTTypeAdapter(rg.ChannelAdapters.SAFTType)
	if layer != doctype.SAFTLayerPayment || code != "RG" {
		t.Fatalf("RG SAF-T: layer=%q code=%q raw=%q", layer, code, rg.ChannelAdapters.SAFTType)
	}
	if saftao.ValidInvoiceType(saftao.InvoiceType("RG")) {
		t.Fatal("InvoiceType must not accept RG")
	}
	if !saftao.ValidPaymentType(saftao.PaymentTypeRG) {
		t.Fatal("PaymentType RG must be valid")
	}
}

func TestParseSAFTTypeAdapter(t *testing.T) {
	cases := []struct {
		in    string
		layer doctype.SAFTLayer
		code  string
	}{
		{"", doctype.SAFTLayerNone, ""},
		{"∅", doctype.SAFTLayerNone, ""},
		{"InvoiceType=FT", doctype.SAFTLayerInvoice, "FT"},
		{"PaymentType=RC", doctype.SAFTLayerPayment, "RC"},
		{"PurchaseType=FT", doctype.SAFTLayerOther, "FT"},
	}
	for _, tc := range cases {
		layer, code := doctype.ParseSAFTTypeAdapter(tc.in)
		if layer != tc.layer || code != tc.code {
			t.Fatalf("%q → %q/%q want %q/%q", tc.in, layer, code, tc.layer, tc.code)
		}
	}
}

func TestGFRemainsConflictOffCDOC001(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	gf, ok := reg.Lookup("bwb.ao.vendas.gf")
	if !ok {
		t.Fatal("GF missing from catalog")
	}
	if gf.Activo != doctype.ActiveOff {
		t.Fatalf("GF must stay off until C-DOC-001 compliance close: %q", gf.Activo)
	}
	if gf.EstadoNormativo != "conflito" {
		t.Fatalf("GF estado_normativo: %q", gf.EstadoNormativo)
	}
	if gf.ChannelAdapters.FECode != "GF" {
		t.Fatalf("FE GF: %q", gf.ChannelAdapters.FECode)
	}
	layer, code := doctype.ParseSAFTTypeAdapter(gf.ChannelAdapters.SAFTType)
	if layer != doctype.SAFTLayerInvoice || code != "GF" {
		t.Fatalf("SAFT GF adapter: layer=%q code=%q raw=%q", layer, code, gf.ChannelAdapters.SAFTType)
	}
	// Availability must stay blocked by normative conflict even if forcefully activated.
	rep := reg.ComputeAvailability(doctype.AvailabilityInput{
		FEEnrollmentStatus: "active",
		Config: doctype.AvailabilityConfig{
			TypeActiveOverride: map[string]bool{"bwb.ao.vendas.gf": true},
		},
	})
	var row *doctype.TypeAvailability
	for i := range rep.Types {
		if rep.Types[i].CodigoCanonico == "bwb.ao.vendas.gf" {
			row = &rep.Types[i]
			break
		}
	}
	if row == nil {
		t.Fatal("GF availability row missing")
	}
	if row.Available {
		t.Fatal("GF must not be available while estado_normativo=conflito")
	}
	found := false
	for _, r := range row.Reasons {
		if r == doctype.ReasonAGTPending {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ReasonAGTPending, got %v", row.Reasons)
	}
}
