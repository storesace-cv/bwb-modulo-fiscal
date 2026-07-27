package saftao

import "encoding/xml"

// AuditFile is a structural skeleton aligned with XSD element AuditFile (SAFTAO1.01_01).
// Field set grows with typed slices; still ≠ conformidade AGT / export de produção.
type AuditFile struct {
	XMLName xml.Name `xml:"urn:OECD:StandardAuditFile-Tax:AO_1.01_01 AuditFile"`

	Header               *Header               `xml:"Header"`
	MasterFiles          *MasterFiles          `xml:"MasterFiles"`
	GeneralLedgerEntries *GeneralLedgerEntries `xml:"GeneralLedgerEntries,omitempty"`
	SourceDocuments      *SourceDocuments      `xml:"SourceDocuments,omitempty"`
}

// Header mirrors required XSD Header children (CompanyAddress typed as AddressStructureAO shape).
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

// AddressStructure matches AddressStructure / AddressStructureAO child sequence (Country "AO" for Header).
type AddressStructure struct {
	AddressDetail string `xml:"AddressDetail"`
	City          string `xml:"City"`
	PostalCode    string `xml:"PostalCode,omitempty"`
	Country       string `xml:"Country"`
}

// MasterFiles is the XSD MasterFiles container.
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

// Customer is MasterFiles/Customer (fields required by XSD sequence for keyref fixtures).
type Customer struct {
	CustomerID           string            `xml:"CustomerID"`
	AccountID            string            `xml:"AccountID"`
	CustomerTaxID        string            `xml:"CustomerTaxID"`
	CompanyName          string            `xml:"CompanyName"`
	BillingAddress       *AddressStructure `xml:"BillingAddress"`
	SelfBillingIndicator int               `xml:"SelfBillingIndicator"`
}

// Supplier stub.
type Supplier struct {
	SupplierID string `xml:"SupplierID"`
}

// Product is MasterFiles/Product (XSD order: ProductType before ProductCode).
type Product struct {
	ProductType        string `xml:"ProductType"`
	ProductCode        string `xml:"ProductCode"`
	ProductDescription string `xml:"ProductDescription"`
	ProductNumberCode  string `xml:"ProductNumberCode"`
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

// SourceDocuments holds the five L3 document tables (DEC-PROD-001); exposure follows enrolment.
type SourceDocuments struct {
	SalesInvoices    *SalesInvoices       `xml:"SalesInvoices,omitempty"`
	MovementOfGoods  *MovementOfGoods     `xml:"MovementOfGoods,omitempty"`
	WorkingDocuments *WorkingDocuments    `xml:"WorkingDocuments,omitempty"`
	Payments         *Payments            `xml:"Payments,omitempty"`
	PurchaseInvoices *DocumentTableTotals `xml:"PurchaseInvoices,omitempty"`
}

// DocumentTableTotals is the NumberOfEntries/TotalDebit/TotalCredit prefix shared by several tables.
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
		SalesInvoices: &SalesInvoices{
			NumberOfEntries: "0",
			TotalDebit:      MustDecimal("0.00"),
			TotalCredit:     MustDecimal("0.00"),
		},
	}
	return doc
}

// SchemaVersion returns the XSD version string (not a compliance claim).
func SchemaVersion() string {
	return Meta().SchemaVersion
}
