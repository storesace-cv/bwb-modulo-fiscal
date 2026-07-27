package saftao

import (
	"fmt"
	"strings"
)

// WorkingDocuments is XSD SourceDocuments/WorkingDocuments (DEC-PROD-001 grupo 3).
type WorkingDocuments struct {
	NumberOfEntries string         `xml:"NumberOfEntries"`
	TotalDebit      DecimalNonNeg  `xml:"TotalDebit"`
	TotalCredit     DecimalNonNeg  `xml:"TotalCredit"`
	WorkDocument    []WorkDocument `xml:"WorkDocument,omitempty"`
}

// WorkDocument is WorkingDocuments/WorkDocument.
type WorkDocument struct {
	DocumentNumber  string             `xml:"DocumentNumber"`
	DocumentStatus  WorkDocumentStatus `xml:"DocumentStatus"`
	Hash            string             `xml:"Hash"` // PendingHashAlgorithm
	HashControl     string             `xml:"HashControl"`
	Period          string             `xml:"Period,omitempty"`
	WorkDate        Date               `xml:"WorkDate"`
	WorkType        WorkType           `xml:"WorkType"`
	SourceID        string             `xml:"SourceID"`
	EACCode         string             `xml:"EACCode,omitempty"`
	SystemEntryDate DateTime           `xml:"SystemEntryDate"`
	TransactionID   string             `xml:"TransactionID,omitempty"`
	CustomerID      string             `xml:"CustomerID"`
	Line            []WorkDocumentLine `xml:"Line"`
	DocumentTotals  DocumentTotals     `xml:"DocumentTotals"`
}

// WorkDocumentStatus is WorkDocument/DocumentStatus.
type WorkDocumentStatus struct {
	WorkStatus     WorkStatus    `xml:"WorkStatus"`
	WorkStatusDate DateTime      `xml:"WorkStatusDate"`
	Reason         string        `xml:"Reason,omitempty"`
	SourceID       string        `xml:"SourceID"`
	SourceBilling  SourceBilling `xml:"SourceBilling"`
}

// WorkDocumentLine is WorkDocument/Line (Debit XOR Credit; Tax optional).
type WorkDocumentLine struct {
	LineNumber         string        `xml:"LineNumber"`
	ProductCode        string        `xml:"ProductCode"`
	ProductDescription string        `xml:"ProductDescription"`
	Quantity           DecimalNonNeg `xml:"Quantity"`
	UnitOfMeasure      string        `xml:"UnitOfMeasure"`
	UnitPrice          DecimalNonNeg `xml:"UnitPrice"`
	TaxBase            *Money2       `xml:"TaxBase,omitempty"`
	TaxPointDate       Date          `xml:"TaxPointDate"`
	Description        string        `xml:"Description"`
	DebitAmount        *Money2       `xml:"DebitAmount,omitempty"`
	CreditAmount       *Money2       `xml:"CreditAmount,omitempty"`
	Tax                *Tax          `xml:"Tax,omitempty"`
	SettlementAmount   *Money2       `xml:"SettlementAmount,omitempty"`
}

// WorkType is XSD WorkType enumeration (PendingWorkTypeSemantics).
type WorkType string

const (
	WorkTypeCM WorkType = "CM"
	WorkTypeCC WorkType = "CC"
	WorkTypeGR WorkType = "GR"
	WorkTypeNR WorkType = "NR"
	WorkTypeFO WorkType = "FO"
	WorkTypeNE WorkType = "NE"
	WorkTypeOU WorkType = "OU"
	WorkTypeOR WorkType = "OR"
	WorkTypePF WorkType = "PF"
	WorkTypeDC WorkType = "DC"
	WorkTypeRP WorkType = "RP"
	WorkTypeRE WorkType = "RE"
	WorkTypeCS WorkType = "CS"
	WorkTypeLD WorkType = "LD"
	WorkTypeRA WorkType = "RA"
	WorkTypePP WorkType = "PP"
	WorkTypeGC WorkType = "GC"
)

// ValidWorkType reports whether t is in the XSD enumeration.
func ValidWorkType(t WorkType) bool {
	switch t {
	case WorkTypeCM, WorkTypeCC, WorkTypeGR, WorkTypeNR, WorkTypeFO, WorkTypeNE,
		WorkTypeOU, WorkTypeOR, WorkTypePF, WorkTypeDC, WorkTypeRP, WorkTypeRE,
		WorkTypeCS, WorkTypeLD, WorkTypeRA, WorkTypePP, WorkTypeGC:
		return true
	default:
		return false
	}
}

// WorkStatus is XSD WorkStatus enumeration.
type WorkStatus string

const (
	WorkStatusN WorkStatus = "N"
	WorkStatusA WorkStatus = "A"
	WorkStatusF WorkStatus = "F"
)

// ValidWorkStatus reports whether s is in the XSD enumeration.
func ValidWorkStatus(s WorkStatus) bool {
	switch s {
	case WorkStatusN, WorkStatusA, WorkStatusF:
		return true
	default:
		return false
	}
}

// ValidateStructural checks WorkingDocuments (≠ AGT).
func (w *WorkingDocuments) ValidateStructural() error {
	if w == nil {
		return fmt.Errorf("%w: WorkingDocuments nil", ErrValidation)
	}
	if strings.TrimSpace(w.NumberOfEntries) == "" {
		return fmt.Errorf("%w: NumberOfEntries", ErrValidation)
	}
	if err := w.TotalDebit.Validate(); err != nil {
		return err
	}
	if err := w.TotalCredit.Validate(); err != nil {
		return err
	}
	for i := range w.WorkDocument {
		if err := w.WorkDocument[i].ValidateStructural(); err != nil {
			return fmt.Errorf("WorkDocument[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks WorkDocument (≠ AGT).
func (wd *WorkDocument) ValidateStructural() error {
	if wd == nil {
		return fmt.Errorf("%w: WorkDocument nil", ErrValidation)
	}
	if err := ValidateDocumentNumber(wd.DocumentNumber); err != nil {
		return err
	}
	if !ValidWorkStatus(wd.DocumentStatus.WorkStatus) {
		return fmt.Errorf("%w: WorkStatus", ErrValidation)
	}
	if err := wd.DocumentStatus.WorkStatusDate.Validate(); err != nil {
		return err
	}
	if !ValidSourceBilling(wd.DocumentStatus.SourceBilling) {
		return fmt.Errorf("%w: SourceBilling", ErrValidation)
	}
	if strings.TrimSpace(wd.DocumentStatus.SourceID) == "" {
		return fmt.Errorf("%w: DocumentStatus.SourceID", ErrValidation)
	}
	if strings.TrimSpace(wd.Hash) == "" || len(wd.Hash) > 172 {
		return fmt.Errorf("%w: Hash", ErrValidation)
	}
	if strings.TrimSpace(wd.HashControl) == "" || len(wd.HashControl) > 70 {
		return fmt.Errorf("%w: HashControl", ErrValidation)
	}
	if err := wd.WorkDate.Validate(); err != nil {
		return err
	}
	if !ValidWorkType(wd.WorkType) {
		return fmt.Errorf("%w: WorkType", ErrValidation)
	}
	if strings.TrimSpace(wd.SourceID) == "" {
		return fmt.Errorf("%w: SourceID", ErrValidation)
	}
	if err := wd.SystemEntryDate.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(wd.CustomerID) == "" {
		return fmt.Errorf("%w: CustomerID", ErrValidation)
	}
	if len(wd.Line) == 0 {
		return fmt.Errorf("%w: Line obrigatório", ErrValidation)
	}
	for i := range wd.Line {
		if err := wd.Line[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Line[%d]: %w", i, err)
		}
	}
	if err := wd.DocumentTotals.TaxPayable.Validate(); err != nil {
		return err
	}
	if err := wd.DocumentTotals.NetTotal.Validate(); err != nil {
		return err
	}
	if err := wd.DocumentTotals.GrossTotal.Validate(); err != nil {
		return err
	}
	return nil
}

// ValidateStructural checks WorkDocumentLine (≠ AGT).
func (ln *WorkDocumentLine) ValidateStructural() error {
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
	if err := ln.TaxPointDate.Validate(); err != nil {
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
