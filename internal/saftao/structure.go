package saftao

import "encoding/xml"

// AuditFile is a structural skeleton aligned with XSD element AuditFile (SAFTAO1.01_01).
// Field set is intentionally minimal — not a full mapping; not export-ready.
type AuditFile struct {
	XMLName xml.Name `xml:"urn:OECD:StandardAuditFile-Tax:AO_1.01_01 AuditFile"`

	Header               *Header               `xml:"Header"`
	MasterFiles          *MasterFiles          `xml:"MasterFiles"`
	GeneralLedgerEntries *GeneralLedgerEntries `xml:"GeneralLedgerEntries,omitempty"`
	SourceDocuments      *SourceDocuments      `xml:"SourceDocuments,omitempty"`
}

// Header mirrors the XSD Header element presence (fields filled by later slices).
type Header struct {
	AuditFileVersion         string `xml:"AuditFileVersion"`
	CompanyID                string `xml:"CompanyID"`
	TaxRegistrationNumber    string `xml:"TaxRegistrationNumber"`
	TaxAccountingBasis       string `xml:"TaxAccountingBasis"`
	CompanyName              string `xml:"CompanyName"`
	FiscalYear               string `xml:"FiscalYear"`
	StartDate                string `xml:"StartDate"`
	EndDate                  string `xml:"EndDate"`
	CurrencyCode             string `xml:"CurrencyCode"`
	DateCreated              string `xml:"DateCreated"`
	TaxEntity                string `xml:"TaxEntity"`
	ProductCompanyTaxID      string `xml:"ProductCompanyTaxID"`
	SoftwareValidationNumber string `xml:"SoftwareValidationNumber"`
	ProductID                string `xml:"ProductID"`
	ProductVersion           string `xml:"ProductVersion"`
}

// MasterFiles is the XSD MasterFiles container.
type MasterFiles struct {
	GeneralLedgerAccounts []string  `xml:"GeneralLedgerAccounts,omitempty"` // placeholder; typed later
	Customer              []string  `xml:"Customer,omitempty"`
	Supplier              []string  `xml:"Supplier,omitempty"`
	Product               []string  `xml:"Product,omitempty"`
	TaxTable              *struct{} `xml:"TaxTable,omitempty"`
}

// GeneralLedgerEntries placeholder container.
type GeneralLedgerEntries struct {
	NumberOfEntries string `xml:"NumberOfEntries"`
	TotalDebit      string `xml:"TotalDebit"`
	TotalCredit     string `xml:"TotalCredit"`
}

// SourceDocuments placeholder; SalesInvoices etc. come in later slices.
type SourceDocuments struct {
	SalesInvoices *struct {
		NumberOfEntries string `xml:"NumberOfEntries"`
		TotalDebit      string `xml:"TotalDebit"`
		TotalCredit     string `xml:"TotalCredit"`
	} `xml:"SalesInvoices,omitempty"`
}

// NewEmptyAuditFile returns a non-certified structural envelope.
func NewEmptyAuditFile() AuditFile {
	return AuditFile{
		Header: &Header{
			AuditFileVersion: SchemaVersion(),
			CurrencyCode:     "AOA",
		},
		MasterFiles: &MasterFiles{},
	}
}

// SchemaVersion returns the XSD version string (not a compliance claim).
func SchemaVersion() string {
	return Meta().SchemaVersion
}
