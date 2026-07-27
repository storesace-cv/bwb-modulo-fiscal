package saftao

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// ExportRequest describes a period-scoped structural SAF-T build (≠ AGT acceptance).
type ExportRequest struct {
	Header Header
	// EnabledGroups must be a subset of AllDocumentGroups (DEC-PROD-001).
	// This slice populates SalesInvoices and/or Payments when enabled; other groups stay absent.
	EnabledGroups []DocumentGroup
	// AllowedInvoiceTypes gates which XSD InvoiceType values may appear (enrolment / DEC-PROD-002).
	// Empty ⇒ fail-closed (no invoices accepted).
	AllowedInvoiceTypes []InvoiceType
	// AllowedPaymentTypes gates PaymentType (enrolment). Empty ⇒ fail-closed when Payments present.
	AllowedPaymentTypes []PaymentType
	Customers           []Customer
	Products            []Product
	// TaxTable is optional MasterFiles/TaxTable (RM-SAFT-012). Nil ⇒ omitted; non-nil must validate.
	// Rates/codes are caller-supplied — never invented here (≠ AO-*).
	TaxTable *TaxTable
	Invoices []Invoice
	Payments []Payment
	// IncludeEmptySalesTotals emits SalesInvoices with zero totals when enabled and no invoices.
	IncludeEmptySalesTotals bool
	// IncludeEmptyPaymentsTotals emits Payments with zero totals when enabled and no payments.
	IncludeEmptyPaymentsTotals bool
	// ValidateAgainstXSD runs xmllint when true and available.
	ValidateAgainstXSD bool
}

// ExportResult is the deterministic export artifact (structural only).
type ExportResult struct {
	XML                []byte
	SHA256             string // of XML bytes — artifact integrity; ≠ Invoice.Hash algorithm
	NumberOfInvoices   int
	TotalDebit         DecimalNonNeg // SalesInvoices totals
	TotalCredit        DecimalNonNeg
	NumberOfPayments   int
	PaymentTotalDebit  DecimalNonNeg
	PaymentTotalCredit DecimalNonNeg
	EnabledGroups      []DocumentGroup
	PendingRegulatory  []PendingRegulatory
	XSDChecked         bool
}

// BuildIncrementalExport builds a period-filtered AuditFile, marshals XML, hashes the artifact,
// and validates structurally (and optionally against the embedded XSD).
func BuildIncrementalExport(req ExportRequest) (*ExportResult, error) {
	if err := validateExportHeader(&req.Header); err != nil {
		return nil, err
	}
	start, err := NewDate(req.Header.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w: Header.StartDate", ErrValidation)
	}
	end, err := NewDate(req.Header.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w: Header.EndDate", ErrValidation)
	}
	if string(start) > string(end) {
		return nil, fmt.Errorf("%w: período StartDate>EndDate", ErrValidation)
	}
	groups, err := normalizeEnabledGroups(req.EnabledGroups)
	if err != nil {
		return nil, err
	}
	allowed := map[InvoiceType]struct{}{}
	for _, t := range req.AllowedInvoiceTypes {
		if !ValidInvoiceType(t) {
			return nil, fmt.Errorf("%w: AllowedInvoiceType %q", ErrValidation, t)
		}
		allowed[t] = struct{}{}
	}
	allowedPay := map[PaymentType]struct{}{}
	for _, t := range req.AllowedPaymentTypes {
		if !ValidPaymentType(t) {
			return nil, fmt.Errorf("%w: AllowedPaymentType %q", ErrValidation, t)
		}
		allowedPay[t] = struct{}{}
	}

	salesEnabled := groupEnabled(groups, GroupSalesInvoices)
	paymentsEnabled := groupEnabled(groups, GroupPayments)

	var filtered []Invoice
	var totalDebit, totalCredit DecimalNonNeg
	totalDebit = MustDecimal("0.00")
	totalCredit = MustDecimal("0.00")

	if salesEnabled {
		for i := range req.Invoices {
			inv := req.Invoices[i]
			if err := inv.InvoiceDate.Validate(); err != nil {
				return nil, fmt.Errorf("Invoice[%d]: %w", i, err)
			}
			d := string(inv.InvoiceDate)
			if d < string(start) || d > string(end) {
				continue // incremental period filter
			}
			if _, ok := allowed[inv.InvoiceType]; !ok {
				return nil, fmt.Errorf("%w: InvoiceType %q não permitido pela adesão/config", ErrValidation, inv.InvoiceType)
			}
			if err := inv.ValidateStructural(); err != nil {
				return nil, fmt.Errorf("Invoice[%d]: %w", i, err)
			}
			filtered = append(filtered, inv)
			for _, ln := range inv.Line {
				if ln.DebitAmount != nil {
					sum, err := addMoney2AsDecimal(totalDebit, *ln.DebitAmount)
					if err != nil {
						return nil, err
					}
					totalDebit = sum
				}
				if ln.CreditAmount != nil {
					sum, err := addMoney2AsDecimal(totalCredit, *ln.CreditAmount)
					if err != nil {
						return nil, err
					}
					totalCredit = sum
				}
			}
		}
	} else if len(req.Invoices) > 0 {
		return nil, fmt.Errorf("%w: Invoices sem SalesInvoices enabled", ErrValidation)
	}

	var filteredPay []Payment
	var payDebit, payCredit DecimalNonNeg
	payDebit = MustDecimal("0.00")
	payCredit = MustDecimal("0.00")

	if paymentsEnabled {
		for i := range req.Payments {
			pay := req.Payments[i]
			if err := pay.TransactionDate.Validate(); err != nil {
				return nil, fmt.Errorf("Payment[%d]: %w", i, err)
			}
			d := string(pay.TransactionDate)
			if d < string(start) || d > string(end) {
				continue
			}
			if _, ok := allowedPay[pay.PaymentType]; !ok {
				return nil, fmt.Errorf("%w: PaymentType %q não permitido pela adesão/config", ErrValidation, pay.PaymentType)
			}
			if err := pay.ValidateStructural(); err != nil {
				return nil, fmt.Errorf("Payment[%d]: %w", i, err)
			}
			filteredPay = append(filteredPay, pay)
			for _, ln := range pay.Line {
				if ln.DebitAmount != nil {
					sum, err := addMoney2AsDecimal(payDebit, *ln.DebitAmount)
					if err != nil {
						return nil, err
					}
					payDebit = sum
				}
				if ln.CreditAmount != nil {
					sum, err := addMoney2AsDecimal(payCredit, *ln.CreditAmount)
					if err != nil {
						return nil, err
					}
					payCredit = sum
				}
			}
		}
	} else if len(req.Payments) > 0 {
		return nil, fmt.Errorf("%w: Payments sem Payments enabled", ErrValidation)
	}

	if err := validateMasterRefs(filtered, req.Customers, req.Products); err != nil {
		return nil, err
	}
	if err := validatePaymentCustomerRefs(filteredPay, req.Customers); err != nil {
		return nil, err
	}
	if req.TaxTable != nil {
		if err := req.TaxTable.ValidateStructural(); err != nil {
			return nil, fmt.Errorf("TaxTable: %w", err)
		}
		if err := validateInvoiceTaxAgainstTable(filtered, req.TaxTable); err != nil {
			return nil, err
		}
		if err := validatePaymentTaxAgainstTable(filteredPay, req.TaxTable); err != nil {
			return nil, err
		}
	}

	doc := AuditFile{
		Header:      cloneHeader(&req.Header),
		MasterFiles: &MasterFiles{Customer: req.Customers, Product: req.Products, TaxTable: req.TaxTable},
	}
	needSource := (salesEnabled && (len(filtered) > 0 || req.IncludeEmptySalesTotals)) ||
		(paymentsEnabled && (len(filteredPay) > 0 || req.IncludeEmptyPaymentsTotals))
	if needSource {
		doc.SourceDocuments = &SourceDocuments{}
		if salesEnabled && (len(filtered) > 0 || req.IncludeEmptySalesTotals) {
			doc.SourceDocuments.SalesInvoices = &SalesInvoices{
				NumberOfEntries: strconv.Itoa(len(filtered)),
				TotalDebit:      totalDebit,
				TotalCredit:     totalCredit,
				Invoice:         filtered,
			}
			if err := doc.SourceDocuments.SalesInvoices.ValidateStructural(); err != nil {
				return nil, err
			}
		}
		if paymentsEnabled && (len(filteredPay) > 0 || req.IncludeEmptyPaymentsTotals) {
			doc.SourceDocuments.Payments = &Payments{
				NumberOfEntries: strconv.Itoa(len(filteredPay)),
				TotalDebit:      payDebit,
				TotalCredit:     payCredit,
				Payment:         filteredPay,
			}
			if err := doc.SourceDocuments.Payments.ValidateStructural(); err != nil {
				return nil, err
			}
		}
	}

	raw, err := MarshalAuditFile(doc)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	out := &ExportResult{
		XML:                raw,
		SHA256:             hex.EncodeToString(sum[:]),
		NumberOfInvoices:   len(filtered),
		TotalDebit:         totalDebit,
		TotalCredit:        totalCredit,
		NumberOfPayments:   len(filteredPay),
		PaymentTotalDebit:  payDebit,
		PaymentTotalCredit: payCredit,
		EnabledGroups:      groups,
		PendingRegulatory: []PendingRegulatory{
			PendingHashAlgorithm,
			PendingInvoiceTypeSemantics,
		},
	}
	if paymentsEnabled {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingPaymentTypeSemantics)
	}
	if req.TaxTable != nil {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingTaxTableSemantics)
	}
	if req.ValidateAgainstXSD {
		if !XSDValidatorAvailable() {
			return nil, fmt.Errorf("%w: xmllint indisponível para ValidateAgainstXSD", ErrValidation)
		}
		if err := ValidateXMLAgainstEmbeddedXSD(raw); err != nil {
			return nil, err
		}
		out.XSDChecked = true
	}
	return out, nil
}

func validateExportHeader(h *Header) error {
	if h == nil {
		return fmt.Errorf("%w: Header nil", ErrValidation)
	}
	if strings.TrimSpace(h.AuditFileVersion) == "" {
		h.AuditFileVersion = SchemaVersion()
	}
	if h.AuditFileVersion != SchemaVersion() {
		return fmt.Errorf("%w: AuditFileVersion", ErrValidation)
	}
	for _, pair := range []struct {
		name, val string
	}{
		{"CompanyID", h.CompanyID},
		{"TaxRegistrationNumber", h.TaxRegistrationNumber},
		{"TaxAccountingBasis", h.TaxAccountingBasis},
		{"CompanyName", h.CompanyName},
		{"FiscalYear", h.FiscalYear},
		{"CurrencyCode", h.CurrencyCode},
		{"DateCreated", h.DateCreated},
		{"TaxEntity", h.TaxEntity},
		{"ProductCompanyTaxID", h.ProductCompanyTaxID},
		{"SoftwareValidationNumber", h.SoftwareValidationNumber},
		{"ProductID", h.ProductID},
		{"ProductVersion", h.ProductVersion},
	} {
		if strings.TrimSpace(pair.val) == "" {
			return fmt.Errorf("%w: Header.%s", ErrValidation, pair.name)
		}
	}
	if h.CompanyAddress == nil || strings.TrimSpace(h.CompanyAddress.Country) == "" {
		return fmt.Errorf("%w: Header.CompanyAddress", ErrValidation)
	}
	return nil
}

func normalizeEnabledGroups(in []DocumentGroup) ([]DocumentGroup, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("%w: EnabledGroups vazio", ErrValidation)
	}
	seen := map[DocumentGroup]struct{}{}
	var out []DocumentGroup
	for _, g := range in {
		if !ValidDocumentGroup(g) {
			return nil, fmt.Errorf("%w: grupo %q", ErrValidation, g)
		}
		if _, ok := seen[g]; ok {
			continue
		}
		seen[g] = struct{}{}
		out = append(out, g)
	}
	return out, nil
}

func groupEnabled(groups []DocumentGroup, want DocumentGroup) bool {
	for _, g := range groups {
		if g == want {
			return true
		}
	}
	return false
}

func validateMasterRefs(invoices []Invoice, customers []Customer, products []Product) error {
	cust := map[string]struct{}{}
	for _, c := range customers {
		if strings.TrimSpace(c.CustomerID) == "" {
			return fmt.Errorf("%w: CustomerID vazio", ErrValidation)
		}
		cust[c.CustomerID] = struct{}{}
	}
	prod := map[string]struct{}{}
	for _, p := range products {
		if strings.TrimSpace(p.ProductCode) == "" {
			return fmt.Errorf("%w: ProductCode vazio", ErrValidation)
		}
		prod[p.ProductCode] = struct{}{}
	}
	for i, inv := range invoices {
		if _, ok := cust[inv.CustomerID]; !ok {
			return fmt.Errorf("%w: Invoice[%d] CustomerID sem MasterFiles", ErrValidation, i)
		}
		for j, ln := range inv.Line {
			if _, ok := prod[ln.ProductCode]; !ok {
				return fmt.Errorf("%w: Invoice[%d].Line[%d] ProductCode sem MasterFiles", ErrValidation, i, j)
			}
		}
	}
	return nil
}

// validateInvoiceTaxAgainstTable ensures each invoice line Tax (TaxType+TaxCode) exists in TaxTable.
// Structural integrity only — does not invent rates or claim AO-* tax law.
func validateInvoiceTaxAgainstTable(invoices []Invoice, table *TaxTable) error {
	if table == nil {
		return nil
	}
	keys := taxTableKeys(table)
	for i, inv := range invoices {
		for j, ln := range inv.Line {
			tt := strings.TrimSpace(ln.Tax.TaxType)
			tc := strings.TrimSpace(ln.Tax.TaxCode)
			if tt == "" && tc == "" {
				continue
			}
			k := tt + "|" + tc
			if _, ok := keys[k]; !ok {
				return fmt.Errorf("%w: Invoice[%d].Line[%d] Tax sem TaxTableEntry", ErrValidation, i, j)
			}
		}
	}
	return nil
}

func validatePaymentCustomerRefs(payments []Payment, customers []Customer) error {
	cust := map[string]struct{}{}
	for _, c := range customers {
		if strings.TrimSpace(c.CustomerID) == "" {
			return fmt.Errorf("%w: CustomerID vazio", ErrValidation)
		}
		cust[c.CustomerID] = struct{}{}
	}
	for i, pay := range payments {
		if _, ok := cust[pay.CustomerID]; !ok {
			return fmt.Errorf("%w: Payment[%d] CustomerID sem MasterFiles", ErrValidation, i)
		}
	}
	return nil
}

func validatePaymentTaxAgainstTable(payments []Payment, table *TaxTable) error {
	if table == nil {
		return nil
	}
	keys := taxTableKeys(table)
	for i, pay := range payments {
		for j, ln := range pay.Line {
			if ln.Tax == nil {
				continue
			}
			tt := strings.TrimSpace(ln.Tax.TaxType)
			tc := strings.TrimSpace(ln.Tax.TaxCode)
			if tt == "" && tc == "" {
				continue
			}
			k := tt + "|" + tc
			if _, ok := keys[k]; !ok {
				return fmt.Errorf("%w: Payment[%d].Line[%d] Tax sem TaxTableEntry", ErrValidation, i, j)
			}
		}
	}
	return nil
}

func taxTableKeys(table *TaxTable) map[string]struct{} {
	keys := map[string]struct{}{}
	if table == nil {
		return keys
	}
	for _, e := range table.TaxTableEntry {
		k := string(e.TaxType) + "|" + strings.TrimSpace(e.TaxCode)
		keys[k] = struct{}{}
	}
	return keys
}

func cloneHeader(h *Header) *Header {
	cp := *h
	if h.CompanyAddress != nil {
		addr := *h.CompanyAddress
		cp.CompanyAddress = &addr
	}
	return &cp
}

// addMoney2AsDecimal adds Money2 into a DecimalNonNeg accumulator (exact cents as int64).
func addMoney2AsDecimal(acc DecimalNonNeg, m Money2) (DecimalNonNeg, error) {
	if err := acc.Validate(); err != nil {
		return "", err
	}
	if err := m.Validate(); err != nil {
		return "", err
	}
	a, err := parseMoneyCents(string(acc))
	if err != nil {
		return "", err
	}
	b, err := parseMoneyCents(string(m))
	if err != nil {
		return "", err
	}
	sum := a + b
	if sum < 0 {
		return "", fmt.Errorf("%w: soma negativa", ErrValidation)
	}
	whole := sum / 100
	frac := sum % 100
	return DecimalNonNeg(fmt.Sprintf("%d.%02d", whole, frac)), nil
}

func parseMoneyCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	var whole, frac int64
	var err error
	switch len(parts) {
	case 1:
		whole, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: money", ErrValidation)
		}
	case 2:
		whole, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: money", ErrValidation)
		}
		fracStr := parts[1]
		if len(fracStr) == 1 {
			fracStr += "0"
		}
		if len(fracStr) > 2 {
			// truncate? fail closed — Money2 is exact 2; Decimal may have more
			return 0, fmt.Errorf("%w: money frac", ErrValidation)
		}
		frac, err = strconv.ParseInt(fracStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: money", ErrValidation)
		}
	default:
		return 0, fmt.Errorf("%w: money", ErrValidation)
	}
	return whole*100 + frac, nil
}
