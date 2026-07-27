package saftao

// DocumentGroup is a SAF-T (AO) L3 SourceDocuments table (DEC-PROD-001).
// Product domain keeps all five groups configurable; exposure to SAF-T/FE
// channels follows client enrolment (DEC-PROD-002/004) — not invented here.
type DocumentGroup string

const (
	GroupSalesInvoices    DocumentGroup = "SalesInvoices"
	GroupMovementOfGoods  DocumentGroup = "MovementOfGoods"
	GroupWorkingDocuments DocumentGroup = "WorkingDocuments"
	GroupPayments         DocumentGroup = "Payments"
	GroupPurchaseInvoices DocumentGroup = "PurchaseInvoices"
)

// AllDocumentGroups is the closed set of L3 groups from DEC-PROD-001 / XSD SourceDocuments.
func AllDocumentGroups() []DocumentGroup {
	return []DocumentGroup{
		GroupSalesInvoices,
		GroupMovementOfGoods,
		GroupWorkingDocuments,
		GroupPayments,
		GroupPurchaseInvoices,
	}
}

// ValidDocumentGroup reports whether g is one of the five XSD L3 tables.
func ValidDocumentGroup(g DocumentGroup) bool {
	switch g {
	case GroupSalesInvoices, GroupMovementOfGoods, GroupWorkingDocuments, GroupPayments, GroupPurchaseInvoices:
		return true
	default:
		return false
	}
}

// PendingRegulatory notes structural fields whose business/algorithm rules are not AO-*-confirmed.
type PendingRegulatory string

const (
	// PendingHashAlgorithm: Hash/HashControl present in XSD; signing algorithm ≠ confirmed AO-* (C-SIGN-001).
	PendingHashAlgorithm PendingRegulatory = "hash_algorithm_pending_ao"
	// PendingInvoiceTypeSemantics: enum values from XSD; product activation gated by DEC-REG-003 / doctype.
	PendingInvoiceTypeSemantics PendingRegulatory = "invoice_type_semantics_pending_ao"
	// PendingMovementTypeSemantics: MovementType enum from XSD; activation gated by enrolment/DEC-PROD.
	PendingMovementTypeSemantics PendingRegulatory = "movement_type_semantics_pending_ao"
	// PendingWorkTypeSemantics: WorkType enum from XSD; activation gated by enrolment/DEC-PROD.
	PendingWorkTypeSemantics PendingRegulatory = "work_type_semantics_pending_ao"
	// PendingPaymentTypeSemantics: PaymentType enum from XSD; activation gated by enrolment/DEC-PROD.
	PendingPaymentTypeSemantics PendingRegulatory = "payment_type_semantics_pending_ao"
	// PendingPurchaseTypeSemantics: PurchaseType enum from XSD; activation gated by enrolment/DEC-PROD.
	PendingPurchaseTypeSemantics PendingRegulatory = "purchase_type_semantics_pending_ao"
	// PendingTaxTableSemantics: TaxType/TaxCode enums from XSD; rates/activation ≠ AO-* confirmed.
	PendingTaxTableSemantics PendingRegulatory = "tax_table_semantics_pending_ao"
)
