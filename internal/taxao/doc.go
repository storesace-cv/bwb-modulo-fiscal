// Package taxao implements Angola line/document tax calculation for the fiscal
// engine (RM-ENG-003). Integer-only money; no float.
//
// Rates and codes are provisional (AO-TAX-001 partial; DE 683/25 Citação G
// @19171–19173, Tabelas 2–6 @19212–19227 pending_validation). ≠ confirmed_normative.
// IS/IEC/retenciones/discounts beyond MVP catalog are fail-closed until sourced.
package taxao
