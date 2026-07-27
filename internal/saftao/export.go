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
	// This slice populates the five L3 SourceDocuments groups when enabled.
	EnabledGroups []DocumentGroup
	// AllowedInvoiceTypes gates which XSD InvoiceType values may appear (enrolment / DEC-PROD-002).
	// Empty ⇒ fail-closed (no invoices accepted).
	AllowedInvoiceTypes []InvoiceType
	// AllowedPaymentTypes gates PaymentType (enrolment). Empty ⇒ fail-closed when Payments present.
	AllowedPaymentTypes []PaymentType
	// AllowedPurchaseTypes gates PurchaseType. Empty ⇒ fail-closed when PurchaseInvoices present.
	AllowedPurchaseTypes []PurchaseType
	// AllowedMovementTypes gates MovementType. Empty ⇒ fail-closed when StockMovements present.
	AllowedMovementTypes []MovementType
	// AllowedWorkTypes gates WorkType. Empty ⇒ fail-closed when WorkDocuments present.
	AllowedWorkTypes []WorkType
	Customers        []Customer
	Suppliers        []Supplier
	Products         []Product
	// TaxTable is optional MasterFiles/TaxTable (RM-SAFT-012). Nil ⇒ omitted; non-nil must validate.
	// Rates/codes are caller-supplied — never invented here (≠ AO-*).
	TaxTable *TaxTable
	// GeneralLedgerAccounts is optional MasterFiles/GeneralLedgerAccounts (RM-SAFT-017).
	// Empty ⇒ omitted; non-empty must validate (≥1 Account each). Caller-supplied — never invented.
	GeneralLedgerAccounts []GeneralLedgerAccounts
	Invoices              []Invoice
	Payments              []Payment
	PurchaseInvoices      []PurchaseInvoice
	StockMovements        []StockMovement
	WorkDocuments         []WorkDocument
	// IncludeEmptySalesTotals emits SalesInvoices with zero totals when enabled and no invoices.
	IncludeEmptySalesTotals bool
	// IncludeEmptyPaymentsTotals emits Payments with zero totals when enabled and no payments.
	IncludeEmptyPaymentsTotals bool
	// IncludeEmptyPurchaseEntries emits PurchaseInvoices with NumberOfEntries=0 when enabled and empty.
	IncludeEmptyPurchaseEntries bool
	// IncludeEmptyMovementTotals emits MovementOfGoods with zero lines/qty when enabled and empty.
	IncludeEmptyMovementTotals bool
	// IncludeEmptyWorkingTotals emits WorkingDocuments with zero totals when enabled and empty.
	IncludeEmptyWorkingTotals bool
	// ValidateAgainstXSD runs xmllint when true and available.
	ValidateAgainstXSD bool
}

// ExportResult is the deterministic export artifact (structural only).
type ExportResult struct {
	XML                      []byte
	SHA256                   string // of XML bytes — artifact integrity; ≠ Invoice.Hash algorithm
	NumberOfInvoices         int
	TotalDebit               DecimalNonNeg // SalesInvoices totals
	TotalCredit              DecimalNonNeg
	NumberOfPayments         int
	PaymentTotalDebit        DecimalNonNeg
	PaymentTotalCredit       DecimalNonNeg
	NumberOfPurchaseInvoices int
	NumberOfStockMovements   int
	NumberOfMovementLines    int
	TotalQuantityIssued      DecimalNonNeg
	NumberOfWorkDocuments    int
	WorkTotalDebit           DecimalNonNeg
	WorkTotalCredit          DecimalNonNeg
	EnabledGroups            []DocumentGroup
	PendingRegulatory        []PendingRegulatory
	XSDChecked               bool
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
	allowedPurchase := map[PurchaseType]struct{}{}
	for _, t := range req.AllowedPurchaseTypes {
		if !ValidPurchaseType(t) {
			return nil, fmt.Errorf("%w: AllowedPurchaseType %q", ErrValidation, t)
		}
		allowedPurchase[t] = struct{}{}
	}
	allowedMovement := map[MovementType]struct{}{}
	for _, t := range req.AllowedMovementTypes {
		if !ValidMovementType(t) {
			return nil, fmt.Errorf("%w: AllowedMovementType %q", ErrValidation, t)
		}
		allowedMovement[t] = struct{}{}
	}
	allowedWork := map[WorkType]struct{}{}
	for _, t := range req.AllowedWorkTypes {
		if !ValidWorkType(t) {
			return nil, fmt.Errorf("%w: AllowedWorkType %q", ErrValidation, t)
		}
		allowedWork[t] = struct{}{}
	}

	salesEnabled := groupEnabled(groups, GroupSalesInvoices)
	paymentsEnabled := groupEnabled(groups, GroupPayments)
	purchaseEnabled := groupEnabled(groups, GroupPurchaseInvoices)
	movementEnabled := groupEnabled(groups, GroupMovementOfGoods)
	workingEnabled := groupEnabled(groups, GroupWorkingDocuments)

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

	var filteredPurchase []PurchaseInvoice
	if purchaseEnabled {
		for i := range req.PurchaseInvoices {
			inv := req.PurchaseInvoices[i]
			if err := inv.InvoiceDate.Validate(); err != nil {
				return nil, fmt.Errorf("PurchaseInvoice[%d]: %w", i, err)
			}
			d := string(inv.InvoiceDate)
			if d < string(start) || d > string(end) {
				continue
			}
			if _, ok := allowedPurchase[inv.PurchaseType]; !ok {
				return nil, fmt.Errorf("%w: PurchaseType %q não permitido pela adesão/config", ErrValidation, inv.PurchaseType)
			}
			if err := inv.ValidateStructural(); err != nil {
				return nil, fmt.Errorf("PurchaseInvoice[%d]: %w", i, err)
			}
			filteredPurchase = append(filteredPurchase, inv)
		}
	} else if len(req.PurchaseInvoices) > 0 {
		return nil, fmt.Errorf("%w: PurchaseInvoices sem PurchaseInvoices enabled", ErrValidation)
	}

	var filteredMov []StockMovement
	var movementLineCount int
	totalQty := MustDecimal("0")
	if movementEnabled {
		for i := range req.StockMovements {
			sm := req.StockMovements[i]
			if err := sm.MovementDate.Validate(); err != nil {
				return nil, fmt.Errorf("StockMovement[%d]: %w", i, err)
			}
			d := string(sm.MovementDate)
			if d < string(start) || d > string(end) {
				continue
			}
			if _, ok := allowedMovement[sm.MovementType]; !ok {
				return nil, fmt.Errorf("%w: MovementType %q não permitido pela adesão/config", ErrValidation, sm.MovementType)
			}
			if err := sm.ValidateStructural(); err != nil {
				return nil, fmt.Errorf("StockMovement[%d]: %w", i, err)
			}
			filteredMov = append(filteredMov, sm)
			for _, ln := range sm.Line {
				movementLineCount++
				sum, err := addDecimalNonNeg(totalQty, ln.Quantity)
				if err != nil {
					return nil, fmt.Errorf("StockMovement[%d]: %w", i, err)
				}
				totalQty = sum
			}
		}
	} else if len(req.StockMovements) > 0 {
		return nil, fmt.Errorf("%w: StockMovements sem MovementOfGoods enabled", ErrValidation)
	}

	var filteredWork []WorkDocument
	workDebit := MustDecimal("0.00")
	workCredit := MustDecimal("0.00")
	if workingEnabled {
		for i := range req.WorkDocuments {
			wd := req.WorkDocuments[i]
			if err := wd.WorkDate.Validate(); err != nil {
				return nil, fmt.Errorf("WorkDocument[%d]: %w", i, err)
			}
			d := string(wd.WorkDate)
			if d < string(start) || d > string(end) {
				continue
			}
			if _, ok := allowedWork[wd.WorkType]; !ok {
				return nil, fmt.Errorf("%w: WorkType %q não permitido pela adesão/config", ErrValidation, wd.WorkType)
			}
			if err := wd.ValidateStructural(); err != nil {
				return nil, fmt.Errorf("WorkDocument[%d]: %w", i, err)
			}
			filteredWork = append(filteredWork, wd)
			for _, ln := range wd.Line {
				if ln.DebitAmount != nil {
					sum, err := addMoney2AsDecimal(workDebit, *ln.DebitAmount)
					if err != nil {
						return nil, err
					}
					workDebit = sum
				}
				if ln.CreditAmount != nil {
					sum, err := addMoney2AsDecimal(workCredit, *ln.CreditAmount)
					if err != nil {
						return nil, err
					}
					workCredit = sum
				}
			}
		}
	} else if len(req.WorkDocuments) > 0 {
		return nil, fmt.Errorf("%w: WorkDocuments sem WorkingDocuments enabled", ErrValidation)
	}

	if err := validateMasterRefs(filtered, req.Customers, req.Products); err != nil {
		return nil, err
	}
	if err := validatePaymentCustomerRefs(filteredPay, req.Customers); err != nil {
		return nil, err
	}
	if err := validatePurchaseSupplierRefs(filteredPurchase, req.Suppliers); err != nil {
		return nil, err
	}
	if err := validateStockMovementRefs(filteredMov, req.Customers, req.Suppliers, req.Products); err != nil {
		return nil, err
	}
	if err := validateWorkDocumentRefs(filteredWork, req.Customers, req.Products); err != nil {
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
		if err := validateMovementTaxAgainstTable(filteredMov, req.TaxTable); err != nil {
			return nil, err
		}
		if err := validateWorkTaxAgainstTable(filteredWork, req.TaxTable); err != nil {
			return nil, err
		}
	}
	if len(req.GeneralLedgerAccounts) > MaxTableEntries {
		return nil, fmt.Errorf("%w: GeneralLedgerAccounts excedeu MaxTableEntries", ErrValidation)
	}
	for i := range req.GeneralLedgerAccounts {
		if err := req.GeneralLedgerAccounts[i].ValidateStructural(); err != nil {
			return nil, fmt.Errorf("GeneralLedgerAccounts[%d]: %w", i, err)
		}
	}

	doc := AuditFile{
		Header: cloneHeader(&req.Header),
		MasterFiles: &MasterFiles{
			GeneralLedgerAccounts: req.GeneralLedgerAccounts,
			Customer:              req.Customers,
			Supplier:              req.Suppliers,
			Product:               req.Products,
			TaxTable:              req.TaxTable,
		},
	}
	needSource := (salesEnabled && (len(filtered) > 0 || req.IncludeEmptySalesTotals)) ||
		(paymentsEnabled && (len(filteredPay) > 0 || req.IncludeEmptyPaymentsTotals)) ||
		(purchaseEnabled && (len(filteredPurchase) > 0 || req.IncludeEmptyPurchaseEntries)) ||
		(movementEnabled && (len(filteredMov) > 0 || req.IncludeEmptyMovementTotals)) ||
		(workingEnabled && (len(filteredWork) > 0 || req.IncludeEmptyWorkingTotals))
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
		if purchaseEnabled && (len(filteredPurchase) > 0 || req.IncludeEmptyPurchaseEntries) {
			doc.SourceDocuments.PurchaseInvoices = &PurchaseInvoices{
				NumberOfEntries: strconv.Itoa(len(filteredPurchase)),
				Invoice:         filteredPurchase,
			}
			if err := doc.SourceDocuments.PurchaseInvoices.ValidateStructural(); err != nil {
				return nil, err
			}
		}
		if movementEnabled && (len(filteredMov) > 0 || req.IncludeEmptyMovementTotals) {
			doc.SourceDocuments.MovementOfGoods = &MovementOfGoods{
				NumberOfMovementLines: strconv.Itoa(movementLineCount),
				TotalQuantityIssued:   totalQty,
				StockMovement:         filteredMov,
			}
			if err := doc.SourceDocuments.MovementOfGoods.ValidateStructural(); err != nil {
				return nil, err
			}
		}
		if workingEnabled && (len(filteredWork) > 0 || req.IncludeEmptyWorkingTotals) {
			doc.SourceDocuments.WorkingDocuments = &WorkingDocuments{
				NumberOfEntries: strconv.Itoa(len(filteredWork)),
				TotalDebit:      workDebit,
				TotalCredit:     workCredit,
				WorkDocument:    filteredWork,
			}
			if err := doc.SourceDocuments.WorkingDocuments.ValidateStructural(); err != nil {
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
		XML:                      raw,
		SHA256:                   hex.EncodeToString(sum[:]),
		NumberOfInvoices:         len(filtered),
		TotalDebit:               totalDebit,
		TotalCredit:              totalCredit,
		NumberOfPayments:         len(filteredPay),
		PaymentTotalDebit:        payDebit,
		PaymentTotalCredit:       payCredit,
		NumberOfPurchaseInvoices: len(filteredPurchase),
		NumberOfStockMovements:   len(filteredMov),
		NumberOfMovementLines:    movementLineCount,
		TotalQuantityIssued:      totalQty,
		NumberOfWorkDocuments:    len(filteredWork),
		WorkTotalDebit:           workDebit,
		WorkTotalCredit:          workCredit,
		EnabledGroups:            groups,
		PendingRegulatory: []PendingRegulatory{
			PendingHashAlgorithm,
			PendingInvoiceTypeSemantics,
		},
	}
	if paymentsEnabled {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingPaymentTypeSemantics)
	}
	if purchaseEnabled {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingPurchaseTypeSemantics)
	}
	if movementEnabled {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingMovementTypeSemantics)
	}
	if workingEnabled {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingWorkTypeSemantics)
	}
	if req.TaxTable != nil {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingTaxTableSemantics)
	}
	if len(req.GeneralLedgerAccounts) > 0 {
		out.PendingRegulatory = append(out.PendingRegulatory, PendingGLAccountSemantics)
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

func validatePurchaseSupplierRefs(invoices []PurchaseInvoice, suppliers []Supplier) error {
	sup := map[string]struct{}{}
	for _, s := range suppliers {
		if strings.TrimSpace(s.SupplierID) == "" {
			return fmt.Errorf("%w: SupplierID vazio", ErrValidation)
		}
		sup[s.SupplierID] = struct{}{}
	}
	for i, inv := range invoices {
		if _, ok := sup[inv.SupplierID]; !ok {
			return fmt.Errorf("%w: PurchaseInvoice[%d] SupplierID sem MasterFiles", ErrValidation, i)
		}
	}
	return nil
}

func validateWorkDocumentRefs(docs []WorkDocument, customers []Customer, products []Product) error {
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
	for i, wd := range docs {
		if _, ok := cust[wd.CustomerID]; !ok {
			return fmt.Errorf("%w: WorkDocument[%d] CustomerID sem MasterFiles", ErrValidation, i)
		}
		for j, ln := range wd.Line {
			if _, ok := prod[ln.ProductCode]; !ok {
				return fmt.Errorf("%w: WorkDocument[%d].Line[%d] ProductCode sem MasterFiles", ErrValidation, i, j)
			}
		}
	}
	return nil
}

func validateWorkTaxAgainstTable(docs []WorkDocument, table *TaxTable) error {
	if table == nil {
		return nil
	}
	keys := taxTableKeys(table)
	for i, wd := range docs {
		for j, ln := range wd.Line {
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
				return fmt.Errorf("%w: WorkDocument[%d].Line[%d] Tax sem TaxTableEntry", ErrValidation, i, j)
			}
		}
	}
	return nil
}

func validateStockMovementRefs(movements []StockMovement, customers []Customer, suppliers []Supplier, products []Product) error {
	cust := map[string]struct{}{}
	for _, c := range customers {
		if strings.TrimSpace(c.CustomerID) == "" {
			return fmt.Errorf("%w: CustomerID vazio", ErrValidation)
		}
		cust[c.CustomerID] = struct{}{}
	}
	sup := map[string]struct{}{}
	for _, s := range suppliers {
		if strings.TrimSpace(s.SupplierID) == "" {
			return fmt.Errorf("%w: SupplierID vazio", ErrValidation)
		}
		sup[s.SupplierID] = struct{}{}
	}
	prod := map[string]struct{}{}
	for _, p := range products {
		if strings.TrimSpace(p.ProductCode) == "" {
			return fmt.Errorf("%w: ProductCode vazio", ErrValidation)
		}
		prod[p.ProductCode] = struct{}{}
	}
	for i, sm := range movements {
		if cid := strings.TrimSpace(sm.CustomerID); cid != "" {
			if _, ok := cust[cid]; !ok {
				return fmt.Errorf("%w: StockMovement[%d] CustomerID sem MasterFiles", ErrValidation, i)
			}
		}
		if sid := strings.TrimSpace(sm.SupplierID); sid != "" {
			if _, ok := sup[sid]; !ok {
				return fmt.Errorf("%w: StockMovement[%d] SupplierID sem MasterFiles", ErrValidation, i)
			}
		}
		for j, ln := range sm.Line {
			if _, ok := prod[ln.ProductCode]; !ok {
				return fmt.Errorf("%w: StockMovement[%d].Line[%d] ProductCode sem MasterFiles", ErrValidation, i, j)
			}
		}
	}
	return nil
}

func validateMovementTaxAgainstTable(movements []StockMovement, table *TaxTable) error {
	if table == nil {
		return nil
	}
	keys := taxTableKeys(table)
	for i, sm := range movements {
		for j, ln := range sm.Line {
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
				return fmt.Errorf("%w: StockMovement[%d].Line[%d] Tax sem TaxTableEntry", ErrValidation, i, j)
			}
		}
	}
	return nil
}

// addDecimalNonNeg adds two SAFdecimalType non-negative values using milli-units (3 frac digits max fail-closed).
func addDecimalNonNeg(a, b DecimalNonNeg) (DecimalNonNeg, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	if err := b.Validate(); err != nil {
		return "", err
	}
	ai, err := parseDecimalMilli(string(a))
	if err != nil {
		return "", err
	}
	bi, err := parseDecimalMilli(string(b))
	if err != nil {
		return "", err
	}
	sum := ai + bi
	if sum < 0 {
		return "", fmt.Errorf("%w: soma negativa", ErrValidation)
	}
	whole := sum / 1000
	frac := sum % 1000
	if frac == 0 {
		return DecimalNonNeg(fmt.Sprintf("%d", whole)), nil
	}
	s := fmt.Sprintf("%d.%03d", whole, frac)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return DecimalNonNeg(s), nil
}

func parseDecimalMilli(s string) (int64, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	var whole, frac int64
	var err error
	switch len(parts) {
	case 1:
		whole, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: decimal", ErrValidation)
		}
	case 2:
		whole, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: decimal", ErrValidation)
		}
		fracStr := parts[1]
		if len(fracStr) > 3 {
			return 0, fmt.Errorf("%w: decimal frac", ErrValidation)
		}
		for len(fracStr) < 3 {
			fracStr += "0"
		}
		frac, err = strconv.ParseInt(fracStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: decimal", ErrValidation)
		}
	default:
		return 0, fmt.Errorf("%w: decimal", ErrValidation)
	}
	return whole*1000 + frac, nil
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
