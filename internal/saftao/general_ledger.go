package saftao

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// GeneralLedgerAccounts is MasterFiles/GeneralLedgerAccounts (XSD: ≥1 Account; no NumberOfEntries).
type GeneralLedgerAccounts struct {
	Account []GLAccount `xml:"Account"`
}

// GLAccount is MasterFiles/GeneralLedgerAccounts/Account.
// Balances are structural SAFmonetaryType (≥0) — ≠ AO-* chart-of-accounts rules.
type GLAccount struct {
	AccountID            string           `xml:"AccountID"`
	AccountDescription   string           `xml:"AccountDescription"`
	OpeningDebitBalance  Money2           `xml:"OpeningDebitBalance"`
	OpeningCreditBalance Money2           `xml:"OpeningCreditBalance"`
	ClosingDebitBalance  Money2           `xml:"ClosingDebitBalance"`
	ClosingCreditBalance Money2           `xml:"ClosingCreditBalance"`
	GroupingCategory     GroupingCategory `xml:"GroupingCategory"`
	GroupingCode         string           `xml:"GroupingCode,omitempty"`
}

// GroupingCategory is XSD GroupingCategory (GR|GA|GM|AR|AA|AM).
type GroupingCategory string

const (
	GroupingCategoryGR GroupingCategory = "GR"
	GroupingCategoryGA GroupingCategory = "GA"
	GroupingCategoryGM GroupingCategory = "GM"
	GroupingCategoryAR GroupingCategory = "AR"
	GroupingCategoryAA GroupingCategory = "AA"
	GroupingCategoryAM GroupingCategory = "AM"
)

// ValidGroupingCategory reports whether c is in the XSD enumeration.
func ValidGroupingCategory(c GroupingCategory) bool {
	switch c {
	case GroupingCategoryGR, GroupingCategoryGA, GroupingCategoryGM,
		GroupingCategoryAR, GroupingCategoryAA, GroupingCategoryAM:
		return true
	default:
		return false
	}
}

// ValidGLAccountID checks XSD SAFAOGLAccountID (pattern + length 1..30).
func ValidGLAccountID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 30 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '-' || r == '/' || r == '.' || r == '_' || r == '+' || r == '*':
		default:
			return false
		}
	}
	return true
}

// ValidateStructural checks GeneralLedgerAccounts (≠ AGT / AO-*).
func (g *GeneralLedgerAccounts) ValidateStructural() error {
	if g == nil {
		return fmt.Errorf("%w: GeneralLedgerAccounts nil", ErrValidation)
	}
	if len(g.Account) == 0 {
		return fmt.Errorf("%w: Account obrigatório (≥1)", ErrValidation)
	}
	if len(g.Account) > MaxTableEntries {
		return fmt.Errorf("%w: GeneralLedgerAccounts excedeu MaxTableEntries", ErrValidation)
	}
	seen := map[string]struct{}{}
	for i := range g.Account {
		if err := g.Account[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Account[%d]: %w", i, err)
		}
		id := strings.TrimSpace(g.Account[i].AccountID)
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: AccountID duplicado %q", ErrValidation, id)
		}
		seen[id] = struct{}{}
	}
	// Optional GroupingCode must reference an AccountID when present (XSD keyref).
	for i, a := range g.Account {
		code := strings.TrimSpace(a.GroupingCode)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; !ok {
			return fmt.Errorf("%w: Account[%d] GroupingCode sem AccountID", ErrValidation, i)
		}
	}
	return nil
}

// ValidateStructural checks GLAccount (≠ AGT / AO-*).
func (a *GLAccount) ValidateStructural() error {
	if a == nil {
		return fmt.Errorf("%w: Account nil", ErrValidation)
	}
	if !ValidGLAccountID(a.AccountID) {
		return fmt.Errorf("%w: AccountID", ErrValidation)
	}
	desc := strings.TrimSpace(a.AccountDescription)
	if desc == "" || utf8.RuneCountInString(desc) > 100 {
		return fmt.Errorf("%w: AccountDescription", ErrValidation)
	}
	for _, pair := range []struct {
		name string
		m    Money2
	}{
		{"OpeningDebitBalance", a.OpeningDebitBalance},
		{"OpeningCreditBalance", a.OpeningCreditBalance},
		{"ClosingDebitBalance", a.ClosingDebitBalance},
		{"ClosingCreditBalance", a.ClosingCreditBalance},
	} {
		if err := pair.m.Validate(); err != nil {
			return fmt.Errorf("%w: %s", ErrValidation, pair.name)
		}
	}
	if !ValidGroupingCategory(a.GroupingCategory) {
		return fmt.Errorf("%w: GroupingCategory", ErrValidation)
	}
	if code := strings.TrimSpace(a.GroupingCode); code != "" && !ValidGLAccountID(code) {
		return fmt.Errorf("%w: GroupingCode", ErrValidation)
	}
	return nil
}
