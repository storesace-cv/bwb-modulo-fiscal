package saftao

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// GeneralLedgerEntries is AuditFile/GeneralLedgerEntries (XSD sequence + Journal*).
// Opening movements belong in GeneralLedgerAccounts — structural note from XSD docs; ≠ AO-*.
type GeneralLedgerEntries struct {
	NumberOfEntries string        `xml:"NumberOfEntries"`
	TotalDebit      DecimalNonNeg `xml:"TotalDebit"`
	TotalCredit     DecimalNonNeg `xml:"TotalCredit"`
	Journal         []Journal     `xml:"Journal,omitempty"`
}

// Journal is GeneralLedgerEntries/Journal.
type Journal struct {
	JournalID   string        `xml:"JournalID"`
	Description string        `xml:"Description"`
	Transaction []Transaction `xml:"Transaction,omitempty"`
}

// Transaction is Journal/Transaction (CustomerID XOR SupplierID optional).
type Transaction struct {
	TransactionID     string           `xml:"TransactionID"`
	Period            int              `xml:"Period"`
	TransactionDate   Date             `xml:"TransactionDate"`
	SourceID          string           `xml:"SourceID"`
	Description       string           `xml:"Description"`
	DocArchivalNumber string           `xml:"DocArchivalNumber"`
	TransactionType   TransactionType  `xml:"TransactionType"`
	GLPostingDate     Date             `xml:"GLPostingDate"`
	CustomerID        string           `xml:"CustomerID,omitempty"`
	SupplierID        string           `xml:"SupplierID,omitempty"`
	Lines             TransactionLines `xml:"Lines"`
}

// TransactionLines holds DebitLine* then CreditLine* (≥1 each per XSD).
type TransactionLines struct {
	DebitLine  []DebitLine  `xml:"DebitLine"`
	CreditLine []CreditLine `xml:"CreditLine"`
}

// DebitLine is Lines/DebitLine.
type DebitLine struct {
	RecordID         string   `xml:"RecordID"`
	AccountID        string   `xml:"AccountID"`
	SourceDocumentID string   `xml:"SourceDocumentID,omitempty"`
	SystemEntryDate  DateTime `xml:"SystemEntryDate"`
	Description      string   `xml:"Description"`
	DebitAmount      Money2   `xml:"DebitAmount"`
}

// CreditLine is Lines/CreditLine.
type CreditLine struct {
	RecordID         string   `xml:"RecordID"`
	AccountID        string   `xml:"AccountID"`
	SourceDocumentID string   `xml:"SourceDocumentID,omitempty"`
	SystemEntryDate  DateTime `xml:"SystemEntryDate"`
	Description      string   `xml:"Description"`
	CreditAmount     Money2   `xml:"CreditAmount"`
}

// TransactionType is XSD TransactionType (N|R|A|J).
type TransactionType string

const (
	TransactionTypeN TransactionType = "N"
	TransactionTypeR TransactionType = "R"
	TransactionTypeA TransactionType = "A"
	TransactionTypeJ TransactionType = "J"
)

// ValidTransactionType reports whether t is in the XSD enumeration.
func ValidTransactionType(t TransactionType) bool {
	switch t {
	case TransactionTypeN, TransactionTypeR, TransactionTypeA, TransactionTypeJ:
		return true
	default:
		return false
	}
}

var (
	journalIDPattern     = regexp.MustCompile(`^[^ ]{1,30}$`)
	docArchivalPattern   = regexp.MustCompile(`^[^ ]{1,20}$`)
	transactionIDPattern = regexp.MustCompile(`^[1-9][0-9]{3}-[01][0-9]-[0-3][0-9] [^ ]{1,30} [^ ]{1,20}$`)
)

// ValidJournalID checks XSD SAFAOJournalID.
func ValidJournalID(id string) bool {
	return journalIDPattern.MatchString(strings.TrimSpace(id))
}

// ValidDocArchivalNumber checks XSD SAFTAODocArchivalNumber.
func ValidDocArchivalNumber(s string) bool {
	return docArchivalPattern.MatchString(strings.TrimSpace(s))
}

// ValidTransactionID checks XSD SAFAOTransactionID pattern (date JournalID DocArchival).
func ValidTransactionID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 70 {
		return false
	}
	return transactionIDPattern.MatchString(id)
}

func validMandatoryMax30(s string) bool {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	return n >= 1 && n <= 30
}

func validMandatoryMax200(s string) bool {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	return n >= 1 && n <= 200
}

// ValidateStructural checks GeneralLedgerEntries (≠ AGT / AO-*).
func (g *GeneralLedgerEntries) ValidateStructural() error {
	if g == nil {
		return fmt.Errorf("%w: GeneralLedgerEntries nil", ErrValidation)
	}
	if strings.TrimSpace(g.NumberOfEntries) == "" {
		return fmt.Errorf("%w: NumberOfEntries", ErrValidation)
	}
	if err := g.TotalDebit.Validate(); err != nil {
		return fmt.Errorf("%w: TotalDebit", ErrValidation)
	}
	if err := g.TotalCredit.Validate(); err != nil {
		return fmt.Errorf("%w: TotalCredit", ErrValidation)
	}
	if len(g.Journal) > MaxTableEntries {
		return fmt.Errorf("%w: Journal excedeu MaxTableEntries", ErrValidation)
	}
	seenJ := map[string]struct{}{}
	txCount := 0
	for i := range g.Journal {
		if err := g.Journal[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Journal[%d]: %w", i, err)
		}
		jid := strings.TrimSpace(g.Journal[i].JournalID)
		if _, ok := seenJ[jid]; ok {
			return fmt.Errorf("%w: JournalID duplicado %q", ErrValidation, jid)
		}
		seenJ[jid] = struct{}{}
		txCount += len(g.Journal[i].Transaction)
	}
	if txCount > MaxTableEntries {
		return fmt.Errorf("%w: Transaction excedeu MaxTableEntries", ErrValidation)
	}
	return nil
}

// ValidateStructural checks Journal.
func (j *Journal) ValidateStructural() error {
	if j == nil {
		return fmt.Errorf("%w: Journal nil", ErrValidation)
	}
	if !ValidJournalID(j.JournalID) {
		return fmt.Errorf("%w: JournalID", ErrValidation)
	}
	if !validMandatoryMax200(j.Description) {
		return fmt.Errorf("%w: Description", ErrValidation)
	}
	if len(j.Transaction) > MaxTableEntries {
		return fmt.Errorf("%w: Transaction excedeu MaxTableEntries", ErrValidation)
	}
	seenTX := map[string]struct{}{}
	for i := range j.Transaction {
		if err := j.Transaction[i].ValidateStructural(); err != nil {
			return fmt.Errorf("Transaction[%d]: %w", i, err)
		}
		tid := strings.TrimSpace(j.Transaction[i].TransactionID)
		if _, ok := seenTX[tid]; ok {
			return fmt.Errorf("%w: TransactionID duplicado %q", ErrValidation, tid)
		}
		seenTX[tid] = struct{}{}
	}
	return nil
}

// ValidateStructural checks Transaction.
func (t *Transaction) ValidateStructural() error {
	if t == nil {
		return fmt.Errorf("%w: Transaction nil", ErrValidation)
	}
	if !ValidTransactionID(t.TransactionID) {
		return fmt.Errorf("%w: TransactionID", ErrValidation)
	}
	if t.Period < 1 || t.Period > 16 {
		return fmt.Errorf("%w: Period (1..16)", ErrValidation)
	}
	if err := t.TransactionDate.Validate(); err != nil {
		return fmt.Errorf("%w: TransactionDate", ErrValidation)
	}
	if !validMandatoryMax30(t.SourceID) {
		return fmt.Errorf("%w: SourceID", ErrValidation)
	}
	if !validMandatoryMax200(t.Description) {
		return fmt.Errorf("%w: Description", ErrValidation)
	}
	if !ValidDocArchivalNumber(t.DocArchivalNumber) {
		return fmt.Errorf("%w: DocArchivalNumber", ErrValidation)
	}
	if !ValidTransactionType(t.TransactionType) {
		return fmt.Errorf("%w: TransactionType", ErrValidation)
	}
	if err := t.GLPostingDate.Validate(); err != nil {
		return fmt.Errorf("%w: GLPostingDate", ErrValidation)
	}
	cust := strings.TrimSpace(t.CustomerID)
	supp := strings.TrimSpace(t.SupplierID)
	if cust != "" && supp != "" {
		return fmt.Errorf("%w: CustomerID e SupplierID mutuamente exclusivos", ErrValidation)
	}
	if cust != "" && !validMandatoryMax30(cust) {
		return fmt.Errorf("%w: CustomerID", ErrValidation)
	}
	if supp != "" && !validMandatoryMax30(supp) {
		return fmt.Errorf("%w: SupplierID", ErrValidation)
	}
	return t.Lines.ValidateStructural()
}

// ValidateStructural checks Lines (≥1 DebitLine and ≥1 CreditLine).
func (l *TransactionLines) ValidateStructural() error {
	if l == nil {
		return fmt.Errorf("%w: Lines nil", ErrValidation)
	}
	if len(l.DebitLine) < 1 || len(l.CreditLine) < 1 {
		return fmt.Errorf("%w: Lines exige ≥1 DebitLine e ≥1 CreditLine", ErrValidation)
	}
	if len(l.DebitLine)+len(l.CreditLine) > MaxTableEntries {
		return fmt.Errorf("%w: Lines excedeu MaxTableEntries", ErrValidation)
	}
	for i := range l.DebitLine {
		if err := l.DebitLine[i].ValidateStructural(); err != nil {
			return fmt.Errorf("DebitLine[%d]: %w", i, err)
		}
	}
	for i := range l.CreditLine {
		if err := l.CreditLine[i].ValidateStructural(); err != nil {
			return fmt.Errorf("CreditLine[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateStructural checks DebitLine.
func (d *DebitLine) ValidateStructural() error {
	if d == nil {
		return fmt.Errorf("%w: DebitLine nil", ErrValidation)
	}
	if !validMandatoryMax30(d.RecordID) {
		return fmt.Errorf("%w: RecordID", ErrValidation)
	}
	if !ValidGLAccountID(d.AccountID) {
		return fmt.Errorf("%w: AccountID", ErrValidation)
	}
	if src := strings.TrimSpace(d.SourceDocumentID); src != "" && !validMandatoryMax30(src) {
		return fmt.Errorf("%w: SourceDocumentID", ErrValidation)
	}
	if err := d.SystemEntryDate.Validate(); err != nil {
		return fmt.Errorf("%w: SystemEntryDate", ErrValidation)
	}
	if !validMandatoryMax200(d.Description) {
		return fmt.Errorf("%w: Description", ErrValidation)
	}
	if err := d.DebitAmount.Validate(); err != nil {
		return fmt.Errorf("%w: DebitAmount", ErrValidation)
	}
	return nil
}

// ValidateStructural checks CreditLine.
func (c *CreditLine) ValidateStructural() error {
	if c == nil {
		return fmt.Errorf("%w: CreditLine nil", ErrValidation)
	}
	if !validMandatoryMax30(c.RecordID) {
		return fmt.Errorf("%w: RecordID", ErrValidation)
	}
	if !ValidGLAccountID(c.AccountID) {
		return fmt.Errorf("%w: AccountID", ErrValidation)
	}
	if src := strings.TrimSpace(c.SourceDocumentID); src != "" && !validMandatoryMax30(src) {
		return fmt.Errorf("%w: SourceDocumentID", ErrValidation)
	}
	if err := c.SystemEntryDate.Validate(); err != nil {
		return fmt.Errorf("%w: SystemEntryDate", ErrValidation)
	}
	if !validMandatoryMax200(c.Description) {
		return fmt.Errorf("%w: Description", ErrValidation)
	}
	if err := c.CreditAmount.Validate(); err != nil {
		return fmt.Errorf("%w: CreditAmount", ErrValidation)
	}
	return nil
}
