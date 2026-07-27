package saftao

import (
	"fmt"
	"strings"
)

// PurchaseInvoices is XSD SourceDocuments/PurchaseInvoices (compras a fornecedores).
// XSD shape: NumberOfEntries + Invoice* — NO TotalDebit/TotalCredit; Invoice has NO Line.
type PurchaseInvoices struct {
	NumberOfEntries string            `xml:"NumberOfEntries"`
	Invoice         []PurchaseInvoice `xml:"Invoice,omitempty"`
}

// PurchaseInvoice is PurchaseInvoices/Invoice (distinct from SalesInvoices/Invoice).
// InvoiceNo is free text max 60 (≠ Sales InvoiceNo pattern). No Line children in XSD.
type PurchaseInvoice struct {
	InvoiceNo      string         `xml:"InvoiceNo"`
	Hash           string         `xml:"Hash"` // PendingHashAlgorithm; "0" allowed when no validation duty
	SourceID       string         `xml:"SourceID"`
	Period         string         `xml:"Period,omitempty"`
	InvoiceDate    Date           `xml:"InvoiceDate"`
	PurchaseType   PurchaseType   `xml:"PurchaseType"`
	SupplierID     string         `xml:"SupplierID"`
	DocumentTotals DocumentTotals `xml:"DocumentTotals"`
}

// PurchaseType is XSD PurchaseType enumeration (PendingPurchaseTypeSemantics).
type PurchaseType string

const (
	PurchaseTypeFT PurchaseType = "FT"
	PurchaseTypeFR PurchaseType = "FR"
	PurchaseTypeGF PurchaseType = "GF"
	PurchaseTypeFG PurchaseType = "FG"
	PurchaseTypeAC PurchaseType = "AC"
	PurchaseTypeAR PurchaseType = "AR"
	PurchaseTypeAF PurchaseType = "AF"
	PurchaseTypeTV PurchaseType = "TV"
	PurchaseTypeNL PurchaseType = "NL"
	PurchaseTypeNC PurchaseType = "NC"
	PurchaseTypeRC PurchaseType = "RC" // regime de caixa only — semantics pending
)

// ValidPurchaseType reports whether t is in the XSD enumeration.
func ValidPurchaseType(t PurchaseType) bool {
	switch t {
	case PurchaseTypeFT, PurchaseTypeFR, PurchaseTypeGF, PurchaseTypeFG,
		PurchaseTypeAC, PurchaseTypeAR, PurchaseTypeAF, PurchaseTypeTV,
		PurchaseTypeNL, PurchaseTypeNC, PurchaseTypeRC:
		return true
	default:
		return false
	}
}

// ValidateStructural checks PurchaseInvoices (≠ AGT). Documents absence of Line in XSD.
func (p *PurchaseInvoices) ValidateStructural() error {
	if p == nil {
		return fmt.Errorf("%w: PurchaseInvoices nil", ErrValidation)
	}
	if strings.TrimSpace(p.NumberOfEntries) == "" {
		return fmt.Errorf("%w: NumberOfEntries", ErrValidation)
	}
	if len(p.Invoice) > MaxTableEntries {
		return fmt.Errorf("%w: PurchaseInvoices excedeu MaxTableEntries", ErrValidation)
	}
	for i := range p.Invoice {
		if err := p.Invoice[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Invoice[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks PurchaseInvoice (≠ AGT).
func (inv *PurchaseInvoice) ValidateStructural() error {
	if inv == nil {
		return fmt.Errorf("%w: PurchaseInvoice nil", ErrValidation)
	}
	no := strings.TrimSpace(inv.InvoiceNo)
	if no == "" || len(no) > 60 {
		return fmt.Errorf("%w: InvoiceNo", ErrValidation)
	}
	if strings.TrimSpace(inv.Hash) == "" || len(inv.Hash) > 172 {
		return fmt.Errorf("%w: Hash", ErrValidation)
	}
	if strings.TrimSpace(inv.SourceID) == "" {
		return fmt.Errorf("%w: SourceID", ErrValidation)
	}
	if err := inv.InvoiceDate.Validate(); err != nil {
		return err
	}
	if !ValidPurchaseType(inv.PurchaseType) {
		return fmt.Errorf("%w: PurchaseType", ErrValidation)
	}
	if strings.TrimSpace(inv.SupplierID) == "" {
		return fmt.Errorf("%w: SupplierID", ErrValidation)
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
