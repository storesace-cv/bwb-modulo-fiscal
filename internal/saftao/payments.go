package saftao

import (
	"fmt"
	"strings"
)

// Hard structural caps (fail-closed). Not AGT limits — protect memory/export size.
const (
	MaxTableEntries      = 100_000
	MaxLinesPerDocument  = 10_000
	MaxPaymentMethods    = 100
	MaxSourceDocumentIDs = 100
)

// Payments is XSD SourceDocuments/Payments (recibos / DEC-PROD-001 grupo 4).
type Payments struct {
	NumberOfEntries string        `xml:"NumberOfEntries"`
	TotalDebit      DecimalNonNeg `xml:"TotalDebit"`
	TotalCredit     DecimalNonNeg `xml:"TotalCredit"`
	Payment         []Payment     `xml:"Payment,omitempty"`
}

// Payment is Payments/Payment.
type Payment struct {
	PaymentRefNo    string             `xml:"PaymentRefNo"`
	Period          string             `xml:"Period,omitempty"`
	TransactionID   string             `xml:"TransactionID,omitempty"`
	TransactionDate Date               `xml:"TransactionDate"`
	PaymentType     PaymentType        `xml:"PaymentType"`
	Description     string             `xml:"Description,omitempty"`
	SystemID        string             `xml:"SystemID,omitempty"`
	DocumentStatus  PaymentDocStatus   `xml:"DocumentStatus"`
	PaymentMethod   []SAFPaymentMethod `xml:"PaymentMethod,omitempty"`
	SourceID        string             `xml:"SourceID"`
	SystemEntryDate DateTime           `xml:"SystemEntryDate"`
	CustomerID      string             `xml:"CustomerID"`
	Line            []PaymentLine      `xml:"Line"`
	DocumentTotals  PaymentDocTotals   `xml:"DocumentTotals"`
}

// PaymentDocStatus is Payment/DocumentStatus (SourcePayment, not SourceBilling).
type PaymentDocStatus struct {
	PaymentStatus     PaymentStatus `xml:"PaymentStatus"`
	PaymentStatusDate DateTime      `xml:"PaymentStatusDate"`
	Reason            string        `xml:"Reason,omitempty"`
	SourceID          string        `xml:"SourceID"`
	SourcePayment     SourcePayment `xml:"SourcePayment"`
}

// SAFPaymentMethod is XSD PaymentMethod (meio de pagamento).
type SAFPaymentMethod struct {
	PaymentMechanism string        `xml:"PaymentMechanism,omitempty"`
	PaymentAmount    DecimalNonNeg `xml:"PaymentAmount"`
	PaymentDate      Date          `xml:"PaymentDate"`
}

// PaymentLine is Payment/Line.
type PaymentLine struct {
	LineNumber       string             `xml:"LineNumber"`
	SourceDocumentID []SourceDocumentID `xml:"SourceDocumentID"`
	SettlementAmount *Money2            `xml:"SettlementAmount,omitempty"`
	DebitAmount      *Money2            `xml:"DebitAmount,omitempty"`
	CreditAmount     *Money2            `xml:"CreditAmount,omitempty"`
	Tax              *PaymentTax        `xml:"Tax,omitempty"`
}

// SourceDocumentID references the originating sales document.
type SourceDocumentID struct {
	OriginatingON string `xml:"OriginatingON"`
	InvoiceDate   Date   `xml:"InvoiceDate"`
	Description   string `xml:"Description,omitempty"`
}

// PaymentTax is XSD PaymentTax (TaxPercentage XOR TaxAmount).
type PaymentTax struct {
	TaxType          string  `xml:"TaxType"`
	TaxCountryRegion string  `xml:"TaxCountryRegion,omitempty"`
	TaxCode          string  `xml:"TaxCode"`
	TaxPercentage    *string `xml:"TaxPercentage,omitempty"`
	TaxAmount        *Money2 `xml:"TaxAmount,omitempty"`
}

// PaymentDocTotals is Payment/DocumentTotals (optional Settlement sum of line discounts).
type PaymentDocTotals struct {
	TaxPayable Money2             `xml:"TaxPayable"`
	NetTotal   Money2             `xml:"NetTotal"`
	GrossTotal Money2             `xml:"GrossTotal"`
	Settlement *PaymentSettlement `xml:"Settlement,omitempty"`
}

// PaymentSettlement is DocumentTotals/Settlement for receipts.
type PaymentSettlement struct {
	SettlementAmount Money2 `xml:"SettlementAmount"`
}

// PaymentType is SAFTAOPaymentType (RC|RG|AR).
type PaymentType string

const (
	PaymentTypeRC PaymentType = "RC"
	PaymentTypeRG PaymentType = "RG"
	PaymentTypeAR PaymentType = "AR"
)

// ValidPaymentType reports whether t is in the XSD enumeration.
func ValidPaymentType(t PaymentType) bool {
	switch t {
	case PaymentTypeRC, PaymentTypeRG, PaymentTypeAR:
		return true
	default:
		return false
	}
}

// PaymentStatus is XSD PaymentStatus (N|A).
type PaymentStatus string

const (
	PaymentStatusN PaymentStatus = "N"
	PaymentStatusA PaymentStatus = "A"
)

// ValidPaymentStatus reports whether s is in the XSD enumeration.
func ValidPaymentStatus(s PaymentStatus) bool {
	switch s {
	case PaymentStatusN, PaymentStatusA:
		return true
	default:
		return false
	}
}

// SourcePayment is SAFTAOSourcePayment (P|I|M).
type SourcePayment string

const (
	SourcePaymentP SourcePayment = "P"
	SourcePaymentI SourcePayment = "I"
	SourcePaymentM SourcePayment = "M"
)

// ValidSourcePayment reports whether s is in the XSD enumeration.
func ValidSourcePayment(s SourcePayment) bool {
	switch s {
	case SourcePaymentP, SourcePaymentI, SourcePaymentM:
		return true
	default:
		return false
	}
}

// ValidatePaymentRefNo checks XSD PaymentRefNo pattern (same shape as InvoiceNo).
func ValidatePaymentRefNo(no string) error {
	return ValidateDocumentNumber(no)
}

// ValidateStructural checks Payments (≠ AGT).
func (p *Payments) ValidateStructural() error {
	if p == nil {
		return fmt.Errorf("%w: Payments nil", ErrValidation)
	}
	if strings.TrimSpace(p.NumberOfEntries) == "" {
		return fmt.Errorf("%w: NumberOfEntries", ErrValidation)
	}
	if err := p.TotalDebit.Validate(); err != nil {
		return err
	}
	if err := p.TotalCredit.Validate(); err != nil {
		return err
	}
	if len(p.Payment) > MaxTableEntries {
		return fmt.Errorf("%w: Payments excedeu MaxTableEntries", ErrValidation)
	}
	for i := range p.Payment {
		if err := p.Payment[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Payment[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks Payment (≠ AGT).
func (pay *Payment) ValidateStructural() error {
	if pay == nil {
		return fmt.Errorf("%w: Payment nil", ErrValidation)
	}
	if err := ValidatePaymentRefNo(pay.PaymentRefNo); err != nil {
		return err
	}
	if err := pay.TransactionDate.Validate(); err != nil {
		return err
	}
	if !ValidPaymentType(pay.PaymentType) {
		return fmt.Errorf("%w: PaymentType", ErrValidation)
	}
	if !ValidPaymentStatus(pay.DocumentStatus.PaymentStatus) {
		return fmt.Errorf("%w: PaymentStatus", ErrValidation)
	}
	if err := pay.DocumentStatus.PaymentStatusDate.Validate(); err != nil {
		return err
	}
	if !ValidSourcePayment(pay.DocumentStatus.SourcePayment) {
		return fmt.Errorf("%w: SourcePayment", ErrValidation)
	}
	if strings.TrimSpace(pay.DocumentStatus.SourceID) == "" {
		return fmt.Errorf("%w: DocumentStatus.SourceID", ErrValidation)
	}
	if len(pay.PaymentMethod) > MaxPaymentMethods {
		return fmt.Errorf("%w: PaymentMethod count", ErrValidation)
	}
	for i := range pay.PaymentMethod {
		if err := pay.PaymentMethod[i].ValidateStructural(); err != nil {
			return fmt.Errorf("PaymentMethod[%d]: %w", i, err)
		}
	}
	if strings.TrimSpace(pay.SourceID) == "" {
		return fmt.Errorf("%w: SourceID", ErrValidation)
	}
	if err := pay.SystemEntryDate.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(pay.CustomerID) == "" {
		return fmt.Errorf("%w: CustomerID", ErrValidation)
	}
	if len(pay.Line) == 0 {
		return fmt.Errorf("%w: Line obrigatório", ErrValidation)
	}
	if len(pay.Line) > MaxLinesPerDocument {
		return fmt.Errorf("%w: Line count", ErrValidation)
	}
	for i := range pay.Line {
		if err := pay.Line[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Line[%d]: %w", i, err)
		}
	}
	if err := pay.DocumentTotals.TaxPayable.Validate(); err != nil {
		return err
	}
	if err := pay.DocumentTotals.NetTotal.Validate(); err != nil {
		return err
	}
	if err := pay.DocumentTotals.GrossTotal.Validate(); err != nil {
		return err
	}
	if pay.DocumentTotals.Settlement != nil {
		if err := pay.DocumentTotals.Settlement.SettlementAmount.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ValidateStructural checks SAFPaymentMethod (≠ AGT).
func (m *SAFPaymentMethod) ValidateStructural() error {
	if err := m.PaymentAmount.Validate(); err != nil {
		return err
	}
	return m.PaymentDate.Validate()
}

// ValidateStructural checks PaymentLine (≠ AGT).
func (ln *PaymentLine) ValidateStructural() error {
	if strings.TrimSpace(ln.LineNumber) == "" {
		return fmt.Errorf("%w: LineNumber", ErrValidation)
	}
	if len(ln.SourceDocumentID) == 0 {
		return fmt.Errorf("%w: SourceDocumentID obrigatório", ErrValidation)
	}
	if len(ln.SourceDocumentID) > MaxSourceDocumentIDs {
		return fmt.Errorf("%w: SourceDocumentID count", ErrValidation)
	}
	for i := range ln.SourceDocumentID {
		if strings.TrimSpace(ln.SourceDocumentID[i].OriginatingON) == "" {
			return fmt.Errorf("%w: OriginatingON", ErrValidation)
		}
		if err := ln.SourceDocumentID[i].InvoiceDate.Validate(); err != nil {
			return err
		}
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

// ValidateStructural checks PaymentTax (≠ AGT).
func (t *PaymentTax) ValidateStructural() error {
	if strings.TrimSpace(t.TaxType) == "" || strings.TrimSpace(t.TaxCode) == "" {
		return fmt.Errorf("%w: PaymentTax TaxType/TaxCode", ErrValidation)
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
