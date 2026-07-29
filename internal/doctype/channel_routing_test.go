package doctype_test

import (
	"strings"
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

func TestCDOC005InsurerDualL3Seed(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	if v := reg.CheckCDOC005Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-005 seed violations: %v", v)
	}
	codes := []string{"RP", "RE", "CS", "LD", "RA"}
	for _, code := range codes {
		canon := "bwb.ao.vendas." + strings.ToLower(code)
		e, ok := reg.Lookup(canon)
		if !ok {
			t.Fatalf("missing %s", canon)
		}
		layer, c := doctype.ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		if layer != doctype.SAFTLayerInvoice || c != code || e.ChannelAdapters.SAFTStructure != "SalesInvoices" {
			t.Fatalf("%s: layer=%q code=%q l3=%q", canon, layer, c, e.ChannelAdapters.SAFTStructure)
		}
		if e.Activo != doctype.ActiveOff {
			t.Fatalf("%s must stay off", canon)
		}
		if !saftao.ValidInvoiceType(saftao.InvoiceType(code)) {
			t.Fatalf("InvoiceType must accept %s", code)
		}
		if !saftao.ValidWorkType(saftao.WorkType(code)) {
			t.Fatalf("WorkType must accept %s (dual membership)", code)
		}
	}
}

func TestCDOC006RCPaymentVsPurchase(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	if v := reg.CheckCDOC006Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-006 seed violations: %v", v)
	}
	pag, ok1 := reg.Lookup("bwb.ao.pagamentos.rc")
	com, ok2 := reg.Lookup("bwb.ao.compras.rc")
	if !ok1 || !ok2 {
		t.Fatal("RC dual seeds missing")
	}
	if pag.CodigoCanonico == com.CodigoCanonico {
		t.Fatal("RC canonicals must remain distinct")
	}
	pLayer, pCode := doctype.ParseSAFTTypeAdapter(pag.ChannelAdapters.SAFTType)
	cLayer, cCode := doctype.ParseSAFTTypeAdapter(com.ChannelAdapters.SAFTType)
	if pLayer != doctype.SAFTLayerPayment || pCode != "RC" || pag.ChannelAdapters.SAFTStructure != "Payments" {
		t.Fatalf("pagamentos.rc: layer=%q code=%q l3=%q", pLayer, pCode, pag.ChannelAdapters.SAFTStructure)
	}
	if cLayer != doctype.SAFTLayerPurchase || cCode != "RC" || com.ChannelAdapters.SAFTStructure != "PurchaseInvoices" {
		t.Fatalf("compras.rc: layer=%q code=%q l3=%q", cLayer, cCode, com.ChannelAdapters.SAFTStructure)
	}
	if pag.ChannelAdapters.FECode != "RC" {
		t.Fatalf("pagamentos.rc FE: %q", pag.ChannelAdapters.FECode)
	}
	if com.ChannelAdapters.FECode != "" {
		t.Fatalf("compras.rc must be FE-empty, got %q", com.ChannelAdapters.FECode)
	}
	if !saftao.ValidPaymentType(saftao.PaymentTypeRC) {
		t.Fatal("PaymentType must accept RC")
	}
	if !saftao.ValidPurchaseType(saftao.PurchaseTypeRC) {
		t.Fatal("PurchaseType must accept RC (dual membership)")
	}
	if saftao.ValidInvoiceType(saftao.InvoiceType("RC")) {
		t.Fatal("InvoiceType must not accept RC")
	}
	if pag.Activo != doctype.ActiveOff || com.Activo != doctype.ActiveOff {
		t.Fatalf("RC seeds must stay off: pag=%q com=%q", pag.Activo, com.Activo)
	}
}

func TestCDOC007GRMovementVsWork(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	if v := reg.CheckCDOC007Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-007 seed violations: %v", v)
	}
	mov, ok1 := reg.Lookup("bwb.ao.movimentacao.gr")
	conf, ok2 := reg.Lookup("bwb.ao.conferencia.gr")
	if !ok1 || !ok2 {
		t.Fatal("GR dual seeds missing")
	}
	if mov.CodigoCanonico == conf.CodigoCanonico {
		t.Fatal("GR canonicals must remain distinct")
	}
	mLayer, mCode := doctype.ParseSAFTTypeAdapter(mov.ChannelAdapters.SAFTType)
	cLayer, cCode := doctype.ParseSAFTTypeAdapter(conf.ChannelAdapters.SAFTType)
	if mLayer != doctype.SAFTLayerMovement || mCode != "GR" || mov.ChannelAdapters.SAFTStructure != "MovementOfGoods" {
		t.Fatalf("movimentacao.gr: layer=%q code=%q l3=%q", mLayer, mCode, mov.ChannelAdapters.SAFTStructure)
	}
	if cLayer != doctype.SAFTLayerWork || cCode != "GR" || conf.ChannelAdapters.SAFTStructure != "WorkingDocuments" {
		t.Fatalf("conferencia.gr: layer=%q code=%q l3=%q", cLayer, cCode, conf.ChannelAdapters.SAFTStructure)
	}
	if mov.ChannelAdapters.FECode != "" || conf.ChannelAdapters.FECode != "" {
		t.Fatalf("GR seeds must be FE-empty: mov=%q conf=%q", mov.ChannelAdapters.FECode, conf.ChannelAdapters.FECode)
	}
	if !saftao.ValidMovementType(saftao.MovementTypeGR) {
		t.Fatal("MovementType must accept GR")
	}
	if !saftao.ValidWorkType(saftao.WorkTypeGR) {
		t.Fatal("WorkType must accept GR (dual membership)")
	}
	if mov.Activo != doctype.ActiveOff || conf.Activo != doctype.ActiveOff {
		t.Fatalf("GR seeds must stay off: mov=%q conf=%q", mov.Activo, conf.Activo)
	}
}

func TestCDOC008FTNCInvoiceVsPurchase(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	if v := reg.CheckCDOC008Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-008 seed violations: %v", v)
	}
	for _, code := range []string{"FT", "NC"} {
		vCanon := "bwb.ao.vendas." + strings.ToLower(code)
		cCanon := "bwb.ao.compras." + strings.ToLower(code)
		vendas, okV := reg.Lookup(vCanon)
		compras, okC := reg.Lookup(cCanon)
		if !okV || !okC {
			t.Fatalf("dual seeds missing for %s", code)
		}
		if vendas.CodigoCanonico == compras.CodigoCanonico {
			t.Fatalf("canonicals must remain distinct for %s", code)
		}
		vLayer, vCode := doctype.ParseSAFTTypeAdapter(vendas.ChannelAdapters.SAFTType)
		cLayer, cCode := doctype.ParseSAFTTypeAdapter(compras.ChannelAdapters.SAFTType)
		if vLayer != doctype.SAFTLayerInvoice || vCode != code || vendas.ChannelAdapters.SAFTStructure != "SalesInvoices" {
			t.Fatalf("%s: layer=%q code=%q l3=%q", vCanon, vLayer, vCode, vendas.ChannelAdapters.SAFTStructure)
		}
		if cLayer != doctype.SAFTLayerPurchase || cCode != code || compras.ChannelAdapters.SAFTStructure != "PurchaseInvoices" {
			t.Fatalf("%s: layer=%q code=%q l3=%q", cCanon, cLayer, cCode, compras.ChannelAdapters.SAFTStructure)
		}
		if vendas.ChannelAdapters.FECode != code {
			t.Fatalf("%s FE: %q", vCanon, vendas.ChannelAdapters.FECode)
		}
		if compras.ChannelAdapters.FECode != "" {
			t.Fatalf("%s must be FE-empty, got %q", cCanon, compras.ChannelAdapters.FECode)
		}
		if !saftao.ValidInvoiceType(saftao.InvoiceType(code)) {
			t.Fatalf("InvoiceType must accept %s", code)
		}
		if !saftao.ValidPurchaseType(saftao.PurchaseType(code)) {
			t.Fatalf("PurchaseType must accept %s (dual membership)", code)
		}
		if compras.Activo != doctype.ActiveOff {
			t.Fatalf("%s must stay off", cCanon)
		}
		// DEC-REG-003 defaults: FT/NC vendas may be on.
		if vendas.Activo != doctype.ActiveOn {
			t.Fatalf("%s expected ActiveOn (DEC-REG-003 defaults), got %q", vCanon, vendas.Activo)
		}
	}
}

func TestCDOC009ARPurchaseThirdL3(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	if v := reg.CheckCDOC009Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-009 seed violations: %v", v)
	}
	// C-DOC-004 dual must still hold.
	if v := reg.CheckCDOC004Invariants(); len(v) != 0 {
		t.Fatalf("C-DOC-004 regressions: %v", v)
	}
	vendas, ok1 := reg.Lookup("bwb.ao.vendas.ar")
	pag, ok2 := reg.Lookup("bwb.ao.pagamentos.ar")
	com, ok3 := reg.Lookup("bwb.ao.compras.ar")
	if !ok1 || !ok2 || !ok3 {
		t.Fatal("AR triple seeds missing")
	}
	ids := map[string]struct{}{vendas.CodigoCanonico: {}, pag.CodigoCanonico: {}, com.CodigoCanonico: {}}
	if len(ids) != 3 {
		t.Fatal("AR canonicals must be three distinct ids")
	}
	cLayer, cCode := doctype.ParseSAFTTypeAdapter(com.ChannelAdapters.SAFTType)
	if cLayer != doctype.SAFTLayerPurchase || cCode != "AR" || com.ChannelAdapters.SAFTStructure != "PurchaseInvoices" {
		t.Fatalf("compras.ar: layer=%q code=%q l3=%q", cLayer, cCode, com.ChannelAdapters.SAFTStructure)
	}
	if com.ChannelAdapters.FECode != "" {
		t.Fatalf("compras.ar must be FE-empty, got %q", com.ChannelAdapters.FECode)
	}
	if !saftao.ValidPurchaseType(saftao.PurchaseTypeAR) {
		t.Fatal("PurchaseType must accept AR (third L3)")
	}
	if com.Activo != doctype.ActiveOff {
		t.Fatalf("compras.ar must stay off: %q", com.Activo)
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
		{"WorkType=RP", doctype.SAFTLayerWork, "RP"},
		{"PurchaseType=RC", doctype.SAFTLayerPurchase, "RC"},
		{"PurchaseType=FT", doctype.SAFTLayerPurchase, "FT"},
		{"MovementType=GR", doctype.SAFTLayerMovement, "GR"},
		{"UnknownType=X", doctype.SAFTLayerOther, "X"},
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
