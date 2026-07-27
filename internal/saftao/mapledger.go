package saftao

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SalesLedgerRecord is a sealed sales document view for SAF-T mapping.
// Persistence adapters fill this; missing required SAF-T fields produce Omissions (no invention).
type SalesLedgerRecord struct {
	ScopeID       string
	DocumentID    string
	DocumentType  string // ledger: invoice | credit_note
	SeriesCode    string
	FiscalSeq     int64
	IssuedAt      time.Time
	SealedAt      time.Time
	SellerTaxID   string
	SellerName    string
	CustomerTaxID string
	CustomerName  string
	Lines         []SalesLedgerLine

	// Optional SAF-T enrichments — empty triggers omission (never invent Hash algorithm).
	InvoiceNo         string
	Hash              string
	HashControl       string
	SourceID          string
	CustomerID        string
	ProductCodeByLine map[int]string // line_no → ProductCode
	UnitOfMeasure     string
	TaxPercentage     string // when TaxAmount not used
}

// SalesLedgerLine is one persisted line (scaled ints — no float).
type SalesLedgerLine struct {
	LineNo         int
	Description    string
	QuantityScaled int64 // e.g. milli-units; ScaleQuantity divides by QuantityScale
	UnitPriceCents int64
	TaxCode        string
}

// QuantityScale is the fixed divisor for QuantityScaled (matches quantity.Factor = 10000).
const QuantityScale int64 = 10000

// Omission records why a document/field was excluded from the export payload.
type Omission struct {
	ScopeID    string
	DocumentID string
	Field      string
	Reason     string
}

// LedgerMapConfig configures period/scope filters and export allowlists.
type LedgerMapConfig struct {
	ScopeID             string
	PeriodStart         Date
	PeriodEnd           Date
	Header              Header
	AllowedInvoiceTypes []InvoiceType
	EnabledGroups       []DocumentGroup
	ValidateAgainstXSD  bool
	IncludeEmptySales   bool
	// TaxTable optional MasterFiles pass-through (caller-supplied; never invented).
	TaxTable *TaxTable
	// MaxOmissions caps omission report size (fail-closed if exceeded).
	MaxOmissions int
}

// LedgerMapResult is the deterministic map outcome (structural only ≠ AGT).
type LedgerMapResult struct {
	Export    *ExportResult
	Mapped    int
	Omitted   int
	Omissions []Omission
}

// MapSalesLedgerToExport filters by scope/period, maps complete records to SalesInvoices,
// reports omissions for incomplete ones, then calls BuildIncrementalExport.
func MapSalesLedgerToExport(cfg LedgerMapConfig, records []SalesLedgerRecord) (*LedgerMapResult, error) {
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

	// Deterministic order
	sorted := append([]SalesLedgerRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].DocumentID != sorted[j].DocumentID {
			return sorted[i].DocumentID < sorted[j].DocumentID
		}
		return sorted[i].FiscalSeq < sorted[j].FiscalSeq
	})

	var (
		invoices  []Invoice
		customers []Customer
		products  []Product
		omissions []Omission
		custSeen  = map[string]struct{}{}
		prodSeen  = map[string]struct{}{}
	)

	for _, rec := range sorted {
		if rec.ScopeID != cfg.ScopeID {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "scope_id", Reason: "fora do scope do export"})
			continue
		}
		issuedDate, err := dateFromTime(rec.IssuedAt)
		if err != nil {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "issued_at", Reason: err.Error()})
			continue
		}
		if string(issuedDate) < string(cfg.PeriodStart) || string(issuedDate) > string(cfg.PeriodEnd) {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "issued_at", Reason: "fora do período"})
			continue
		}
		inv, cust, prods, oms := mapOneSalesRecord(rec, issuedDate)
		if len(oms) > 0 {
			omissions = append(omissions, oms...)
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		invoices = append(invoices, inv)
		if _, ok := custSeen[cust.CustomerID]; !ok {
			custSeen[cust.CustomerID] = struct{}{}
			customers = append(customers, cust)
		}
		for _, p := range prods {
			if _, ok := prodSeen[p.ProductCode]; !ok {
				prodSeen[p.ProductCode] = struct{}{}
				products = append(products, p)
			}
		}
		if len(invoices) > MaxTableEntries {
			return nil, fmt.Errorf("%w: invoices excederam MaxTableEntries", ErrValidation)
		}
	}

	if len(omissions) > maxOm {
		return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
	}

	hdr := cfg.Header
	hdr.StartDate = string(cfg.PeriodStart)
	hdr.EndDate = string(cfg.PeriodEnd)

	groups := cfg.EnabledGroups
	if len(groups) == 0 {
		groups = []DocumentGroup{GroupSalesInvoices}
	}

	exp, err := BuildIncrementalExport(ExportRequest{
		Header:                  hdr,
		EnabledGroups:           groups,
		AllowedInvoiceTypes:     cfg.AllowedInvoiceTypes,
		Customers:               customers,
		Products:                products,
		TaxTable:                cfg.TaxTable,
		Invoices:                invoices,
		IncludeEmptySalesTotals: cfg.IncludeEmptySales,
		ValidateAgainstXSD:      cfg.ValidateAgainstXSD,
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

func countOmittedDocs(oms []Omission) int {
	seen := map[string]struct{}{}
	for _, o := range oms {
		key := o.ScopeID + "|" + o.DocumentID
		if o.DocumentID == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	return len(seen)
}

func mapOneSalesRecord(rec SalesLedgerRecord, issuedDate Date) (Invoice, Customer, []Product, []Omission) {
	var oms []Omission
	omit := func(field, reason string) {
		oms = append(oms, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: field, Reason: reason})
	}

	invType, err := ledgerDocTypeToInvoiceType(rec.DocumentType)
	if err != nil {
		omit("document_type", err.Error())
		return Invoice{}, Customer{}, nil, oms
	}
	invoiceNo := strings.TrimSpace(rec.InvoiceNo)
	if invoiceNo == "" {
		invoiceNo, err = deriveInvoiceNo(invType, rec.SeriesCode, rec.FiscalSeq)
		if err != nil {
			omit("invoice_no", err.Error())
			return Invoice{}, Customer{}, nil, oms
		}
	}
	if err := ValidateInvoiceNo(invoiceNo); err != nil {
		omit("invoice_no", err.Error())
		return Invoice{}, Customer{}, nil, oms
	}
	hash := strings.TrimSpace(rec.Hash)
	if hash == "" {
		omit("hash", "Hash SAF-T ausente no livro (PendingHashAlgorithm; não inventado)")
	}
	hashCtrl := strings.TrimSpace(rec.HashControl)
	if hashCtrl == "" {
		omit("hash_control", "HashControl ausente no livro")
	}
	sourceID := strings.TrimSpace(rec.SourceID)
	if sourceID == "" {
		omit("source_id", "SourceID ausente no livro")
	}
	customerID := strings.TrimSpace(rec.CustomerID)
	if customerID == "" {
		omit("customer_id", "CustomerID SAF-T ausente (não usar NIF cru sem mapping)")
	}
	uom := strings.TrimSpace(rec.UnitOfMeasure)
	if uom == "" {
		omit("unit_of_measure", "UnitOfMeasure ausente")
	}
	if len(rec.Lines) == 0 {
		omit("lines", "documento sem linhas")
	}
	if len(oms) > 0 {
		return Invoice{}, Customer{}, nil, oms
	}

	sealedDT, err := dateTimeFromTime(rec.SealedAt)
	if err != nil {
		omit("sealed_at", err.Error())
		return Invoice{}, Customer{}, nil, oms
	}

	var lines []InvoiceLine
	var products []Product
	var netCents int64
	for _, ln := range rec.Lines {
		if ln.LineNo <= 0 {
			omit("line_no", "LineNo inválido")
			return Invoice{}, Customer{}, nil, oms
		}
		pc := ""
		if rec.ProductCodeByLine != nil {
			pc = rec.ProductCodeByLine[ln.LineNo]
		}
		if strings.TrimSpace(pc) == "" {
			omit("product_code", fmt.Sprintf("linha %d sem ProductCode", ln.LineNo))
			return Invoice{}, Customer{}, nil, oms
		}
		qty, err := formatQuantity(ln.QuantityScaled)
		if err != nil {
			omit("quantity", err.Error())
			return Invoice{}, Customer{}, nil, oms
		}
		unitPrice := formatCents(ln.UnitPriceCents)
		lineTotal := (ln.QuantityScaled * ln.UnitPriceCents) / QuantityScale
		netCents += lineTotal
		amt := MustMoney2(formatCents(lineTotal))
		taxPct := strings.TrimSpace(rec.TaxPercentage)
		if taxPct == "" {
			omit("tax_percentage", "TaxPercentage ausente (não inventar taxa)")
			return Invoice{}, Customer{}, nil, oms
		}
		lines = append(lines, InvoiceLine{
			LineNumber:         strconv.Itoa(ln.LineNo),
			ProductCode:        pc,
			ProductDescription: ln.Description,
			Quantity:           MustDecimal(qty),
			UnitOfMeasure:      uom,
			UnitPrice:          MustDecimal(unitPrice),
			TaxPointDate:       issuedDate,
			Description:        ln.Description,
			CreditAmount:       &amt,
			Tax: Tax{
				TaxType:       "IVA",
				TaxCode:       ln.TaxCode,
				TaxPercentage: &taxPct,
			},
		})
		products = append(products, Product{
			ProductType:        "S",
			ProductCode:        pc,
			ProductDescription: ln.Description,
			ProductNumberCode:  pc,
		})
	}
	if len(oms) > 0 {
		return Invoice{}, Customer{}, nil, oms
	}

	net := MustMoney2(formatCents(netCents))
	// Tax/gross left as net with zero tax payable when percentage present but amount not computed —
	// require explicit enrichment for TaxPayable/Gross to avoid inventing tax math.
	// Use NetTotal=GrossTotal=line sum and TaxPayable=0.00 only when TaxPercentage is "0" or "0.00".
	taxPay := MustMoney2("0.00")
	gross := net
	if taxPct := strings.TrimSpace(rec.TaxPercentage); taxPct != "0" && taxPct != "0.00" {
		omit("document_totals", "TaxPayable/GrossTotal não calculados sem regra AO-* de imposto confirmada")
		return Invoice{}, Customer{}, nil, oms
	}

	addr := &AddressStructure{AddressDetail: "Desconhecido", City: "Desconhecido", Country: "AO"}
	cust := Customer{
		CustomerID:           customerID,
		AccountID:            "Desconhecido",
		CustomerTaxID:        strings.TrimSpace(rec.CustomerTaxID),
		CompanyName:          strings.TrimSpace(rec.CustomerName),
		BillingAddress:       addr,
		SelfBillingIndicator: 0,
	}
	if cust.CustomerTaxID == "" {
		cust.CustomerTaxID = "Desconhecido"
	}
	if cust.CompanyName == "" {
		cust.CompanyName = "Desconhecido"
	}

	inv := Invoice{
		InvoiceNo: invoiceNo,
		DocumentStatus: DocumentStatus{
			InvoiceStatus:     InvoiceStatusN,
			InvoiceStatusDate: sealedDT,
			SourceID:          sourceID,
			SourceBilling:     SourceBillingP,
		},
		Hash:            hash,
		HashControl:     hashCtrl,
		InvoiceDate:     issuedDate,
		InvoiceType:     invType,
		SpecialRegimes:  SpecialRegimes{},
		SourceID:        sourceID,
		SystemEntryDate: sealedDT,
		CustomerID:      customerID,
		Line:            lines,
		DocumentTotals: DocumentTotals{
			TaxPayable: taxPay,
			NetTotal:   net,
			GrossTotal: gross,
		},
	}
	return inv, cust, products, nil
}

func ledgerDocTypeToInvoiceType(dt string) (InvoiceType, error) {
	switch strings.TrimSpace(dt) {
	case "invoice":
		return InvoiceTypeFT, nil
	case "credit_note":
		return InvoiceTypeNC, nil
	default:
		return "", fmt.Errorf("document_type %q sem mapping SAF-T InvoiceType", dt)
	}
}

func deriveInvoiceNo(t InvoiceType, series string, seq int64) (string, error) {
	series = strings.TrimSpace(series)
	if series == "" || strings.ContainsAny(series, " /") {
		return "", fmt.Errorf("série inválida para InvoiceNo")
	}
	if seq < 1 {
		return "", fmt.Errorf("fiscal_seq inválido")
	}
	return fmt.Sprintf("%s %s/%d", t, series, seq), nil
}

func dateFromTime(t time.Time) (Date, error) {
	if t.IsZero() {
		return "", fmt.Errorf("instante zero")
	}
	return NewDate(t.UTC().Format("2006-01-02"))
}

func dateTimeFromTime(t time.Time) (DateTime, error) {
	if t.IsZero() {
		return "", fmt.Errorf("instante zero")
	}
	return NewDateTime(t.UTC().Format(time.RFC3339))
}

func formatCents(cents int64) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func formatQuantity(scaled int64) (string, error) {
	if scaled <= 0 {
		return "", fmt.Errorf("quantidade não positiva")
	}
	whole := scaled / QuantityScale
	frac := scaled % QuantityScale
	if frac == 0 {
		return strconv.FormatInt(whole, 10), nil
	}
	s := fmt.Sprintf("%d.%04d", whole, frac)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s, nil
}
