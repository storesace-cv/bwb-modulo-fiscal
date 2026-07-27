package saftao

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MovementLedgerRecord is a stock movement view for SAF-T mapping.
// Missing fields → Omissions (no invention). Reasons must not leak NIF/tokens/payloads.
type MovementLedgerRecord struct {
	ScopeID         string
	DocumentID      string
	MovementAt      time.Time
	SealedAt        time.Time
	DocumentNumber  string
	Hash            string
	HashControl     string
	MovementType    MovementType
	SourceID        string
	CustomerID      string // XOR SupplierID
	SupplierID      string
	MovementStatus  MovementStatus
	SourceBilling   SourceBilling
	StatusAt        time.Time
	StartAt         time.Time
	Lines           []MovementLedgerLine
	TaxPayableCents int64
	NetTotalCents   int64
	GrossTotalCents int64
}

// MovementLedgerLine is one stock movement line (scaled qty + cents).
type MovementLedgerLine struct {
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

// MovementLedgerMapConfig configures scope/period for movement mapping.
type MovementLedgerMapConfig struct {
	ScopeID              string
	PeriodStart          Date
	PeriodEnd            Date
	Header               Header
	AllowedMovementTypes []MovementType
	Customers            []Customer
	Suppliers            []Supplier
	Products             []Product
	ValidateAgainstXSD   bool
	IncludeEmptyMovement bool
	TaxTable             *TaxTable
	MaxOmissions         int
}

// MapMovementLedgerToExport filters by scope/period, maps complete records to MovementOfGoods.
func MapMovementLedgerToExport(cfg MovementLedgerMapConfig, records []MovementLedgerRecord) (*LedgerMapResult, error) {
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

	sorted := append([]MovementLedgerRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DocumentID < sorted[j].DocumentID
	})

	var (
		movements []StockMovement
		omissions []Omission
		customers []Customer
		suppliers []Supplier
		products  []Product
		custSeen  = map[string]struct{}{}
		supSeen   = map[string]struct{}{}
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
	for _, s := range cfg.Suppliers {
		if id := strings.TrimSpace(s.SupplierID); id != "" {
			if _, ok := supSeen[id]; !ok {
				supSeen[id] = struct{}{}
				suppliers = append(suppliers, s)
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
		if rec.MovementAt.IsZero() {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "movement_at", Reason: "MovementAt ausente"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		movDate, err := dateFromTime(rec.MovementAt)
		if err != nil {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "movement_at", Reason: "data inválida"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		if string(movDate) < string(cfg.PeriodStart) || string(movDate) > string(cfg.PeriodEnd) {
			continue
		}
		sm, cust, sup, prods, oms := mapOneMovementRecord(rec, movDate)
		if len(oms) > 0 {
			omissions = append(omissions, oms...)
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		movements = append(movements, sm)
		if cust.CustomerID != "" {
			if _, ok := custSeen[cust.CustomerID]; !ok {
				custSeen[cust.CustomerID] = struct{}{}
				customers = append(customers, cust)
			}
		}
		if sup.SupplierID != "" {
			if _, ok := supSeen[sup.SupplierID]; !ok {
				supSeen[sup.SupplierID] = struct{}{}
				suppliers = append(suppliers, sup)
			}
		}
		for _, p := range prods {
			if _, ok := prodSeen[p.ProductCode]; !ok {
				prodSeen[p.ProductCode] = struct{}{}
				products = append(products, p)
			}
		}
		if len(movements) > MaxTableEntries {
			return nil, fmt.Errorf("%w: stock movements excederam MaxTableEntries", ErrValidation)
		}
	}

	hdr := cfg.Header
	hdr.StartDate = string(cfg.PeriodStart)
	hdr.EndDate = string(cfg.PeriodEnd)

	exp, err := BuildIncrementalExport(ExportRequest{
		Header:                     hdr,
		EnabledGroups:              []DocumentGroup{GroupMovementOfGoods},
		AllowedMovementTypes:       cfg.AllowedMovementTypes,
		Customers:                  customers,
		Suppliers:                  suppliers,
		Products:                   products,
		TaxTable:                   cfg.TaxTable,
		StockMovements:             movements,
		IncludeEmptyMovementTotals: cfg.IncludeEmptyMovement,
		ValidateAgainstXSD:         cfg.ValidateAgainstXSD,
	})
	if err != nil {
		return nil, err
	}
	return &LedgerMapResult{
		Export:    exp,
		Mapped:    len(movements),
		Omitted:   countOmittedDocs(omissions),
		Omissions: omissions,
	}, nil
}

func mapOneMovementRecord(rec MovementLedgerRecord, movDate Date) (StockMovement, Customer, Supplier, []Product, []Omission) {
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
	if !ValidMovementType(rec.MovementType) {
		omit("movement_type", "MovementType ausente ou fora do XSD")
	}
	sourceID := strings.TrimSpace(rec.SourceID)
	if sourceID == "" {
		omit("source_id", "SourceID ausente")
	}
	hasCust := strings.TrimSpace(rec.CustomerID) != ""
	hasSupp := strings.TrimSpace(rec.SupplierID) != ""
	if hasCust == hasSupp {
		omit("party", "CustomerID XOR SupplierID obrigatório")
	}
	if !ValidMovementStatus(rec.MovementStatus) {
		omit("movement_status", "MovementStatus ausente ou inválido")
	}
	if !ValidSourceBilling(rec.SourceBilling) {
		omit("source_billing", "SourceBilling ausente ou inválido")
	}
	if len(rec.Lines) == 0 {
		omit("lines", "movimento sem linhas")
	}
	if len(oms) > 0 {
		return StockMovement{}, Customer{}, Supplier{}, nil, oms
	}

	statusAt := rec.StatusAt
	if statusAt.IsZero() {
		statusAt = rec.SealedAt
	}
	if statusAt.IsZero() {
		statusAt = rec.MovementAt
	}
	statusDT, err := dateTimeFromTime(statusAt)
	if err != nil {
		omit("status_at", "instante inválido")
		return StockMovement{}, Customer{}, Supplier{}, nil, oms
	}
	entryAt := rec.SealedAt
	if entryAt.IsZero() {
		entryAt = rec.MovementAt
	}
	entryDT, err := dateTimeFromTime(entryAt)
	if err != nil {
		omit("system_entry_date", "instante inválido")
		return StockMovement{}, Customer{}, Supplier{}, nil, oms
	}
	startAt := rec.StartAt
	if startAt.IsZero() {
		startAt = rec.MovementAt
	}
	startDT, err := dateTimeFromTime(startAt)
	if err != nil {
		omit("movement_start_time", "instante inválido")
		return StockMovement{}, Customer{}, Supplier{}, nil, oms
	}

	var lines []StockMovementLine
	var products []Product
	for _, ln := range rec.Lines {
		if ln.LineNo <= 0 {
			omit("line_no", "LineNo inválido")
			return StockMovement{}, Customer{}, Supplier{}, nil, oms
		}
		pc := strings.TrimSpace(ln.ProductCode)
		if pc == "" {
			omit("product_code", "ProductCode ausente")
			return StockMovement{}, Customer{}, Supplier{}, nil, oms
		}
		uom := strings.TrimSpace(ln.UnitOfMeasure)
		if uom == "" {
			omit("unit_of_measure", "UnitOfMeasure ausente")
			return StockMovement{}, Customer{}, Supplier{}, nil, oms
		}
		qty, err := formatQuantity(ln.QuantityScaled)
		if err != nil {
			omit("quantity", "quantidade inválida")
			return StockMovement{}, Customer{}, Supplier{}, nil, oms
		}
		hasDebit := ln.DebitCents > 0
		hasCredit := ln.CreditCents > 0
		if hasDebit == hasCredit {
			omit("line_amount", "Debit XOR Credit em cents")
			return StockMovement{}, Customer{}, Supplier{}, nil, oms
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
		var tax *MovementTax
		if taxPct != "" || taxCode != "" || taxType != "" {
			if taxType == "" || taxCode == "" || taxPct == "" {
				omit("tax", "Tax incompleto (não inventar)")
				return StockMovement{}, Customer{}, Supplier{}, nil, oms
			}
			tax = &MovementTax{TaxType: taxType, TaxCode: taxCode, TaxPercentage: taxPct}
		}
		desc := strings.TrimSpace(ln.Description)
		if desc == "" {
			desc = pc
		}
		lines = append(lines, StockMovementLine{
			LineNumber:         strconv.Itoa(ln.LineNo),
			ProductCode:        pc,
			ProductDescription: desc,
			Quantity:           MustDecimal(qty),
			UnitOfMeasure:      uom,
			UnitPrice:          MustDecimal(formatCents(ln.UnitPriceCents)),
			Description:        desc,
			DebitAmount:        debit,
			CreditAmount:       credit,
			Tax:                tax,
		})
		products = append(products, Product{
			ProductType:        "P",
			ProductCode:        pc,
			ProductDescription: desc,
			ProductNumberCode:  pc,
		})
	}
	if rec.GrossTotalCents <= 0 {
		omit("document_totals", "totais inválidos")
		return StockMovement{}, Customer{}, Supplier{}, nil, oms
	}

	sm := StockMovement{
		DocumentNumber: docNo,
		DocumentStatus: MovementDocumentStatus{
			MovementStatus:     rec.MovementStatus,
			MovementStatusDate: statusDT,
			SourceID:           sourceID,
			SourceBilling:      rec.SourceBilling,
		},
		Hash:              hash,
		HashControl:       hashCtrl,
		MovementDate:      movDate,
		MovementType:      rec.MovementType,
		SystemEntryDate:   entryDT,
		CustomerID:        strings.TrimSpace(rec.CustomerID),
		SupplierID:        strings.TrimSpace(rec.SupplierID),
		SourceID:          sourceID,
		MovementStartTime: startDT,
		Line:              lines,
		DocumentTotals: DocumentTotals{
			TaxPayable: MustMoney2(formatCents(rec.TaxPayableCents)),
			NetTotal:   MustMoney2(formatCents(rec.NetTotalCents)),
			GrossTotal: MustMoney2(formatCents(rec.GrossTotalCents)),
		},
	}
	var cust Customer
	var sup Supplier
	if sm.CustomerID != "" {
		cust = Customer{
			CustomerID: sm.CustomerID, AccountID: "Desconhecido", CustomerTaxID: "Desconhecido",
			CompanyName: "Desconhecido", BillingAddress: &AddressStructure{AddressDetail: "Desconhecido", City: "Desconhecido", Country: "AO"},
		}
	}
	if sm.SupplierID != "" {
		sup = Supplier{
			SupplierID: sm.SupplierID, AccountID: "Desconhecido", SupplierTaxID: "Desconhecido",
			CompanyName: "Desconhecido", BillingAddress: &AddressStructure{AddressDetail: "Desconhecido", City: "Desconhecido", Country: "AO"},
		}
	}
	return sm, cust, sup, products, nil
}
