package saftao

import (
	"fmt"
	"regexp"
	"strings"
)

// MovementOfGoods is XSD SourceDocuments/MovementOfGoods (DEC-PROD-001 grupo 2).
type MovementOfGoods struct {
	NumberOfMovementLines string          `xml:"NumberOfMovementLines"`
	TotalQuantityIssued   DecimalNonNeg   `xml:"TotalQuantityIssued"`
	StockMovement         []StockMovement `xml:"StockMovement,omitempty"`
}

// StockMovement is MovementOfGoods/StockMovement.
type StockMovement struct {
	DocumentNumber    string                 `xml:"DocumentNumber"`
	DocumentStatus    MovementDocumentStatus `xml:"DocumentStatus"`
	Hash              string                 `xml:"Hash"` // PendingHashAlgorithm
	HashControl       string                 `xml:"HashControl"`
	Period            string                 `xml:"Period,omitempty"`
	MovementDate      Date                   `xml:"MovementDate"`
	MovementType      MovementType           `xml:"MovementType"`
	SystemEntryDate   DateTime               `xml:"SystemEntryDate"`
	TransactionID     string                 `xml:"TransactionID,omitempty"`
	CustomerID        string                 `xml:"CustomerID,omitempty"`
	SupplierID        string                 `xml:"SupplierID,omitempty"`
	SourceID          string                 `xml:"SourceID"`
	EACCode           string                 `xml:"EACCode,omitempty"`
	MovementComments  string                 `xml:"MovementComments,omitempty"`
	MovementStartTime DateTime               `xml:"MovementStartTime"`
	Line              []StockMovementLine    `xml:"Line"`
	DocumentTotals    DocumentTotals         `xml:"DocumentTotals"`
}

// MovementDocumentStatus is StockMovement/DocumentStatus.
type MovementDocumentStatus struct {
	MovementStatus     MovementStatus `xml:"MovementStatus"`
	MovementStatusDate DateTime       `xml:"MovementStatusDate"`
	Reason             string         `xml:"Reason,omitempty"`
	SourceID           string         `xml:"SourceID"`
	SourceBilling      SourceBilling  `xml:"SourceBilling"`
}

// StockMovementLine is StockMovement/Line (Debit XOR Credit; Tax optional).
type StockMovementLine struct {
	LineNumber         string        `xml:"LineNumber"`
	ProductCode        string        `xml:"ProductCode"`
	ProductDescription string        `xml:"ProductDescription"`
	Quantity           DecimalNonNeg `xml:"Quantity"`
	UnitOfMeasure      string        `xml:"UnitOfMeasure"`
	UnitPrice          DecimalNonNeg `xml:"UnitPrice"`
	Description        string        `xml:"Description"`
	DebitAmount        *Money2       `xml:"DebitAmount,omitempty"`
	CreditAmount       *Money2       `xml:"CreditAmount,omitempty"`
	Tax                *MovementTax  `xml:"Tax,omitempty"`
	SettlementAmount   *Money2       `xml:"SettlementAmount,omitempty"`
}

// MovementTax is XSD MovementTax (TaxPercentage required; ≠ Invoice Tax choice).
type MovementTax struct {
	TaxType          string `xml:"TaxType"`
	TaxCountryRegion string `xml:"TaxCountryRegion,omitempty"`
	TaxCode          string `xml:"TaxCode"`
	TaxPercentage    string `xml:"TaxPercentage"`
}

// MovementType is XSD MovementType enumeration (PendingMovementTypeSemantics).
type MovementType string

const (
	MovementTypeGR MovementType = "GR"
	MovementTypeGT MovementType = "GT"
	MovementTypeGA MovementType = "GA"
	MovementTypeGD MovementType = "GD"
)

// ValidMovementType reports whether t is in the XSD enumeration.
func ValidMovementType(t MovementType) bool {
	switch t {
	case MovementTypeGR, MovementTypeGT, MovementTypeGA, MovementTypeGD:
		return true
	default:
		return false
	}
}

// MovementStatus is XSD MovementStatus enumeration.
type MovementStatus string

const (
	MovementStatusN MovementStatus = "N"
	MovementStatusT MovementStatus = "T"
	MovementStatusA MovementStatus = "A"
	MovementStatusF MovementStatus = "F"
	MovementStatusR MovementStatus = "R"
)

// ValidMovementStatus reports whether s is in the XSD enumeration.
func ValidMovementStatus(s MovementStatus) bool {
	switch s {
	case MovementStatusN, MovementStatusT, MovementStatusA, MovementStatusF, MovementStatusR:
		return true
	default:
		return false
	}
}

var documentNumberPattern = regexp.MustCompile(`^[^ ]+ [^/^ ]+/[0-9]+$`)

// ValidateDocumentNumber checks XSD DocumentNumber pattern (same shape as InvoiceNo).
func ValidateDocumentNumber(no string) error {
	if len(no) < 1 || len(no) > 60 || !documentNumberPattern.MatchString(no) {
		return fmt.Errorf("%w: DocumentNumber", ErrValidation)
	}
	return nil
}

// ValidateStructural checks MovementOfGoods against XSD-derived rules (≠ AGT).
func (m *MovementOfGoods) ValidateStructural() error {
	if m == nil {
		return fmt.Errorf("%w: MovementOfGoods nil", ErrValidation)
	}
	if strings.TrimSpace(m.NumberOfMovementLines) == "" {
		return fmt.Errorf("%w: NumberOfMovementLines", ErrValidation)
	}
	if err := m.TotalQuantityIssued.Validate(); err != nil {
		return err
	}
	for i := range m.StockMovement {
		if err := m.StockMovement[i].ValidateStructural(); err != nil {
			return fmt.Errorf("StockMovement[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks StockMovement against XSD-derived rules (≠ AGT).
func (sm *StockMovement) ValidateStructural() error {
	if sm == nil {
		return fmt.Errorf("%w: StockMovement nil", ErrValidation)
	}
	if err := ValidateDocumentNumber(sm.DocumentNumber); err != nil {
		return err
	}
	if !ValidMovementStatus(sm.DocumentStatus.MovementStatus) {
		return fmt.Errorf("%w: MovementStatus", ErrValidation)
	}
	if err := sm.DocumentStatus.MovementStatusDate.Validate(); err != nil {
		return err
	}
	if !ValidSourceBilling(sm.DocumentStatus.SourceBilling) {
		return fmt.Errorf("%w: SourceBilling", ErrValidation)
	}
	if strings.TrimSpace(sm.DocumentStatus.SourceID) == "" {
		return fmt.Errorf("%w: DocumentStatus.SourceID", ErrValidation)
	}
	if strings.TrimSpace(sm.Hash) == "" || len(sm.Hash) > 172 {
		return fmt.Errorf("%w: Hash", ErrValidation)
	}
	if strings.TrimSpace(sm.HashControl) == "" || len(sm.HashControl) > 70 {
		return fmt.Errorf("%w: HashControl", ErrValidation)
	}
	if err := sm.MovementDate.Validate(); err != nil {
		return err
	}
	if !ValidMovementType(sm.MovementType) {
		return fmt.Errorf("%w: MovementType", ErrValidation)
	}
	if err := sm.SystemEntryDate.Validate(); err != nil {
		return err
	}
	hasCust := strings.TrimSpace(sm.CustomerID) != ""
	hasSupp := strings.TrimSpace(sm.SupplierID) != ""
	if hasCust == hasSupp {
		return fmt.Errorf("%w: CustomerID XOR SupplierID", ErrValidation)
	}
	if strings.TrimSpace(sm.SourceID) == "" {
		return fmt.Errorf("%w: SourceID", ErrValidation)
	}
	if err := sm.MovementStartTime.Validate(); err != nil {
		return err
	}
	if len(sm.Line) == 0 {
		return fmt.Errorf("%w: Line obrigatório", ErrValidation)
	}
	for i := range sm.Line {
		if err := sm.Line[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Line[%d]: %w", i, err)
		}
	}
	if err := sm.DocumentTotals.TaxPayable.Validate(); err != nil {
		return err
	}
	if err := sm.DocumentTotals.NetTotal.Validate(); err != nil {
		return err
	}
	if err := sm.DocumentTotals.GrossTotal.Validate(); err != nil {
		return err
	}
	return nil
}

// ValidateStructural checks StockMovementLine (≠ AGT).
func (ln *StockMovementLine) ValidateStructural() error {
	if strings.TrimSpace(ln.LineNumber) == "" {
		return fmt.Errorf("%w: LineNumber", ErrValidation)
	}
	if strings.TrimSpace(ln.ProductCode) == "" || strings.TrimSpace(ln.ProductDescription) == "" {
		return fmt.Errorf("%w: Product*", ErrValidation)
	}
	if err := ln.Quantity.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(ln.UnitOfMeasure) == "" {
		return fmt.Errorf("%w: UnitOfMeasure", ErrValidation)
	}
	if err := ln.UnitPrice.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(ln.Description) == "" {
		return fmt.Errorf("%w: Description", ErrValidation)
	}
	hasDebit := ln.DebitAmount != nil
	hasCredit := ln.CreditAmount != nil
	if hasDebit == hasCredit {
		return fmt.Errorf("%w: DebitAmount XOR CreditAmount", ErrValidation)
	}
	if hasDebit {
		if err := ln.DebitAmount.Validate(); err != nil {
			return err
		}
	}
	if hasCredit {
		if err := ln.CreditAmount.Validate(); err != nil {
			return err
		}
	}
	if ln.Tax != nil {
		return ln.Tax.ValidateStructural()
	}
	return nil
}

// ValidateStructural checks MovementTax (≠ AGT).
func (t *MovementTax) ValidateStructural() error {
	if t.TaxType != "IVA" && t.TaxType != "NS" {
		return fmt.Errorf("%w: MovementTax.TaxType", ErrValidation)
	}
	switch t.TaxCode {
	case "RED", "INT", "NOR", "ISE", "OUT", "NS", "NA":
	default:
		return fmt.Errorf("%w: MovementTax.TaxCode", ErrValidation)
	}
	if strings.TrimSpace(t.TaxPercentage) == "" {
		return fmt.Errorf("%w: MovementTax.TaxPercentage", ErrValidation)
	}
	return nil
}
