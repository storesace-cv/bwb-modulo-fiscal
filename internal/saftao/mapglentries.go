package saftao

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GLEntriesLedgerRecord is one accounting transaction view for SAF-T mapping.
// Missing fields → Omissions (no invention). Reasons must not leak NIF/tokens/payloads.
type GLEntriesLedgerRecord struct {
	ScopeID           string
	DocumentID        string // internal id for omission reports
	TransactionAt     time.Time
	SealedAt          time.Time
	JournalID         string
	JournalDesc       string
	TransactionID     string
	Period            int
	SourceID          string
	Description       string
	DocArchivalNumber string
	TransactionType   TransactionType
	GLPostingDate     Date // empty ⇒ derive from TransactionAt
	CustomerID        string
	SupplierID        string
	DebitLines        []GLEntriesLedgerLine
	CreditLines       []GLEntriesLedgerLine
}

// GLEntriesLedgerLine is one debit or credit line (cents — no float).
type GLEntriesLedgerLine struct {
	RecordID         string
	AccountID        string
	SourceDocumentID string
	Description      string
	AmountCents      int64
	SystemEntryAt    time.Time
}

// GLEntriesLedgerMapConfig configures scope/period for GeneralLedgerEntries mapping.
type GLEntriesLedgerMapConfig struct {
	ScopeID                          string
	PeriodStart                      Date
	PeriodEnd                        Date
	Header                           Header
	AllowedTransactionTypes          []TransactionType
	GeneralLedgerAccounts            []GeneralLedgerAccounts
	Customers                        []Customer
	Suppliers                        []Supplier
	ValidateAgainstXSD               bool
	IncludeEmptyGeneralLedgerEntries bool
	MaxOmissions                     int
}

// MapGLEntriesLedgerToExport filters by scope/period, maps complete records to GeneralLedgerEntries,
// reports omissions, then calls BuildIncrementalExport.
func MapGLEntriesLedgerToExport(cfg GLEntriesLedgerMapConfig, records []GLEntriesLedgerRecord) (*LedgerMapResult, error) {
	if strings.TrimSpace(cfg.ScopeID) == "" {
		return nil, fmt.Errorf("%w: ScopeID obrigatório", ErrValidation)
	}
	if err := cfg.PeriodStart.Validate(); err != nil {
		return nil, fmt.Errorf("%w: PeriodStart", ErrValidation)
	}
	if err := cfg.PeriodEnd.Validate(); err != nil {
		return nil, fmt.Errorf("%w: PeriodEnd", ErrValidation)
	}
	if string(cfg.PeriodStart) > string(cfg.PeriodEnd) {
		return nil, fmt.Errorf("%w: PeriodStart>PeriodEnd", ErrValidation)
	}
	maxOm := cfg.MaxOmissions
	if maxOm <= 0 {
		maxOm = MaxTableEntries
	}

	sorted := append([]GLEntriesLedgerRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].JournalID != sorted[j].JournalID {
			return sorted[i].JournalID < sorted[j].JournalID
		}
		if sorted[i].DocumentID != sorted[j].DocumentID {
			return sorted[i].DocumentID < sorted[j].DocumentID
		}
		return sorted[i].TransactionAt.Before(sorted[j].TransactionAt)
	})

	journals := map[string]*Journal{}
	var journalOrder []string
	var omissions []Omission
	mapped := 0

	for _, rec := range sorted {
		if strings.TrimSpace(rec.ScopeID) != cfg.ScopeID {
			continue
		}
		if rec.TransactionAt.IsZero() {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "transaction_at", Reason: "TransactionAt ausente"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		txDate, err := dateFromTime(rec.TransactionAt)
		if err != nil {
			omissions = append(omissions, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: "transaction_at", Reason: "data inválida"})
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		if string(txDate) < string(cfg.PeriodStart) || string(txDate) > string(cfg.PeriodEnd) {
			continue
		}
		tx, oms := mapOneGLEntriesRecord(rec, txDate)
		if len(oms) > 0 {
			omissions = append(omissions, oms...)
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		jid := strings.TrimSpace(rec.JournalID)
		j, ok := journals[jid]
		if !ok {
			j = &Journal{JournalID: jid, Description: strings.TrimSpace(rec.JournalDesc)}
			journals[jid] = j
			journalOrder = append(journalOrder, jid)
		}
		j.Transaction = append(j.Transaction, tx)
		mapped++
		if mapped > MaxTableEntries {
			return nil, fmt.Errorf("%w: transactions excederam MaxTableEntries", ErrValidation)
		}
	}
	if len(omissions) > maxOm {
		return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
	}

	var gle *GeneralLedgerEntries
	if mapped > 0 {
		var outJ []Journal
		for _, jid := range journalOrder {
			outJ = append(outJ, *journals[jid])
		}
		gle = &GeneralLedgerEntries{
			NumberOfEntries: strconv.Itoa(mapped),
			TotalDebit:      MustDecimal("0.00"), // recomputed in BuildIncrementalExport filter path
			TotalCredit:     MustDecimal("0.00"),
			Journal:         outJ,
		}
		// Totals will be recalculated by filterGeneralLedgerEntries; seed with line sums for ValidateStructural.
		td := MustDecimal("0.00")
		tc := MustDecimal("0.00")
		for _, j := range outJ {
			for _, tx := range j.Transaction {
				for _, ln := range tx.Lines.DebitLine {
					sum, err := addMoney2AsDecimal(td, ln.DebitAmount)
					if err != nil {
						return nil, err
					}
					td = sum
				}
				for _, ln := range tx.Lines.CreditLine {
					sum, err := addMoney2AsDecimal(tc, ln.CreditAmount)
					if err != nil {
						return nil, err
					}
					tc = sum
				}
			}
		}
		gle.TotalDebit = td
		gle.TotalCredit = tc
	} else if cfg.IncludeEmptyGeneralLedgerEntries {
		gle = &GeneralLedgerEntries{
			NumberOfEntries: "0",
			TotalDebit:      MustDecimal("0.00"),
			TotalCredit:     MustDecimal("0.00"),
		}
	}

	hdr := cfg.Header
	hdr.StartDate = string(cfg.PeriodStart)
	hdr.EndDate = string(cfg.PeriodEnd)

	exp, err := BuildIncrementalExport(ExportRequest{
		Header:                           hdr,
		EnabledGroups:                    []DocumentGroup{GroupSalesInvoices},
		AllowedInvoiceTypes:              []InvoiceType{InvoiceTypeFT},
		IncludeEmptySalesTotals:          true,
		GeneralLedgerAccounts:            cfg.GeneralLedgerAccounts,
		GeneralLedgerEntries:             gle,
		AllowedTransactionTypes:          cfg.AllowedTransactionTypes,
		IncludeEmptyGeneralLedgerEntries: cfg.IncludeEmptyGeneralLedgerEntries,
		Customers:                        cfg.Customers,
		Suppliers:                        cfg.Suppliers,
		ValidateAgainstXSD:               cfg.ValidateAgainstXSD,
	})
	if err != nil {
		return nil, err
	}
	return &LedgerMapResult{
		Export:    exp,
		Mapped:    mapped,
		Omitted:   countOmittedDocs(omissions),
		Omissions: omissions,
	}, nil
}

func mapOneGLEntriesRecord(rec GLEntriesLedgerRecord, txDate Date) (Transaction, []Omission) {
	var oms []Omission
	omit := func(field, reason string) {
		oms = append(oms, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: field, Reason: reason})
	}

	if !ValidJournalID(rec.JournalID) {
		omit("journal_id", "JournalID ausente ou inválido")
	}
	if !validMandatoryMax200(rec.JournalDesc) {
		omit("journal_desc", "Journal Description ausente ou inválida")
	}
	tid := strings.TrimSpace(rec.TransactionID)
	if !ValidTransactionID(tid) {
		omit("transaction_id", "TransactionID ausente ou inválido (não inventado)")
	}
	if rec.Period < 1 || rec.Period > 16 {
		omit("period", "Period fora de 1..16")
	}
	sourceID := strings.TrimSpace(rec.SourceID)
	if !validMandatoryMax30(sourceID) {
		omit("source_id", "SourceID ausente")
	}
	if !validMandatoryMax200(rec.Description) {
		omit("description", "Description ausente")
	}
	if !ValidDocArchivalNumber(rec.DocArchivalNumber) {
		omit("doc_archival", "DocArchivalNumber ausente ou inválido")
	}
	if !ValidTransactionType(rec.TransactionType) {
		omit("transaction_type", "TransactionType ausente ou fora do XSD")
	}
	posting := rec.GLPostingDate
	if strings.TrimSpace(string(posting)) == "" {
		posting = txDate
	} else if err := posting.Validate(); err != nil {
		omit("gl_posting_date", "GLPostingDate inválida")
	}
	cust := strings.TrimSpace(rec.CustomerID)
	supp := strings.TrimSpace(rec.SupplierID)
	if cust != "" && supp != "" {
		omit("party", "CustomerID e SupplierID mutuamente exclusivos")
	}

	if len(rec.DebitLines) < 1 || len(rec.CreditLines) < 1 {
		omit("lines", "exige ≥1 débito e ≥1 crédito")
	}

	var debits []DebitLine
	for i, ln := range rec.DebitLines {
		if !validMandatoryMax30(ln.RecordID) {
			omit("debit_record_id", "RecordID débito inválido")
			break
		}
		if !ValidGLAccountID(ln.AccountID) {
			omit("debit_account", "AccountID débito inválido")
			break
		}
		if ln.AmountCents <= 0 {
			omit("debit_amount", "AmountCents débito inválido")
			break
		}
		if !validMandatoryMax200(ln.Description) {
			omit("debit_desc", "Description débito ausente")
			break
		}
		entryAt := ln.SystemEntryAt
		if entryAt.IsZero() {
			entryAt = rec.SealedAt
		}
		if entryAt.IsZero() {
			entryAt = rec.TransactionAt
		}
		dt, err := dateTimeFromTime(entryAt)
		if err != nil {
			omit("debit_entry", "SystemEntryDate débito inválido")
			break
		}
		_ = i
		debits = append(debits, DebitLine{
			RecordID:         strings.TrimSpace(ln.RecordID),
			AccountID:        strings.TrimSpace(ln.AccountID),
			SourceDocumentID: strings.TrimSpace(ln.SourceDocumentID),
			SystemEntryDate:  dt,
			Description:      strings.TrimSpace(ln.Description),
			DebitAmount:      MustMoney2(formatCents(ln.AmountCents)),
		})
	}

	var credits []CreditLine
	for _, ln := range rec.CreditLines {
		if !validMandatoryMax30(ln.RecordID) {
			omit("credit_record_id", "RecordID crédito inválido")
			break
		}
		if !ValidGLAccountID(ln.AccountID) {
			omit("credit_account", "AccountID crédito inválido")
			break
		}
		if ln.AmountCents <= 0 {
			omit("credit_amount", "AmountCents crédito inválido")
			break
		}
		if !validMandatoryMax200(ln.Description) {
			omit("credit_desc", "Description crédito ausente")
			break
		}
		entryAt := ln.SystemEntryAt
		if entryAt.IsZero() {
			entryAt = rec.SealedAt
		}
		if entryAt.IsZero() {
			entryAt = rec.TransactionAt
		}
		dt, err := dateTimeFromTime(entryAt)
		if err != nil {
			omit("credit_entry", "SystemEntryDate crédito inválido")
			break
		}
		credits = append(credits, CreditLine{
			RecordID:         strings.TrimSpace(ln.RecordID),
			AccountID:        strings.TrimSpace(ln.AccountID),
			SourceDocumentID: strings.TrimSpace(ln.SourceDocumentID),
			SystemEntryDate:  dt,
			Description:      strings.TrimSpace(ln.Description),
			CreditAmount:     MustMoney2(formatCents(ln.AmountCents)),
		})
	}

	if len(oms) > 0 {
		return Transaction{}, oms
	}

	return Transaction{
		TransactionID:     tid,
		Period:            rec.Period,
		TransactionDate:   txDate,
		SourceID:          sourceID,
		Description:       strings.TrimSpace(rec.Description),
		DocArchivalNumber: strings.TrimSpace(rec.DocArchivalNumber),
		TransactionType:   rec.TransactionType,
		GLPostingDate:     posting,
		CustomerID:        cust,
		SupplierID:        supp,
		Lines: TransactionLines{
			DebitLine:  debits,
			CreditLine: credits,
		},
	}, nil
}
