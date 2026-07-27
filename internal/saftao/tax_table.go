package saftao

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// TaxTable is MasterFiles/TaxTable (XSD: ≥1 TaxTableEntry).
type TaxTable struct {
	TaxTableEntry []TaxTableEntry `xml:"TaxTableEntry"`
}

// TaxTableEntry is MasterFiles/TaxTable/TaxTableEntry.
// TaxPercentage XOR TaxAmount (XSD choice). Rates/codes here are structural — ≠ AO-* rates.
type TaxTableEntry struct {
	TaxType           TaxType `xml:"TaxType"`
	TaxCountryRegion  string  `xml:"TaxCountryRegion,omitempty"`
	TaxCode           string  `xml:"TaxCode"`
	Description       string  `xml:"Description"`
	TaxExpirationDate *Date   `xml:"TaxExpirationDate,omitempty"`
	TaxPercentage     *string `xml:"TaxPercentage,omitempty"`
	TaxAmount         *Money2 `xml:"TaxAmount,omitempty"`
}

// TaxType is XSD TaxType enumeration (IVA|IS|NS).
type TaxType string

const (
	TaxTypeIVA TaxType = "IVA"
	TaxTypeIS  TaxType = "IS"
	TaxTypeNS  TaxType = "NS"
)

// ValidTaxType reports whether t is in the XSD enumeration.
func ValidTaxType(t TaxType) bool {
	switch t {
	case TaxTypeIVA, TaxTypeIS, TaxTypeNS:
		return true
	default:
		return false
	}
}

// ValidTaxTableEntryTaxCode checks XSD TaxTableEntryTaxCode pattern
// (RED|INT|NOR|ISE|OUT|([0-9.])*|NS|NA), length 1..10 — structural only.
func ValidTaxTableEntryTaxCode(code string) bool {
	c := strings.TrimSpace(code)
	if c == "" || len(c) > 10 {
		return false
	}
	switch c {
	case "RED", "INT", "NOR", "ISE", "OUT", "NS", "NA":
		return true
	}
	for _, r := range c {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

// ValidateStructural checks TaxTable (≠ AGT / AO-*).
func (tt *TaxTable) ValidateStructural() error {
	if tt == nil {
		return fmt.Errorf("%w: TaxTable nil", ErrValidation)
	}
	if len(tt.TaxTableEntry) == 0 {
		return fmt.Errorf("%w: TaxTableEntry obrigatório (≥1)", ErrValidation)
	}
	if len(tt.TaxTableEntry) > MaxTableEntries {
		return fmt.Errorf("%w: TaxTable excedeu MaxTableEntries", ErrValidation)
	}
	for i := range tt.TaxTableEntry {
		if err := tt.TaxTableEntry[i].ValidateStructural(); err != nil {
			return fmt.Errorf("TaxTableEntry[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks TaxTableEntry (≠ AGT / AO-*).
func (e *TaxTableEntry) ValidateStructural() error {
	if e == nil {
		return fmt.Errorf("%w: TaxTableEntry nil", ErrValidation)
	}
	if !ValidTaxType(e.TaxType) {
		return fmt.Errorf("%w: TaxType", ErrValidation)
	}
	if region := strings.TrimSpace(e.TaxCountryRegion); region != "" {
		n := utf8.RuneCountInString(region)
		if n < 2 || n > 6 {
			return fmt.Errorf("%w: TaxCountryRegion length", ErrValidation)
		}
	}
	if !ValidTaxTableEntryTaxCode(e.TaxCode) {
		return fmt.Errorf("%w: TaxCode", ErrValidation)
	}
	desc := strings.TrimSpace(e.Description)
	if desc == "" || utf8.RuneCountInString(desc) > 255 {
		return fmt.Errorf("%w: Description", ErrValidation)
	}
	if e.TaxExpirationDate != nil {
		if err := e.TaxExpirationDate.Validate(); err != nil {
			return fmt.Errorf("%w: TaxExpirationDate", ErrValidation)
		}
	}
	hasPct := e.TaxPercentage != nil && strings.TrimSpace(*e.TaxPercentage) != ""
	hasAmt := e.TaxAmount != nil
	if hasPct == hasAmt {
		return fmt.Errorf("%w: TaxPercentage XOR TaxAmount", ErrValidation)
	}
	if hasAmt {
		return e.TaxAmount.Validate()
	}
	return nil
}
