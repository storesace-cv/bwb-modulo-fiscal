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
