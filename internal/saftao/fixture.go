package saftao

// MinimalSalesInvoiceFixture returns a synthetic AuditFile with one FT invoice
// shaped for local XSD validation. Values are placeholders — not fiscal truth,
// not AGT-accepted, and Hash algorithm remains PendingHashAlgorithm.
func MinimalSalesInvoiceFixture() AuditFile {
	credit := MustMoney2("100.00")
	taxPct := "14.00"
	addr := &AddressStructure{
		AddressDetail: "Rua Sintetica 1",
		City:          "Luanda",
		Country:       "AO",
	}
	inv := Invoice{
		InvoiceNo: "FT S001/1",
		DocumentStatus: DocumentStatus{
			InvoiceStatus:     InvoiceStatusN,
			InvoiceStatusDate: MustDateTime("2026-01-15T10:00:00"),
			SourceID:          "POS1",
			SourceBilling:     SourceBillingP,
		},
		Hash:        "SYNTHETIC-HASH-NOT-A-SIGNATURE",
		HashControl: "0",
		InvoiceDate: MustDate("2026-01-15"),
		InvoiceType: InvoiceTypeFT,
		SpecialRegimes: SpecialRegimes{
			SelfBillingIndicator:         0,
			CashVATSchemeIndicator:       0,
			ThirdPartiesBillingIndicator: 0,
		},
		SourceID:        "POS1",
		SystemEntryDate: MustDateTime("2026-01-15T10:00:00"),
		CustomerID:      "C1",
		Line: []InvoiceLine{{
			LineNumber:         "1",
			ProductCode:        "P1",
			ProductDescription: "Servico sintetico",
			Quantity:           MustDecimal("1"),
			UnitOfMeasure:      "UN",
			UnitPrice:          MustDecimal("100.00"),
			TaxPointDate:       MustDate("2026-01-15"),
			Description:        "Linha sintetica",
			CreditAmount:       &credit,
			Tax: Tax{
				TaxType:       "IVA",
				TaxCode:       "NOR",
				TaxPercentage: &taxPct,
			},
		}},
		DocumentTotals: DocumentTotals{
			TaxPayable: MustMoney2("14.00"),
			NetTotal:   MustMoney2("100.00"),
			GrossTotal: MustMoney2("114.00"),
		},
	}
	return AuditFile{
		Header: &Header{
			AuditFileVersion:         SchemaVersion(),
			CompanyID:                "5000000000",
			TaxRegistrationNumber:    "5000000000",
			TaxAccountingBasis:       "F",
			CompanyName:              "Empresa Sintetica Lda",
			CompanyAddress:           addr,
			FiscalYear:               "2026",
			StartDate:                "2026-01-01",
			EndDate:                  "2026-12-31",
			CurrencyCode:             "AOA",
			DateCreated:              "2026-01-31",
			TaxEntity:                "Global",
			ProductCompanyTaxID:      "5417000000",
			SoftwareValidationNumber: "0",
			ProductID:                "BWBFiscal/BWB",
			ProductVersion:           "0.0.0-test",
		},
		MasterFiles: &MasterFiles{
			Customer: []Customer{{
				CustomerID:           "C1",
				AccountID:            "Desconhecido",
				CustomerTaxID:        "999999999",
				CompanyName:          "Cliente Sintetico",
				BillingAddress:       addr,
				SelfBillingIndicator: 0,
			}},
			Product: []Product{{
				ProductType:        "S",
				ProductCode:        "P1",
				ProductDescription: "Servico sintetico",
				ProductNumberCode:  "P1",
			}},
		},
		SourceDocuments: &SourceDocuments{
			SalesInvoices: &SalesInvoices{
				NumberOfEntries: "1",
				TotalDebit:      MustDecimal("0.00"),
				TotalCredit:     MustDecimal("100.00"),
				Invoice:         []Invoice{inv},
			},
		},
	}
}

// MinimalMovementOfGoodsFixture returns a synthetic AuditFile with one GR stock movement
// for local XSD validation. Placeholders only — ≠ AGT / AO-* / Hash algorithm.
func MinimalMovementOfGoodsFixture() AuditFile {
	credit := MustMoney2("50.00")
	addr := &AddressStructure{
		AddressDetail: "Rua Sintetica 1",
		City:          "Luanda",
		Country:       "AO",
	}
	sm := StockMovement{
		DocumentNumber: "GR S001/1",
		DocumentStatus: MovementDocumentStatus{
			MovementStatus:     MovementStatusN,
			MovementStatusDate: MustDateTime("2026-01-15T10:00:00"),
			SourceID:           "POS1",
			SourceBilling:      SourceBillingP,
		},
		Hash:              "SYNTHETIC-HASH-NOT-A-SIGNATURE",
		HashControl:       "0",
		MovementDate:      MustDate("2026-01-15"),
		MovementType:      MovementTypeGR,
		SystemEntryDate:   MustDateTime("2026-01-15T10:00:00"),
		CustomerID:        "C1",
		SourceID:          "POS1",
		MovementStartTime: MustDateTime("2026-01-15T09:00:00"),
		Line: []StockMovementLine{{
			LineNumber:         "1",
			ProductCode:        "P1",
			ProductDescription: "Mercadoria sintetica",
			Quantity:           MustDecimal("1"),
			UnitOfMeasure:      "UN",
			UnitPrice:          MustDecimal("50.00"),
			Description:        "Linha movimentacao",
			CreditAmount:       &credit,
			Tax: &MovementTax{
				TaxType:       "IVA",
				TaxCode:       "NOR",
				TaxPercentage: "14.00",
			},
		}},
		DocumentTotals: DocumentTotals{
			TaxPayable: MustMoney2("7.00"),
			NetTotal:   MustMoney2("50.00"),
			GrossTotal: MustMoney2("57.00"),
		},
	}
	return AuditFile{
		Header: &Header{
			AuditFileVersion:         SchemaVersion(),
			CompanyID:                "5000000000",
			TaxRegistrationNumber:    "5000000000",
			TaxAccountingBasis:       "F",
			CompanyName:              "Empresa Sintetica Lda",
			CompanyAddress:           addr,
			FiscalYear:               "2026",
			StartDate:                "2026-01-01",
			EndDate:                  "2026-12-31",
			CurrencyCode:             "AOA",
			DateCreated:              "2026-01-31",
			TaxEntity:                "Global",
			ProductCompanyTaxID:      "5417000000",
			SoftwareValidationNumber: "0",
			ProductID:                "BWBFiscal/BWB",
			ProductVersion:           "0.0.0-test",
		},
		MasterFiles: &MasterFiles{
			Customer: []Customer{{
				CustomerID:           "C1",
				AccountID:            "Desconhecido",
				CustomerTaxID:        "999999999",
				CompanyName:          "Cliente Sintetico",
				BillingAddress:       addr,
				SelfBillingIndicator: 0,
			}},
			Product: []Product{{
				ProductType:        "P",
				ProductCode:        "P1",
				ProductDescription: "Mercadoria sintetica",
				ProductNumberCode:  "P1",
			}},
		},
		SourceDocuments: &SourceDocuments{
			MovementOfGoods: &MovementOfGoods{
				NumberOfMovementLines: "1",
				TotalQuantityIssued:   MustDecimal("1"),
				StockMovement:         []StockMovement{sm},
			},
		},
	}
}

// MinimalWorkingDocumentsFixture returns a synthetic AuditFile with one PF work document
// for local XSD validation. Placeholders only — ≠ AGT / AO-* / Hash algorithm.
func MinimalWorkingDocumentsFixture() AuditFile {
	credit := MustMoney2("25.00")
	taxPct := "14.00"
	addr := &AddressStructure{
		AddressDetail: "Rua Sintetica 1",
		City:          "Luanda",
		Country:       "AO",
	}
	wd := WorkDocument{
		DocumentNumber: "PF S001/1",
		DocumentStatus: WorkDocumentStatus{
			WorkStatus:     WorkStatusN,
			WorkStatusDate: MustDateTime("2026-01-15T10:00:00"),
			SourceID:       "POS1",
			SourceBilling:  SourceBillingP,
		},
		Hash:            "SYNTHETIC-HASH-NOT-A-SIGNATURE",
		HashControl:     "0",
		WorkDate:        MustDate("2026-01-15"),
		WorkType:        WorkTypePF,
		SourceID:        "POS1",
		SystemEntryDate: MustDateTime("2026-01-15T10:00:00"),
		CustomerID:      "C1",
		Line: []WorkDocumentLine{{
			LineNumber:         "1",
			ProductCode:        "P1",
			ProductDescription: "Servico conferencia",
			Quantity:           MustDecimal("1"),
			UnitOfMeasure:      "UN",
			UnitPrice:          MustDecimal("25.00"),
			TaxPointDate:       MustDate("2026-01-15"),
			Description:        "Linha proforma",
			CreditAmount:       &credit,
			Tax: &Tax{
				TaxType:       "IVA",
				TaxCode:       "NOR",
				TaxPercentage: &taxPct,
			},
		}},
		DocumentTotals: DocumentTotals{
			TaxPayable: MustMoney2("3.50"),
			NetTotal:   MustMoney2("25.00"),
			GrossTotal: MustMoney2("28.50"),
		},
	}
	return AuditFile{
		Header: &Header{
			AuditFileVersion:         SchemaVersion(),
			CompanyID:                "5000000000",
			TaxRegistrationNumber:    "5000000000",
			TaxAccountingBasis:       "F",
			CompanyName:              "Empresa Sintetica Lda",
			CompanyAddress:           addr,
			FiscalYear:               "2026",
			StartDate:                "2026-01-01",
			EndDate:                  "2026-12-31",
			CurrencyCode:             "AOA",
			DateCreated:              "2026-01-31",
			TaxEntity:                "Global",
			ProductCompanyTaxID:      "5417000000",
			SoftwareValidationNumber: "0",
			ProductID:                "BWBFiscal/BWB",
			ProductVersion:           "0.0.0-test",
		},
		MasterFiles: &MasterFiles{
			Customer: []Customer{{
				CustomerID:           "C1",
				AccountID:            "Desconhecido",
				CustomerTaxID:        "999999999",
				CompanyName:          "Cliente Sintetico",
				BillingAddress:       addr,
				SelfBillingIndicator: 0,
			}},
			Product: []Product{{
				ProductType:        "S",
				ProductCode:        "P1",
				ProductDescription: "Servico conferencia",
				ProductNumberCode:  "P1",
			}},
		},
		SourceDocuments: &SourceDocuments{
			WorkingDocuments: &WorkingDocuments{
				NumberOfEntries: "1",
				TotalDebit:      MustDecimal("0.00"),
				TotalCredit:     MustDecimal("25.00"),
				WorkDocument:    []WorkDocument{wd},
			},
		},
	}
}
