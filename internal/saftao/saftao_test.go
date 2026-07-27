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
	// Never claim certification in payload comments / fields.
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
