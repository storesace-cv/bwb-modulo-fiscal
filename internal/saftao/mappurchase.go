package saftao

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// PurchaseLedgerRecord is a purchase invoice view for SAF-T mapping (XSD: no Line).
// Missing SAF-T fields → Omissions (no invention). Reasons must not leak NIF/tokens/payloads.
type PurchaseLedgerRecord struct {
	ScopeID    string
	DocumentID string
	IssuedAt   time.Time
	SealedAt   time.Time

	InvoiceNo       string
	Hash            string
	SourceID        string
	PurchaseType    PurchaseType
	SupplierID      string
	TaxPayableCents int64
	NetTotalCents   int64
	GrossTotalCents int64
}

// PurchaseLedgerMapConfig configures scope/period filters for purchase mapping.
type PurchaseLedgerMapConfig struct {
	ScopeID              string
	PeriodStart          Date
	PeriodEnd            Date
	Header               Header
	AllowedPurchaseTypes []PurchaseType
	Suppliers            []Supplier
	ValidateAgainstXSD   bool
	IncludeEmptyPurchase bool
	MaxOmissions         int
}

// MapPurchaseLedgerToExport filters by scope/period, maps complete records to PurchaseInvoices,
// reports omissions, then calls BuildIncrementalExport.
func MapPurchaseLedgerToExport(cfg PurchaseLedgerMapConfig, records []PurchaseLedgerRecord) (*LedgerMapResult, error) {
	if strings.TrimSpace(cfg.ScopeID) == "" {
		return nil, fmt.Errorf("%w: ScopeID obrigatório", ErrValidation)
	}
	if err := cfg.PeriodStart.Validate(); err != nil {
		return nil, fmt.Errorf("%w: PeriodStart", ErrValidation)
	}
	if err := cfg.PeriodEnd.Validate(); err != nil {
		return nil, fmt.Errorf("%w: PeriodEnd", ErrValidation)
	}
	if string(cfg.PeriodStart) > string(cfg.PeriodEnd) {
		return nil, fmt.Errorf("%w: PeriodStart>PeriodEnd", ErrValidation)
	}
	maxOm := cfg.MaxOmissions
	if maxOm <= 0 {
		maxOm = MaxTableEntries
	}

	sorted := append([]PurchaseLedgerRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DocumentID < sorted[j].DocumentID
	})

	var (
		invoices  []PurchaseInvoice
		omissions []Omission
		suppliers []Supplier
		supSeen   = map[string]struct{}{}
	)
	for _, s := range cfg.Suppliers {
		if id := strings.TrimSpace(s.SupplierID); id != "" {
			if _, ok := supSeen[id]; !ok {
				supSeen[id] = struct{}{}
				suppliers = append(suppliers, s)
			}
		}
	}

	for _, rec := range sorted {
		if strings.TrimSpace(rec.ScopeID) != cfg.ScopeID {
			continue
		}
		if rec.IssuedAt.IsZero() {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "issued_at", Reason: "IssuedAt ausente"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		invDate, err := dateFromTime(rec.IssuedAt)
		if err != nil {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "issued_at", Reason: "data inválida"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		if string(invDate) < string(cfg.PeriodStart) || string(invDate) > string(cfg.PeriodEnd) {
			continue
		}
		inv, sup, oms := mapOnePurchaseRecord(rec, invDate)
		if len(oms) > 0 {
			omissions = append(omissions, oms...)
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		invoices = append(invoices, inv)
		if sup.SupplierID != "" {
			if _, ok := supSeen[sup.SupplierID]; !ok {
				supSeen[sup.SupplierID] = struct{}{}
				suppliers = append(suppliers, sup)
			}
		}
		if len(invoices) > MaxTableEntries {
			return nil, fmt.Errorf("%w: purchase invoices excederam MaxTableEntries", ErrValidation)
		}
	}

	hdr := cfg.Header
	hdr.StartDate = string(cfg.PeriodStart)
	hdr.EndDate = string(cfg.PeriodEnd)

	exp, err := BuildIncrementalExport(ExportRequest{
		Header:                      hdr,
		EnabledGroups:               []DocumentGroup{GroupPurchaseInvoices},
		AllowedPurchaseTypes:        cfg.AllowedPurchaseTypes,
		Suppliers:                   suppliers,
		PurchaseInvoices:            invoices,
		IncludeEmptyPurchaseEntries: cfg.IncludeEmptyPurchase,
		ValidateAgainstXSD:          cfg.ValidateAgainstXSD,
	})
	if err != nil {
		return nil, err
	}
	return &LedgerMapResult{
		Export:    exp,
		Mapped:    len(invoices),
		Omitted:   countOmittedDocs(omissions),
		Omissions: omissions,
	}, nil
}

func mapOnePurchaseRecord(rec PurchaseLedgerRecord, invDate Date) (PurchaseInvoice, Supplier, []Omission) {
	var oms []Omission
	omit := func(field, reason string) {
		oms = append(oms, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: field, Reason: reason})
	}
	no := strings.TrimSpace(rec.InvoiceNo)
	if no == "" || len(no) > 60 {
		omit("invoice_no", "InvoiceNo ausente ou inválido (não inventado)")
	}
	hash := strings.TrimSpace(rec.Hash)
	if hash == "" {
		omit("hash", "Hash ausente (PendingHashAlgorithm; não inventado)")
	}
	sourceID := strings.TrimSpace(rec.SourceID)
	if sourceID == "" {
		omit("source_id", "SourceID ausente")
	}
	if !ValidPurchaseType(rec.PurchaseType) {
		omit("purchase_type", "PurchaseType ausente ou fora do XSD")
	}
	supplierID := strings.TrimSpace(rec.SupplierID)
	if supplierID == "" {
		omit("supplier_id", "SupplierID SAF-T ausente (não usar identificador fiscal cru)")
	}
	if rec.GrossTotalCents <= 0 || rec.NetTotalCents < 0 || rec.TaxPayableCents < 0 {
		omit("document_totals", "totais inválidos")
	}
	if len(oms) > 0 {
		return PurchaseInvoice{}, Supplier{}, oms
	}
	inv := PurchaseInvoice{
		InvoiceNo:    no,
		Hash:         hash,
		SourceID:     sourceID,
		InvoiceDate:  invDate,
		PurchaseType: rec.PurchaseType,
		SupplierID:   supplierID,
		DocumentTotals: DocumentTotals{
			TaxPayable: MustMoney2(formatCents(rec.TaxPayableCents)),
			NetTotal:   MustMoney2(formatCents(rec.NetTotalCents)),
			GrossTotal: MustMoney2(formatCents(rec.GrossTotalCents)),
		},
	}
	sup := Supplier{
		SupplierID:           supplierID,
		AccountID:            "Desconhecido",
		SupplierTaxID:        "Desconhecido",
		CompanyName:          "Desconhecido",
		BillingAddress:       &AddressStructure{AddressDetail: "Desconhecido", City: "Desconhecido", Country: "AO"},
		SelfBillingIndicator: 0,
	}
	return inv, sup, nil
}
