package saftao

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PaymentLedgerRecord is a sealed payment/receipt view for SAF-T mapping.
// Persistence adapters fill this; missing SAF-T fields produce Omissions (no invention).
// Reasons must not include NIF, tokens, or full fiscal payloads.
type PaymentLedgerRecord struct {
	ScopeID       string
	DocumentID    string
	TransactionAt time.Time
	SealedAt      time.Time

	// Optional SAF-T enrichments — empty triggers omission.
	PaymentRefNo       string
	PaymentType        PaymentType
	SourceID           string
	CustomerID         string
	PaymentStatus      PaymentStatus
	SourcePayment      SourcePayment
	PaymentStatusAt    time.Time
	PaymentMechanism   string // e.g. NU — structural allowlist only when set
	PaymentAmountCents int64
	PaymentDate        Date // empty ⇒ derive from TransactionAt
	Lines              []PaymentLedgerLine
	TaxPayableCents    int64
	NetTotalCents      int64
	GrossTotalCents    int64
}

// PaymentLedgerLine is one payment line (cents — no float).
type PaymentLedgerLine struct {
	LineNo        int
	OriginatingON string
	InvoiceDate   Date
	DebitCents    int64
	CreditCents   int64
}

// PaymentLedgerMapConfig configures scope/period filters for payment mapping.
type PaymentLedgerMapConfig struct {
	ScopeID              string
	PeriodStart          Date
	PeriodEnd            Date
	Header               Header
	AllowedPaymentTypes  []PaymentType
	Customers            []Customer // MasterFiles pass-through when mapping succeeds
	ValidateAgainstXSD   bool
	IncludeEmptyPayments bool
	TaxTable             *TaxTable
	MaxOmissions         int
}

// MapPaymentsLedgerToExport filters by scope/period, maps complete records to Payments,
// reports omissions for incomplete ones, then calls BuildIncrementalExport.
func MapPaymentsLedgerToExport(cfg PaymentLedgerMapConfig, records []PaymentLedgerRecord) (*LedgerMapResult, error) {
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

	sorted := append([]PaymentLedgerRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].DocumentID != sorted[j].DocumentID {
			return sorted[i].DocumentID < sorted[j].DocumentID
		}
		return sorted[i].TransactionAt.Before(sorted[j].TransactionAt)
	})

	var (
		payments  []Payment
		omissions []Omission
		customers []Customer
		custSeen  = map[string]struct{}{}
	)
	for _, c := range cfg.Customers {
		if id := strings.TrimSpace(c.CustomerID); id != "" {
			if _, ok := custSeen[id]; !ok {
				custSeen[id] = struct{}{}
				customers = append(customers, c)
			}
		}
	}

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
		pay, cust, oms := mapOnePaymentRecord(rec, txDate)
		if len(oms) > 0 {
			omissions = append(omissions, oms...)
			if len(omissions) > maxOm {
				return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
			}
			continue
		}
		payments = append(payments, pay)
		if cust.CustomerID != "" {
			if _, ok := custSeen[cust.CustomerID]; !ok {
				custSeen[cust.CustomerID] = struct{}{}
				customers = append(customers, cust)
			}
		}
		if len(payments) > MaxTableEntries {
			return nil, fmt.Errorf("%w: payments excederam MaxTableEntries", ErrValidation)
		}
	}
	if len(omissions) > maxOm {
		return nil, fmt.Errorf("%w: omissões excederam MaxOmissions", ErrValidation)
	}

	hdr := cfg.Header
	hdr.StartDate = string(cfg.PeriodStart)
	hdr.EndDate = string(cfg.PeriodEnd)

	exp, err := BuildIncrementalExport(ExportRequest{
		Header:                     hdr,
		EnabledGroups:              []DocumentGroup{GroupPayments},
		AllowedPaymentTypes:        cfg.AllowedPaymentTypes,
		Customers:                  customers,
		TaxTable:                   cfg.TaxTable,
		Payments:                   payments,
		IncludeEmptyPaymentsTotals: cfg.IncludeEmptyPayments,
		ValidateAgainstXSD:         cfg.ValidateAgainstXSD,
	})
	if err != nil {
		return nil, err
	}
	return &LedgerMapResult{
		Export:    exp,
		Mapped:    len(payments),
		Omitted:   countOmittedDocs(omissions),
		Omissions: omissions,
	}, nil
}

func mapOnePaymentRecord(rec PaymentLedgerRecord, txDate Date) (Payment, Customer, []Omission) {
	var oms []Omission
	omit := func(field, reason string) {
		oms = append(oms, Omission{ScopeID: rec.ScopeID, DocumentID: rec.DocumentID, Field: field, Reason: reason})
	}

	ref := strings.TrimSpace(rec.PaymentRefNo)
	if ref == "" {
		omit("payment_ref_no", "PaymentRefNo ausente (não inventado)")
	} else if err := ValidatePaymentRefNo(ref); err != nil {
		omit("payment_ref_no", "PaymentRefNo inválido")
	}
	if !ValidPaymentType(rec.PaymentType) {
		omit("payment_type", "PaymentType ausente ou fora do XSD")
	}
	sourceID := strings.TrimSpace(rec.SourceID)
	if sourceID == "" {
		omit("source_id", "SourceID ausente")
	}
	customerID := strings.TrimSpace(rec.CustomerID)
	if customerID == "" {
		omit("customer_id", "CustomerID SAF-T ausente (não usar identificador fiscal cru)")
	}
	if !ValidPaymentStatus(rec.PaymentStatus) {
		omit("payment_status", "PaymentStatus ausente ou inválido")
	}
	if !ValidSourcePayment(rec.SourcePayment) {
		omit("source_payment", "SourcePayment ausente ou inválido")
	}
	if len(rec.Lines) == 0 {
		omit("lines", "pagamento sem linhas")
	}
	if len(oms) > 0 {
		return Payment{}, Customer{}, oms
	}

	statusAt := rec.PaymentStatusAt
	if statusAt.IsZero() {
		statusAt = rec.SealedAt
	}
	if statusAt.IsZero() {
		statusAt = rec.TransactionAt
	}
	statusDT, err := dateTimeFromTime(statusAt)
	if err != nil {
		omit("payment_status_at", "instante inválido")
		return Payment{}, Customer{}, oms
	}
	entryDT, err := dateTimeFromTime(rec.SealedAt)
	if err != nil || rec.SealedAt.IsZero() {
		entryDT, err = dateTimeFromTime(rec.TransactionAt)
	}
	if err != nil {
		omit("system_entry_date", "instante inválido")
		return Payment{}, Customer{}, oms
	}

	payDate := rec.PaymentDate
	if strings.TrimSpace(string(payDate)) == "" {
		payDate = txDate
	} else if err := payDate.Validate(); err != nil {
		omit("payment_date", "PaymentDate inválida")
		return Payment{}, Customer{}, oms
	}

	var lines []PaymentLine
	for _, ln := range rec.Lines {
		if ln.LineNo <= 0 {
			omit("line_no", "LineNo inválido")
			return Payment{}, Customer{}, oms
		}
		if strings.TrimSpace(ln.OriginatingON) == "" {
			omit("originating_on", "OriginatingON ausente")
			return Payment{}, Customer{}, oms
		}
		if err := ln.InvoiceDate.Validate(); err != nil {
			omit("invoice_date", "InvoiceDate inválida")
			return Payment{}, Customer{}, oms
		}
		hasDebit := ln.DebitCents > 0
		hasCredit := ln.CreditCents > 0
		if hasDebit == hasCredit {
			omit("line_amount", "Debit XOR Credit em cents")
			return Payment{}, Customer{}, oms
		}
		var debit, credit *Money2
		if hasDebit {
			m := MustMoney2(formatCents(ln.DebitCents))
			debit = &m
		}
		if hasCredit {
			m := MustMoney2(formatCents(ln.CreditCents))
			credit = &m
		}
		lines = append(lines, PaymentLine{
			LineNumber: strconv.Itoa(ln.LineNo),
			SourceDocumentID: []SourceDocumentID{{
				OriginatingON: ln.OriginatingON,
				InvoiceDate:   ln.InvoiceDate,
			}},
			DebitAmount:  debit,
			CreditAmount: credit,
		})
	}

	mech := strings.TrimSpace(rec.PaymentMechanism)
	amtCents := rec.PaymentAmountCents
	if amtCents <= 0 {
		// derive from lines when not set
		for _, ln := range rec.Lines {
			amtCents += ln.CreditCents + ln.DebitCents
		}
	}
	if amtCents <= 0 {
		omit("payment_amount", "PaymentAmount ausente")
		return Payment{}, Customer{}, oms
	}
	methods := []SAFPaymentMethod{{
		PaymentMechanism: mech, // may be empty — XSD optional
		PaymentAmount:    MustDecimal(formatCents(amtCents)),
		PaymentDate:      payDate,
	}}

	gross := rec.GrossTotalCents
	net := rec.NetTotalCents
	tax := rec.TaxPayableCents
	if gross <= 0 {
		gross = amtCents
	}
	if net < 0 || tax < 0 {
		omit("document_totals", "totais inválidos")
		return Payment{}, Customer{}, oms
	}
	if net == 0 && tax == 0 {
		net = gross
	}

	pay := Payment{
		PaymentRefNo:    ref,
		TransactionDate: txDate,
		PaymentType:     rec.PaymentType,
		DocumentStatus: PaymentDocStatus{
			PaymentStatus:     rec.PaymentStatus,
			PaymentStatusDate: statusDT,
			SourceID:          sourceID,
			SourcePayment:     rec.SourcePayment,
		},
		PaymentMethod:   methods,
		SourceID:        sourceID,
		SystemEntryDate: entryDT,
		CustomerID:      customerID,
		Line:            lines,
		DocumentTotals: PaymentDocTotals{
			TaxPayable: MustMoney2(formatCents(tax)),
			NetTotal:   MustMoney2(formatCents(net)),
			GrossTotal: MustMoney2(formatCents(gross)),
		},
	}
	cust := Customer{
		CustomerID:           customerID,
		AccountID:            "Desconhecido",
		CustomerTaxID:        "Desconhecido",
		CompanyName:          "Desconhecido",
		BillingAddress:       &AddressStructure{AddressDetail: "Desconhecido", City: "Desconhecido", Country: "AO"},
		SelfBillingIndicator: 0,
	}
	return pay, cust, nil
}
