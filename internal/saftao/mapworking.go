package saftao

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// WorkingLedgerRecord is a work document view for SAF-T mapping.
// Missing fields → Omissions (no invention). Reasons must not leak NIF/tokens/payloads.
type WorkingLedgerRecord struct {
	ScopeID         string
	DocumentID      string
	WorkAt          time.Time
	SealedAt        time.Time
	DocumentNumber  string
	Hash            string
	HashControl     string
	WorkType        WorkType
	SourceID        string
	CustomerID      string
	WorkStatus      WorkStatus
	SourceBilling   SourceBilling
	StatusAt        time.Time
	Lines           []WorkingLedgerLine
	TaxPayableCents int64
	NetTotalCents   int64
	GrossTotalCents int64
}

// WorkingLedgerLine is one work document line (scaled qty + cents).
type WorkingLedgerLine struct {
	LineNo         int
	ProductCode    string
	Description    string
	QuantityScaled int64
	UnitOfMeasure  string
	UnitPriceCents int64
	DebitCents     int64
	CreditCents    int64
	TaxType        string
	TaxCode        string
	TaxPercentage  string
}

// WorkingLedgerMapConfig configures scope/period for working-document mapping.
type WorkingLedgerMapConfig struct {
	ScopeID             string
	PeriodStart         Date
	PeriodEnd           Date
	Header              Header
	AllowedWorkTypes    []WorkType
	Customers           []Customer
	Products            []Product
	ValidateAgainstXSD  bool
	IncludeEmptyWorking bool
	TaxTable            *TaxTable
	MaxOmissions        int
}

// MapWorkingLedgerToExport filters by scope/period, maps complete records to WorkingDocuments.
func MapWorkingLedgerToExport(cfg WorkingLedgerMapConfig, records []WorkingLedgerRecord) (*LedgerMapResult, error) {
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

	sorted := append([]WorkingLedgerRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DocumentID < sorted[j].DocumentID
	})

	var (
		docs      []WorkDocument
		omissions []Omission
		customers []Customer
		products  []Product
		custSeen  = map[string]struct{}{}
		prodSeen  = map[string]struct{}{}
	)
	for _, c := range cfg.Customers {
		if id := strings.TrimSpace(c.CustomerID); id != "" {
			if _, ok := custSeen[id]; !ok {
				custSeen[id] = struct{}{}
				customers = append(customers, c)
			}
		}
	}
	for _, p := range cfg.Products {
		if id := strings.TrimSpace(p.ProductCode); id != "" {
			if _, ok := prodSeen[id]; !ok {
				prodSeen[id] = struct{}{}
				products = append(products, p)
			}
		}
	}

	for _, rec := range sorted {
		if strings.TrimSpace(rec.ScopeID) != cfg.ScopeID {
			continue
		}
		if rec.WorkAt.IsZero() {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "work_at", Reason: "WorkAt ausente"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		workDate, err := dateFromTime(rec.WorkAt)
		if err != nil {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "work_at", Reason: "data inválida"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		if string(workDate) < string(cfg.PeriodStart) || string(workDate) > string(cfg.PeriodEnd) {
			continue
		}
		wd, cust, prods, oms := mapOneWorkingRecord(rec, workDate)
		if len(oms) > 0 {
			omissions = append(omissions, oms...)
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		docs = append(docs, wd)
		if cust.CustomerID != "" {
			if _, ok := custSeen[cust.CustomerID]; !ok {
				custSeen[cust.CustomerID] = struct{}{}
				customers = append(customers, cust)
			}
		}
		for _, p := range prods {
			if _, ok := prodSeen[p.ProductCode]; !ok {
				prodSeen[p.ProductCode] = struct{}{}
				products = append(products, p)
			}
		}
		if len(docs) > MaxTableEntries {
			return nil, fmt.Errorf("%w: work documents excederam MaxTableEntries", ErrValidation)
		}
	}

	hdr := cfg.Header
	hdr.StartDate = string(cfg.PeriodStart)
	hdr.EndDate = string(cfg.PeriodEnd)

	exp, err := BuildIncrementalExport(ExportRequest{
		Header:                    hdr,
		EnabledGroups:             []DocumentGroup{GroupWorkingDocuments},
		AllowedWorkTypes:          cfg.AllowedWorkTypes,
		Customers:                 customers,
		Products:                  products,
		TaxTable:                  cfg.TaxTable,
		WorkDocuments:             docs,
		IncludeEmptyWorkingTotals: cfg.IncludeEmptyWorking,
		ValidateAgainstXSD:        cfg.ValidateAgainstXSD,
	})
	if err != nil {
		return nil, err
	}
	return &LedgerMapResult{
		Export:    exp,
		Mapped:    len(docs),
		Omitted:   countOmittedDocs(omissions),
		Omissions: omissions,
	}, nil
}

func mapOneWorkingRecord(rec WorkingLedgerRecord, workDate Date) (WorkDocument, Customer, []Product, []Omission) {
	var oms []Omission
	omit := func(field, reason string) {
		oms = append(oms, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: field, Reason: reason})
	}
	docNo := strings.TrimSpace(rec.DocumentNumber)
	if docNo == "" {
		omit("document_number", "DocumentNumber ausente (não inventado)")
	} else if err := ValidateDocumentNumber(docNo); err != nil {
		omit("document_number", "DocumentNumber inválido")
	}
	hash := strings.TrimSpace(rec.Hash)
	if hash == "" {
		omit("hash", "Hash ausente (PendingHashAlgorithm; não inventado)")
	}
	hashCtrl := strings.TrimSpace(rec.HashControl)
	if hashCtrl == "" {
		omit("hash_control", "HashControl ausente")
	}
	if !ValidWorkType(rec.WorkType) {
		omit("work_type", "WorkType ausente ou fora do XSD")
	}
	sourceID := strings.TrimSpace(rec.SourceID)
	if sourceID == "" {
		omit("source_id", "SourceID ausente")
	}
	customerID := strings.TrimSpace(rec.CustomerID)
	if customerID == "" {
		omit("customer_id", "CustomerID SAF-T ausente (não usar identificador fiscal cru)")
	}
	if !ValidWorkStatus(rec.WorkStatus) {
		omit("work_status", "WorkStatus ausente ou inválido")
	}
	if !ValidSourceBilling(rec.SourceBilling) {
		omit("source_billing", "SourceBilling ausente ou inválido")
	}
	if len(rec.Lines) == 0 {
		omit("lines", "documento sem linhas")
	}
	if len(oms) > 0 {
		return WorkDocument{}, Customer{}, nil, oms
	}

	statusAt := rec.StatusAt
	if statusAt.IsZero() {
		statusAt = rec.SealedAt
	}
	if statusAt.IsZero() {
		statusAt = rec.WorkAt
	}
	statusDT, err := dateTimeFromTime(statusAt)
	if err != nil {
		omit("status_at", "instante inválido")
		return WorkDocument{}, Customer{}, nil, oms
	}
	entryAt := rec.SealedAt
	if entryAt.IsZero() {
		entryAt = rec.WorkAt
	}
	entryDT, err := dateTimeFromTime(entryAt)
	if err != nil {
		omit("system_entry_date", "instante inválido")
		return WorkDocument{}, Customer{}, nil, oms
	}

	var lines []WorkDocumentLine
	var products []Product
	for _, ln := range rec.Lines {
		if ln.LineNo <= 0 {
			omit("line_no", "LineNo inválido")
			return WorkDocument{}, Customer{}, nil, oms
		}
		pc := strings.TrimSpace(ln.ProductCode)
		if pc == "" {
			omit("product_code", "ProductCode ausente")
			return WorkDocument{}, Customer{}, nil, oms
		}
		uom := strings.TrimSpace(ln.UnitOfMeasure)
		if uom == "" {
			omit("unit_of_measure", "UnitOfMeasure ausente")
			return WorkDocument{}, Customer{}, nil, oms
		}
		qty, err := formatQuantity(ln.QuantityScaled)
		if err != nil {
			omit("quantity", "quantidade inválida")
			return WorkDocument{}, Customer{}, nil, oms
		}
		hasDebit := ln.DebitCents > 0
		hasCredit := ln.CreditCents > 0
		if hasDebit == hasCredit {
			omit("line_amount", "Debit XOR Credit em cents")
			return WorkDocument{}, Customer{}, nil, oms
		}
		var debit, credit *Money2
		if hasDebit {
			m := MustMoney2(formatCents(ln.DebitCents))
			debit = &m
		}
		if hasCredit {
			m := MustMoney2(formatCents(ln.CreditCents))
			credit = &m
		}
		taxPct := strings.TrimSpace(ln.TaxPercentage)
		taxCode := strings.TrimSpace(ln.TaxCode)
		taxType := strings.TrimSpace(ln.TaxType)
		var tax *Tax
		if taxPct != "" || taxCode != "" || taxType != "" {
			if taxType == "" || taxCode == "" || taxPct == "" {
				omit("tax", "Tax incompleto (não inventar)")
				return WorkDocument{}, Customer{}, nil, oms
			}
			pct := taxPct
			tax = &Tax{TaxType: taxType, TaxCode: taxCode, TaxPercentage: &pct}
		}
		desc := strings.TrimSpace(ln.Description)
		if desc == "" {
			desc = pc
		}
		lines = append(lines, WorkDocumentLine{
			LineNumber:         strconv.Itoa(ln.LineNo),
			ProductCode:        pc,
			ProductDescription: desc,
			Quantity:           MustDecimal(qty),
			UnitOfMeasure:      uom,
			UnitPrice:          MustDecimal(formatCents(ln.UnitPriceCents)),
			TaxPointDate:       workDate,
			Description:        desc,
			DebitAmount:        debit,
			CreditAmount:       credit,
			Tax:                tax,
		})
		products = append(products, Product{
			ProductType: "S", ProductCode: pc, ProductDescription: desc, ProductNumberCode: pc,
		})
	}
	if rec.GrossTotalCents <= 0 {
		omit("document_totals", "totais inválidos")
		return WorkDocument{}, Customer{}, nil, oms
	}

	wd := WorkDocument{
		DocumentNumber: docNo,
		DocumentStatus: WorkDocumentStatus{
			WorkStatus:     rec.WorkStatus,
			WorkStatusDate: statusDT,
			SourceID:       sourceID,
			SourceBilling:  rec.SourceBilling,
		},
		Hash:            hash,
		HashControl:     hashCtrl,
		WorkDate:        workDate,
		WorkType:        rec.WorkType,
		SourceID:        sourceID,
		SystemEntryDate: entryDT,
		CustomerID:      customerID,
		Line:            lines,
		DocumentTotals: DocumentTotals{
			TaxPayable: MustMoney2(formatCents(rec.TaxPayableCents)),
			NetTotal:   MustMoney2(formatCents(rec.NetTotalCents)),
			GrossTotal: MustMoney2(formatCents(rec.GrossTotalCents)),
		},
	}
	cust := Customer{
		CustomerID: customerID, AccountID: "Desconhecido", CustomerTaxID: "Desconhecido",
		CompanyName: "Desconhecido", BillingAddress: &AddressStructure{AddressDetail: "Desconhecido", City: "Desconhecido", Country: "AO"},
	}
	return wd, cust, products, nil
}
