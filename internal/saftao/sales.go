package saftao

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// Namespace is the XSD targetNamespace (AO_1.01_01).
const Namespace = "urn:OECD:StandardAuditFile-Tax:AO_1.01_01"

// SalesInvoices is XSD SourceDocuments/SalesInvoices (documentos comerciais a clientes).
type SalesInvoices struct {
	NumberOfEntries string        `xml:"NumberOfEntries"`
	TotalDebit      DecimalNonNeg `xml:"TotalDebit"`
	TotalCredit     DecimalNonNeg `xml:"TotalCredit"`
	Invoice         []Invoice     `xml:"Invoice,omitempty"`
}

// Invoice is one SalesInvoices/Invoice (XSD sequence; structural model).
type Invoice struct {
	InvoiceNo       string         `xml:"InvoiceNo"`
	DocumentStatus  DocumentStatus `xml:"DocumentStatus"`
	Hash            string         `xml:"Hash"` // PendingHashAlgorithm
	HashControl     string         `xml:"HashControl"`
	Period          string         `xml:"Period,omitempty"`
	InvoiceDate     Date           `xml:"InvoiceDate"`
	InvoiceType     InvoiceType    `xml:"InvoiceType"`
	SpecialRegimes  SpecialRegimes `xml:"SpecialRegimes"`
	SourceID        string         `xml:"SourceID"`
	EACCode         string         `xml:"EACCode,omitempty"`
	SystemEntryDate DateTime       `xml:"SystemEntryDate"`
	TransactionID   string         `xml:"TransactionID,omitempty"`
	CustomerID      string         `xml:"CustomerID"`
	Line            []InvoiceLine  `xml:"Line"`
	DocumentTotals  DocumentTotals `xml:"DocumentTotals"`
}

// DocumentStatus is Invoice/DocumentStatus.
type DocumentStatus struct {
	InvoiceStatus     InvoiceStatus `xml:"InvoiceStatus"`
	InvoiceStatusDate DateTime      `xml:"InvoiceStatusDate"`
	Reason            string        `xml:"Reason,omitempty"`
	SourceID          string        `xml:"SourceID"`
	SourceBilling     SourceBilling `xml:"SourceBilling"`
}

// SpecialRegimes is XSD SpecialRegimes (0/1 indicators).
type SpecialRegimes struct {
	SelfBillingIndicator         int `xml:"SelfBillingIndicator"`
	CashVATSchemeIndicator       int `xml:"CashVATSchemeIndicator"`
	ThirdPartiesBillingIndicator int `xml:"ThirdPartiesBillingIndicator"`
}

// InvoiceLine is Invoice/Line (Debit XOR Credit per XSD choice).
type InvoiceLine struct {
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
	Tax                Tax           `xml:"Tax"`
	SettlementAmount   *Money2       `xml:"SettlementAmount,omitempty"`
}

// Tax is XSD complexType Tax (TaxPercentage XOR TaxAmount).
type Tax struct {
	TaxType          string  `xml:"TaxType"`
	TaxCountryRegion string  `xml:"TaxCountryRegion,omitempty"`
	TaxCode          string  `xml:"TaxCode"`
	TaxPercentage    *string `xml:"TaxPercentage,omitempty"`
	TaxAmount        *Money2 `xml:"TaxAmount,omitempty"`
}

// DocumentTotals is Invoice/DocumentTotals.
type DocumentTotals struct {
	TaxPayable Money2 `xml:"TaxPayable"`
	NetTotal   Money2 `xml:"NetTotal"`
	GrossTotal Money2 `xml:"GrossTotal"`
}

// InvoiceType is XSD InvoiceType enumeration (values from schema only — PendingInvoiceTypeSemantics).
type InvoiceType string

const (
	InvoiceTypeFT InvoiceType = "FT"
	InvoiceTypeFR InvoiceType = "FR"
	InvoiceTypeGF InvoiceType = "GF"
	InvoiceTypeFG InvoiceType = "FG"
	InvoiceTypeAC InvoiceType = "AC"
	InvoiceTypeAR InvoiceType = "AR"
	InvoiceTypeND InvoiceType = "ND"
	InvoiceTypeNC InvoiceType = "NC"
	InvoiceTypeAF InvoiceType = "AF"
	InvoiceTypeTV InvoiceType = "TV"
	InvoiceTypeRP InvoiceType = "RP"
	InvoiceTypeRE InvoiceType = "RE"
	InvoiceTypeCS InvoiceType = "CS"
	InvoiceTypeLD InvoiceType = "LD"
	InvoiceTypeRA InvoiceType = "RA"
)

// ValidInvoiceType reports whether t is in the XSD enumeration.
func ValidInvoiceType(t InvoiceType) bool {
	switch t {
	case InvoiceTypeFT, InvoiceTypeFR, InvoiceTypeGF, InvoiceTypeFG,
		InvoiceTypeAC, InvoiceTypeAR, InvoiceTypeND, InvoiceTypeNC,
		InvoiceTypeAF, InvoiceTypeTV, InvoiceTypeRP, InvoiceTypeRE,
		InvoiceTypeCS, InvoiceTypeLD, InvoiceTypeRA:
		return true
	default:
		return false
	}
}

// InvoiceStatus is XSD InvoiceStatus enumeration.
type InvoiceStatus string

const (
	InvoiceStatusN InvoiceStatus = "N"
	InvoiceStatusS InvoiceStatus = "S"
	InvoiceStatusA InvoiceStatus = "A"
	InvoiceStatusR InvoiceStatus = "R"
)

// ValidInvoiceStatus reports whether s is in the XSD enumeration.
func ValidInvoiceStatus(s InvoiceStatus) bool {
	switch s {
	case InvoiceStatusN, InvoiceStatusS, InvoiceStatusA, InvoiceStatusR:
		return true
	default:
		return false
	}
}

// SourceBilling is SAFTAOSourceBilling (P|I|M).
type SourceBilling string

const (
	SourceBillingP SourceBilling = "P"
	SourceBillingI SourceBilling = "I"
	SourceBillingM SourceBilling = "M"
)

// ValidSourceBilling reports whether s is in the XSD enumeration.
func ValidSourceBilling(s SourceBilling) bool {
	switch s {
	case SourceBillingP, SourceBillingI, SourceBillingM:
		return true
	default:
		return false
	}
}

var invoiceNoPattern = regexp.MustCompile(`^[^ ]+ [^/^ ]+/[0-9]+$`)

// ValidateInvoiceNo checks XSD InvoiceNo pattern.
func ValidateInvoiceNo(no string) error {
	if len(no) < 1 || len(no) > 60 || !invoiceNoPattern.MatchString(no) {
		return fmt.Errorf("%w: InvoiceNo", ErrValidation)
	}
	return nil
}

// ValidateStructural checks SalesInvoices against XSD-derived structural rules (≠ AGT).
func (si *SalesInvoices) ValidateStructural() error {
	if si == nil {
		return fmt.Errorf("%w: SalesInvoices nil", ErrValidation)
	}
	if strings.TrimSpace(si.NumberOfEntries) == "" {
		return fmt.Errorf("%w: NumberOfEntries", ErrValidation)
	}
	if err := si.TotalDebit.Validate(); err != nil {
		return err
	}
	if err := si.TotalCredit.Validate(); err != nil {
		return err
	}
	for i := range si.Invoice {
		if err := si.Invoice[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Invoice[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks Invoice against XSD-derived structural rules (≠ AGT).
func (inv *Invoice) ValidateStructural() error {
	if inv == nil {
		return fmt.Errorf("%w: Invoice nil", ErrValidation)
	}
	if err := ValidateInvoiceNo(inv.InvoiceNo); err != nil {
		return err
	}
	if !ValidInvoiceStatus(inv.DocumentStatus.InvoiceStatus) {
		return fmt.Errorf("%w: InvoiceStatus", ErrValidation)
	}
	if err := inv.DocumentStatus.InvoiceStatusDate.Validate(); err != nil {
		return err
	}
	if !ValidSourceBilling(inv.DocumentStatus.SourceBilling) {
		return fmt.Errorf("%w: SourceBilling", ErrValidation)
	}
	if strings.TrimSpace(inv.DocumentStatus.SourceID) == "" {
		return fmt.Errorf("%w: DocumentStatus.SourceID", ErrValidation)
	}
	if strings.TrimSpace(inv.Hash) == "" || len(inv.Hash) > 172 {
		return fmt.Errorf("%w: Hash", ErrValidation)
	}
	if strings.TrimSpace(inv.HashControl) == "" || len(inv.HashControl) > 70 {
		return fmt.Errorf("%w: HashControl", ErrValidation)
	}
	if err := inv.InvoiceDate.Validate(); err != nil {
		return err
	}
	if !ValidInvoiceType(inv.InvoiceType) {
		return fmt.Errorf("%w: InvoiceType", ErrValidation)
	}
	for _, ind := range []int{
		inv.SpecialRegimes.SelfBillingIndicator,
		inv.SpecialRegimes.CashVATSchemeIndicator,
		inv.SpecialRegimes.ThirdPartiesBillingIndicator,
	} {
		if ind < 0 || ind > 1 {
			return fmt.Errorf("%w: SpecialRegimes indicator", ErrValidation)
		}
	}
	if strings.TrimSpace(inv.SourceID) == "" {
		return fmt.Errorf("%w: SourceID", ErrValidation)
	}
	if err := inv.SystemEntryDate.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(inv.CustomerID) == "" {
		return fmt.Errorf("%w: CustomerID", ErrValidation)
	}
	if len(inv.Line) == 0 {
		return fmt.Errorf("%w: Line obrigatório", ErrValidation)
	}
	for i := range inv.Line {
		if err := inv.Line[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Line[%d]: %w", i, err)
		}
	}
	if err := inv.DocumentTotals.TaxPayable.Validate(); err != nil {
		return err
	}
	if err := inv.DocumentTotals.NetTotal.Validate(); err != nil {
		return err
	}
	if err := inv.DocumentTotals.GrossTotal.Validate(); err != nil {
		return err
	}
	return nil
}

// ValidateStructural checks InvoiceLine against XSD-derived structural rules (≠ AGT).
func (ln *InvoiceLine) ValidateStructural() error {
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
	return ln.Tax.ValidateStructural()
}

// ValidateStructural checks Tax against XSD choice rules (≠ AGT).
func (t *Tax) ValidateStructural() error {
	if strings.TrimSpace(t.TaxType) == "" || strings.TrimSpace(t.TaxCode) == "" {
		return fmt.Errorf("%w: TaxType/TaxCode", ErrValidation)
	}
	hasPct := t.TaxPercentage != nil && strings.TrimSpace(*t.TaxPercentage) != ""
	hasAmt := t.TaxAmount != nil
	if hasPct == hasAmt {
		return fmt.Errorf("%w: TaxPercentage XOR TaxAmount", ErrValidation)
	}
	if hasAmt {
		return t.TaxAmount.Validate()
	}
	return nil
}

// Ensure encoding/xml is imported for struct tags used by Marshal.
var _ xml.Name
