package saftao_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestEmbeddedXSDIntegrityAndInventory(t *testing.T) {
	if err := saftao.VerifyEmbeddedXSD(); err != nil {
		t.Fatal(err)
	}
	if err := saftao.EnsureRequiredStructure(); err != nil {
		t.Fatal(err)
	}
	meta := saftao.Meta()
	if meta.SourceID != "AO-SAFT-XSD-1.01_01" {
		t.Fatalf("source_id %q", meta.SourceID)
	}
	if meta.Certified || meta.Status != "pending_validation" {
		t.Fatalf("must not claim certification: %+v", meta)
	}
	if meta.TargetNamespace != "urn:OECD:StandardAuditFile-Tax:AO_1.01_01" {
		t.Fatalf("ns %q", meta.TargetNamespace)
	}
}

func TestEmptyAuditFileMarshalSkeleton(t *testing.T) {
	doc := saftao.NewEmptyAuditFile()
	raw, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "AuditFile") || !strings.Contains(s, "Header") {
		t.Fatalf("skeleton: %s", s)
	}
	if strings.Contains(strings.ToLower(s), "certified=\"true\"") {
		t.Fatal("unexpected certified claim")
	}
}

func TestHeaderAndSourceDocumentsShape(t *testing.T) {
	if err := saftao.EnsureHeaderShape(); err != nil {
		t.Fatal(err)
	}
	if err := saftao.EnsureSourceDocumentsTables(); err != nil {
		t.Fatal(err)
	}
	fields, err := saftao.HeaderFieldInventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) < len(saftao.RequiredHeaderChildren) {
		t.Fatalf("header fields too short: %v", fields)
	}
	doc := saftao.NewSalesSkeleton()
	raw, err := xml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "SalesInvoices") || !strings.Contains(s, "NumberOfEntries") {
		t.Fatalf("sales skeleton: %s", s)
	}
}

func TestDocumentGroupsFiveL3(t *testing.T) {
	groups := saftao.AllDocumentGroups()
	if len(groups) != 5 {
		t.Fatalf("want 5 groups, got %d", len(groups))
	}
	want := map[saftao.DocumentGroup]bool{
		saftao.GroupSalesInvoices:    true,
		saftao.GroupMovementOfGoods:  true,
		saftao.GroupWorkingDocuments: true,
		saftao.GroupPayments:         true,
		saftao.GroupPurchaseInvoices: true,
	}
	for _, g := range groups {
		if !want[g] || !saftao.ValidDocumentGroup(g) {
			t.Fatalf("unexpected group %q", g)
		}
	}
	if saftao.ValidDocumentGroup("Invented") {
		t.Fatal("invented group must fail")
	}
}

func TestMoneyAndDatesNoFloat(t *testing.T) {
	if _, err := saftao.NewMoney2("10.5"); err == nil {
		t.Fatal("Money2 must require exactly 2 decimals")
	}
	m, err := saftao.NewMoney2("10.50")
	if err != nil {
		t.Fatal(err)
	}
	if m.String() != "10.50" {
		t.Fatalf("got %q", m)
	}
	if _, err := saftao.NewDate("2026-13-01"); err == nil {
		t.Fatal("invalid date")
	}
	if _, err := saftao.NewDateTime("not-a-datetime"); err == nil {
		t.Fatal("invalid datetime")
	}
}

func TestInvoiceStructuralValidation(t *testing.T) {
	doc := saftao.MinimalSalesInvoiceFixture()
	if err := doc.SourceDocuments.SalesInvoices.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	bad := doc.SourceDocuments.SalesInvoices.Invoice[0]
	bad.InvoiceNo = "BAD"
	if err := bad.ValidateStructural(); err == nil {
		t.Fatal("expected InvoiceNo failure")
	}
	line := bad
	line.InvoiceNo = "FT S001/1"
	line.Line[0].DebitAmount = line.Line[0].CreditAmount
	line.Line[0].CreditAmount = nil
	// both set or both nil — set both
	c := saftao.MustMoney2("1.00")
	d := saftao.MustMoney2("1.00")
	line.Line[0].CreditAmount = &c
	line.Line[0].DebitAmount = &d
	if err := line.ValidateStructural(); err == nil {
		t.Fatal("expected Debit XOR Credit failure")
	}
}

func TestMarshalDeterministicAndNamespace(t *testing.T) {
	doc := saftao.MinimalSalesInvoiceFixture()
	a, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	b, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("marshal not deterministic")
	}
	s := string(a)
	if !strings.HasPrefix(s, xml.Header) {
		t.Fatal("missing XML declaration")
	}
	if !strings.Contains(s, saftao.Namespace) {
		t.Fatalf("missing namespace: %s", s[:200])
	}
	if !strings.Contains(s, "<Invoice>") || !strings.Contains(s, "<Line>") {
		t.Fatalf("missing Invoice/Line: %s", s)
	}
	if strings.Contains(strings.ToLower(s), "certified") {
		t.Fatal("must not claim certified")
	}
}

func TestMinimalFixtureAgainstXSD(t *testing.T) {
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	doc := saftao.MinimalSalesInvoiceFixture()
	if err := doc.SourceDocuments.SalesInvoices.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD structural validation failed:\n%s\nXML:\n%s", err, raw)
	}
}

func TestPendingRegulatoryMarkers(t *testing.T) {
	if saftao.PendingHashAlgorithm == "" || saftao.PendingInvoiceTypeSemantics == "" {
		t.Fatal("pending markers required")
	}
	if saftao.PendingMovementTypeSemantics == "" || saftao.PendingWorkTypeSemantics == "" {
		t.Fatal("pending movement/work markers required")
	}
	if saftao.PendingPaymentTypeSemantics == "" {
		t.Fatal("pending payment marker required")
	}
	if saftao.PendingPurchaseTypeSemantics == "" {
		t.Fatal("pending purchase marker required")
	}
	if saftao.PendingTaxTableSemantics == "" {
		t.Fatal("pending tax table marker required")
	}
	if saftao.PendingGLAccountSemantics == "" {
		t.Fatal("pending GL account marker required")
	}
}

func TestMovementOfGoodsStructuralAndXSD(t *testing.T) {
	doc := saftao.MinimalMovementOfGoodsFixture()
	if err := doc.SourceDocuments.MovementOfGoods.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	bad := doc.SourceDocuments.MovementOfGoods.StockMovement[0]
	bad.CustomerID = ""
	bad.SupplierID = ""
	if err := bad.ValidateStructural(); err == nil {
		t.Fatal("expected CustomerID XOR SupplierID")
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "MovementOfGoods") || !strings.Contains(string(raw), "StockMovement") {
		t.Fatalf("marshal: %s", raw)
	}
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD: %v\n%s", err, raw)
	}
}

func TestWorkingDocumentsStructuralAndXSD(t *testing.T) {
	doc := saftao.MinimalWorkingDocumentsFixture()
	if err := doc.SourceDocuments.WorkingDocuments.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "WorkingDocuments") || !strings.Contains(string(raw), "WorkDocument") {
		t.Fatalf("marshal: %s", raw)
	}
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD: %v\n%s", err, raw)
	}
}

func TestPaymentsStructuralAndXSD(t *testing.T) {
	doc := saftao.MinimalPaymentsFixture()
	if err := doc.SourceDocuments.Payments.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Payments") || !strings.Contains(string(raw), "PaymentRefNo") {
		t.Fatalf("marshal: %s", raw)
	}
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD: %v\n%s", err, raw)
	}
}

func TestPurchaseInvoicesStructuralAndXSD(t *testing.T) {
	doc := saftao.MinimalPurchaseInvoicesFixture()
	if err := doc.SourceDocuments.PurchaseInvoices.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "PurchaseInvoices") || !strings.Contains(s, "PurchaseType") {
		t.Fatalf("marshal: %s", s)
	}
	if strings.Contains(s, "<Line>") {
		t.Fatal("PurchaseInvoices XSD has no Line — must not invent lines")
	}
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD: %v\n%s", err, raw)
	}
}

func TestTaxTableStructuralAndXSD(t *testing.T) {
	doc := saftao.MinimalTaxTableFixture()
	if err := doc.MasterFiles.TaxTable.ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "TaxTable") || !strings.Contains(s, "TaxTableEntry") {
		t.Fatalf("marshal: %s", s)
	}
	if !strings.Contains(s, "<TaxPercentage>") {
		t.Fatal("expected TaxPercentage in fixture")
	}
	bad := *doc.MasterFiles.TaxTable
	bad.TaxTableEntry = nil
	if err := bad.ValidateStructural(); err == nil {
		t.Fatal("empty TaxTable must fail-closed")
	}
	entry := doc.MasterFiles.TaxTable.TaxTableEntry[0]
	entry.TaxPercentage = nil
	entry.TaxAmount = nil
	if err := entry.ValidateStructural(); err == nil {
		t.Fatal("TaxPercentage XOR TaxAmount required")
	}
	if !saftao.ValidTaxType(saftao.TaxTypeIVA) || saftao.ValidTaxType("XYZ") {
		t.Fatal("TaxType enum gate")
	}
	if !saftao.ValidTaxTableEntryTaxCode("NOR") || saftao.ValidTaxTableEntryTaxCode("INVALIDCODE") {
		t.Fatal("TaxCode pattern gate")
	}
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD: %v\n%s", err, raw)
	}
}

func TestGeneralLedgerAccountsStructuralAndXSD(t *testing.T) {
	doc := saftao.MinimalGeneralLedgerAccountsFixture()
	if err := doc.MasterFiles.GeneralLedgerAccounts[0].ValidateStructural(); err != nil {
		t.Fatal(err)
	}
	raw, err := saftao.MarshalAuditFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "GeneralLedgerAccounts") || !strings.Contains(s, "AccountID") {
		t.Fatalf("marshal: %s", s)
	}
	if strings.Contains(s, "NumberOfEntries") {
		t.Fatal("MasterFiles/GeneralLedgerAccounts XSD has no NumberOfEntries")
	}
	bad := doc.MasterFiles.GeneralLedgerAccounts[0]
	bad.Account = nil
	if err := bad.ValidateStructural(); err == nil {
		t.Fatal("empty Account must fail-closed")
	}
	if !saftao.ValidGroupingCategory(saftao.GroupingCategoryGR) || saftao.ValidGroupingCategory("XX") {
		t.Fatal("GroupingCategory enum gate")
	}
	if !saftao.ValidGLAccountID("11.1") || saftao.ValidGLAccountID("") {
		t.Fatal("AccountID pattern gate")
	}
	if !saftao.XSDValidatorAvailable() {
		t.Skip("xmllint not available")
	}
	if err := saftao.ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
		t.Fatalf("XSD: %v\n%s", err, raw)
	}
}
