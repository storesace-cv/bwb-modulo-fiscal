package saftao_test

import (
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestIncrementalExportPeriodFilterAndHash(t *testing.T) {
	base := saftao.MinimalSalesInvoiceFixture()
	invOut := base.SourceDocuments.SalesInvoices.Invoice[0]
	invOut.InvoiceNo = "FT S001/2"
	invOut.InvoiceDate = saftao.MustDate("2025-12-31") // outside 2026 period

	req := saftao.ExportRequest{
		Header:                  *base.Header,
		EnabledGroups:           []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		AllowedInvoiceTypes:     []saftao.InvoiceType{saftao.InvoiceTypeFT},
		Customers:               base.MasterFiles.Customer,
		Products:                base.MasterFiles.Product,
		Invoices:                append([]saftao.Invoice{}, base.SourceDocuments.SalesInvoices.Invoice[0], invOut),
		IncludeEmptySalesTotals: true,
		ValidateAgainstXSD:      saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.NumberOfInvoices != 1 {
		t.Fatalf("want 1 in-period invoice, got %d", res.NumberOfInvoices)
	}
	if res.TotalCredit.String() != "100.00" {
		t.Fatalf("TotalCredit %q", res.TotalCredit)
	}
	if len(res.SHA256) != 64 {
		t.Fatalf("sha256 %q", res.SHA256)
	}
	if !strings.Contains(string(res.XML), "FT S001/1") {
		t.Fatal("missing in-period invoice")
	}
	if strings.Contains(string(res.XML), "FT S001/2") {
		t.Fatal("out-of-period invoice must be filtered")
	}
	if req.ValidateAgainstXSD && !res.XSDChecked {
		t.Fatal("expected XSDChecked")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 || string(res.XML) != string(res2.XML) {
		t.Fatal("export must be deterministic")
	}
}

func TestIncrementalExportFailClosedAllowedTypes(t *testing.T) {
	base := saftao.MinimalSalesInvoiceFixture()
	req := saftao.ExportRequest{
		Header:              *base.Header,
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		AllowedInvoiceTypes: nil, // fail-closed
		Customers:           base.MasterFiles.Customer,
		Products:            base.MasterFiles.Product,
		Invoices:            base.SourceDocuments.SalesInvoices.Invoice,
	}
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("empty AllowedInvoiceTypes must reject invoices")
	}
}

func TestIncrementalExportRejectsUnknownGroup(t *testing.T) {
	base := saftao.MinimalSalesInvoiceFixture()
	req := saftao.ExportRequest{
		Header:        *base.Header,
		EnabledGroups: []saftao.DocumentGroup{"Invented"},
		Customers:     nil,
		Products:      nil,
		Invoices:      nil,
	}
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("invented group")
	}
}

func TestIncrementalExportWithTaxTable(t *testing.T) {
	base := saftao.MinimalSalesInvoiceFixture()
	tax := saftao.MinimalTaxTableFixture().MasterFiles.TaxTable
	req := saftao.ExportRequest{
		Header:              *base.Header,
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		AllowedInvoiceTypes: []saftao.InvoiceType{saftao.InvoiceTypeFT},
		Customers:           base.MasterFiles.Customer,
		Products:            base.MasterFiles.Product,
		TaxTable:            tax,
		Invoices:            base.SourceDocuments.SalesInvoices.Invoice,
		ValidateAgainstXSD:  saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.XML), "TaxTable") || !strings.Contains(string(res.XML), "TaxTableEntry") {
		t.Fatalf("TaxTable missing from XML: %s", res.XML)
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingTaxTableSemantics {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PendingTaxTableSemantics")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 {
		t.Fatal("TaxTable export must be deterministic")
	}

	badTax := *tax
	badTax.TaxTableEntry = []saftao.TaxTableEntry{tax.TaxTableEntry[1]} // ISE only — invoice uses NOR
	req.TaxTable = &badTax
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("invoice Tax without TaxTableEntry must fail-closed")
	}
}

func TestIncrementalExportPayments(t *testing.T) {
	base := saftao.MinimalPaymentsFixture()
	payOut := base.SourceDocuments.Payments.Payment[0]
	payOut.PaymentRefNo = "RC S001/2"
	payOut.TransactionDate = saftao.MustDate("2025-12-31")

	req := saftao.ExportRequest{
		Header:              *base.Header,
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupPayments},
		AllowedPaymentTypes: []saftao.PaymentType{saftao.PaymentTypeRC},
		Customers:           base.MasterFiles.Customer,
		Payments:            append([]saftao.Payment{}, base.SourceDocuments.Payments.Payment[0], payOut),
		ValidateAgainstXSD:  saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.NumberOfPayments != 1 {
		t.Fatalf("want 1 in-period payment, got %d", res.NumberOfPayments)
	}
	if res.PaymentTotalCredit.String() != "114.00" {
		t.Fatalf("PaymentTotalCredit %q", res.PaymentTotalCredit)
	}
	if !strings.Contains(string(res.XML), "RC S001/1") {
		t.Fatal("missing in-period payment")
	}
	if strings.Contains(string(res.XML), "RC S001/2") {
		t.Fatal("out-of-period payment must be filtered")
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingPaymentTypeSemantics {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PendingPaymentTypeSemantics")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 {
		t.Fatal("payments export must be deterministic")
	}

	req.AllowedPaymentTypes = nil
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("empty AllowedPaymentTypes must reject payments")
	}
}

func TestIncrementalExportPurchaseInvoices(t *testing.T) {
	base := saftao.MinimalPurchaseInvoicesFixture()
	out := base.SourceDocuments.PurchaseInvoices.Invoice[0]
	out.InvoiceNo = "FORN-FT-2025-9999"
	out.InvoiceDate = saftao.MustDate("2025-12-31")

	req := saftao.ExportRequest{
		Header:               *base.Header,
		EnabledGroups:        []saftao.DocumentGroup{saftao.GroupPurchaseInvoices},
		AllowedPurchaseTypes: []saftao.PurchaseType{saftao.PurchaseTypeFT},
		Suppliers:            base.MasterFiles.Supplier,
		PurchaseInvoices:     append([]saftao.PurchaseInvoice{}, base.SourceDocuments.PurchaseInvoices.Invoice[0], out),
		ValidateAgainstXSD:   saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.NumberOfPurchaseInvoices != 1 {
		t.Fatalf("want 1 in-period purchase, got %d", res.NumberOfPurchaseInvoices)
	}
	s := string(res.XML)
	if !strings.Contains(s, "FORN-FT-2026-0001") {
		t.Fatal("missing in-period purchase")
	}
	if strings.Contains(s, "FORN-FT-2025-9999") {
		t.Fatal("out-of-period purchase must be filtered")
	}
	if strings.Contains(s, "<Line>") {
		t.Fatal("PurchaseInvoices must not invent Line")
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingPurchaseTypeSemantics {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PendingPurchaseTypeSemantics")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 {
		t.Fatal("purchase export must be deterministic")
	}
	req.AllowedPurchaseTypes = nil
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("empty AllowedPurchaseTypes must reject purchases")
	}
}

func TestIncrementalExportMovementOfGoods(t *testing.T) {
	base := saftao.MinimalMovementOfGoodsFixture()
	out := base.SourceDocuments.MovementOfGoods.StockMovement[0]
	out.DocumentNumber = "GR S001/2"
	out.MovementDate = saftao.MustDate("2025-12-31")

	req := saftao.ExportRequest{
		Header:               *base.Header,
		EnabledGroups:        []saftao.DocumentGroup{saftao.GroupMovementOfGoods},
		AllowedMovementTypes: []saftao.MovementType{saftao.MovementTypeGR},
		Customers:            base.MasterFiles.Customer,
		Products:             base.MasterFiles.Product,
		StockMovements:       append([]saftao.StockMovement{}, base.SourceDocuments.MovementOfGoods.StockMovement[0], out),
		ValidateAgainstXSD:   saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.NumberOfStockMovements != 1 || res.NumberOfMovementLines != 1 {
		t.Fatalf("want 1 movement/1 line, got %d/%d", res.NumberOfStockMovements, res.NumberOfMovementLines)
	}
	if res.TotalQuantityIssued.String() != "1" {
		t.Fatalf("TotalQuantityIssued %q", res.TotalQuantityIssued)
	}
	s := string(res.XML)
	if !strings.Contains(s, "GR S001/1") || strings.Contains(s, "GR S001/2") {
		t.Fatalf("period filter failed: %s", s)
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingMovementTypeSemantics {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PendingMovementTypeSemantics")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 {
		t.Fatal("movement export must be deterministic")
	}
	req.AllowedMovementTypes = nil
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("empty AllowedMovementTypes must reject movements")
	}
}

func TestIncrementalExportWorkingDocuments(t *testing.T) {
	base := saftao.MinimalWorkingDocumentsFixture()
	out := base.SourceDocuments.WorkingDocuments.WorkDocument[0]
	out.DocumentNumber = "PF S001/2"
	out.WorkDate = saftao.MustDate("2025-12-31")

	req := saftao.ExportRequest{
		Header:             *base.Header,
		EnabledGroups:      []saftao.DocumentGroup{saftao.GroupWorkingDocuments},
		AllowedWorkTypes:   []saftao.WorkType{saftao.WorkTypePF},
		Customers:          base.MasterFiles.Customer,
		Products:           base.MasterFiles.Product,
		WorkDocuments:      append([]saftao.WorkDocument{}, base.SourceDocuments.WorkingDocuments.WorkDocument[0], out),
		ValidateAgainstXSD: saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.NumberOfWorkDocuments != 1 {
		t.Fatalf("want 1 work doc, got %d", res.NumberOfWorkDocuments)
	}
	s := string(res.XML)
	if !strings.Contains(s, "PF S001/1") || strings.Contains(s, "PF S001/2") {
		t.Fatalf("period filter failed: %s", s)
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingWorkTypeSemantics {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PendingWorkTypeSemantics")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 {
		t.Fatal("working export must be deterministic")
	}
	req.AllowedWorkTypes = nil
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("empty AllowedWorkTypes must reject work documents")
	}
}

func TestIncrementalExportGeneralLedgerAccounts(t *testing.T) {
	base := saftao.MinimalSalesInvoiceFixture()
	gla := saftao.MinimalGeneralLedgerAccountsFixture().MasterFiles.GeneralLedgerAccounts
	req := saftao.ExportRequest{
		Header:                *base.Header,
		EnabledGroups:         []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		AllowedInvoiceTypes:   []saftao.InvoiceType{saftao.InvoiceTypeFT},
		Customers:             base.MasterFiles.Customer,
		Products:              base.MasterFiles.Product,
		GeneralLedgerAccounts: gla,
		Invoices:              base.SourceDocuments.SalesInvoices.Invoice,
		ValidateAgainstXSD:    saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.XML), "GeneralLedgerAccounts") || !strings.Contains(string(res.XML), "GroupingCategory") {
		t.Fatalf("GL missing: %s", res.XML)
	}
	found := false
	for _, p := range res.PendingRegulatory {
		if p == saftao.PendingGLAccountSemantics {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PendingGLAccountSemantics")
	}
	res2, err := saftao.BuildIncrementalExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != res2.SHA256 {
		t.Fatal("GL export must be deterministic")
	}
	bad := gla[0]
	bad.Account[1].GroupingCode = "MISSING"
	req.GeneralLedgerAccounts = []saftao.GeneralLedgerAccounts{bad}
	if _, err := saftao.BuildIncrementalExport(req); err == nil {
		t.Fatal("GroupingCode without AccountID must fail-closed")
	}
}
