package saftao

import "encoding/xml"

// AuditFile is a structural skeleton aligned with XSD element AuditFile (SAFTAO1.01_01).
// Field set is intentionally incomplete — not export-ready; ≠ conformidade AGT.
type AuditFile struct {
	XMLName xml.Name `xml:"urn:OECD:StandardAuditFile-Tax:AO_1.01_01 AuditFile"`

	Header               *Header               `xml:"Header"`
	MasterFiles          *MasterFiles          `xml:"MasterFiles"`
	GeneralLedgerEntries *GeneralLedgerEntries `xml:"GeneralLedgerEntries,omitempty"`
	SourceDocuments      *SourceDocuments      `xml:"SourceDocuments,omitempty"`
}

// Header mirrors required XSD Header children (CompanyAddress typed as stub).
type Header struct {
	AuditFileVersion         string            `xml:"AuditFileVersion"`
	CompanyID                string            `xml:"CompanyID"`
	TaxRegistrationNumber    string            `xml:"TaxRegistrationNumber"`
	TaxAccountingBasis       string            `xml:"TaxAccountingBasis"`
	CompanyName              string            `xml:"CompanyName"`
	BusinessName             string            `xml:"BusinessName,omitempty"`
	CompanyAddress           *AddressStructure `xml:"CompanyAddress"`
	FiscalYear               string            `xml:"FiscalYear"`
	StartDate                string            `xml:"StartDate"`
	EndDate                  string            `xml:"EndDate"`
	CurrencyCode             string            `xml:"CurrencyCode"`
	DateCreated              string            `xml:"DateCreated"`
	TaxEntity                string            `xml:"TaxEntity"`
	ProductCompanyTaxID      string            `xml:"ProductCompanyTaxID"`
	SoftwareValidationNumber string            `xml:"SoftwareValidationNumber"`
	ProductID                string            `xml:"ProductID"`
	ProductVersion           string            `xml:"ProductVersion"`
}

// AddressStructure is a minimal address stub (full XSD mapping later).
type AddressStructure struct {
	AddressDetail string `xml:"AddressDetail"`
	City          string `xml:"City"`
	PostalCode    string `xml:"PostalCode,omitempty"`
	Country       string `xml:"Country"`
}

// MasterFiles is the XSD MasterFiles container (typed stubs).
type MasterFiles struct {
	GeneralLedgerAccounts []GeneralLedgerAccounts `xml:"GeneralLedgerAccounts,omitempty"`
	Customer              []Customer              `xml:"Customer,omitempty"`
	Supplier              []Supplier              `xml:"Supplier,omitempty"`
	Product               []Product               `xml:"Product,omitempty"`
	TaxTable              *TaxTable               `xml:"TaxTable,omitempty"`
}

// GeneralLedgerAccounts stub.
type GeneralLedgerAccounts struct {
	NumberOfEntries string `xml:"NumberOfEntries,omitempty"`
}

// Customer stub (IDs only — no personal payload helpers here).
type Customer struct {
	CustomerID string `xml:"CustomerID"`
}

// Supplier stub.
type Supplier struct {
	SupplierID string `xml:"SupplierID"`
}

// Product stub.
type Product struct {
	ProductCode        string `xml:"ProductCode"`
	ProductType        string `xml:"ProductType"`
	ProductDescription string `xml:"ProductDescription"`
}

// TaxTable stub.
type TaxTable struct {
	TaxTableEntry []TaxTableEntry `xml:"TaxTableEntry,omitempty"`
}

// TaxTableEntry stub.
type TaxTableEntry struct {
	TaxType string `xml:"TaxType"`
	TaxCode string `xml:"TaxCode"`
}

// GeneralLedgerEntries placeholder container.
type GeneralLedgerEntries struct {
	NumberOfEntries string `xml:"NumberOfEntries"`
	TotalDebit      string `xml:"TotalDebit"`
	TotalCredit     string `xml:"TotalCredit"`
}

// SourceDocuments holds optional document tables from the XSD.
type SourceDocuments struct {
	SalesInvoices    *DocumentTableTotals `xml:"SalesInvoices,omitempty"`
	WorkingDocuments *DocumentTableTotals `xml:"WorkingDocuments,omitempty"`
	Payments         *DocumentTableTotals `xml:"Payments,omitempty"`
	PurchaseInvoices *DocumentTableTotals `xml:"PurchaseInvoices,omitempty"`
}

// DocumentTableTotals is the NumberOfEntries/TotalDebit/TotalCredit prefix shared by tables.
type DocumentTableTotals struct {
	NumberOfEntries string `xml:"NumberOfEntries"`
	TotalDebit      string `xml:"TotalDebit"`
	TotalCredit     string `xml:"TotalCredit"`
}

// NewEmptyAuditFile returns a non-certified structural envelope (Header + empty MasterFiles).
func NewEmptyAuditFile() AuditFile {
	return AuditFile{
		Header: &Header{
			AuditFileVersion: SchemaVersion(),
			CurrencyCode:     "AOA",
			CompanyAddress:   &AddressStructure{Country: "AO"},
		},
		MasterFiles: &MasterFiles{},
	}
}

// NewSalesSkeleton returns Header + empty SalesInvoices totals (still ≠ export válido).
func NewSalesSkeleton() AuditFile {
	doc := NewEmptyAuditFile()
	doc.SourceDocuments = &SourceDocuments{
		SalesInvoices: &DocumentTableTotals{
			NumberOfEntries: "0",
			TotalDebit:      "0.00",
			TotalCredit:     "0.00",
		},
	}
	return doc
}

// SchemaVersion returns the XSD version string (not a compliance claim).
func SchemaVersion() string {
	return Meta().SchemaVersion
}
