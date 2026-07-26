package saftao

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// TopLevelElements are the XSD global elements required for the structural foundation.
var requiredGlobalElements = []string{
	"AuditFile", "Header", "MasterFiles", "GeneralLedgerAccounts",
	"Customer", "Supplier", "Product", "TaxTable",
	"GeneralLedgerEntries", "SourceDocuments",
}

// Inventory lists global xs:element names found in the embedded XSD.
func Inventory() ([]string, error) {
	raw, err := XSDBytes()
	if err != nil {
		return nil, err
	}
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	var out []string
	seen := map[string]struct{}{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("saftao: parse XSD: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "element" {
			continue
		}
		// Only global elements: direct children of schema (depth approx via no nested tracking —
		// accept name= attributes on xs:element; filter refs later).
		var name, ref string
		for _, a := range se.Attr {
			switch a.Name.Local {
			case "name":
				name = a.Value
			case "ref":
				ref = a.Value
			}
		}
		if name == "" || ref != "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// EnsureRequiredStructure fails if the embedded XSD lacks foundation elements.
func EnsureRequiredStructure() error {
	inv, err := Inventory()
	if err != nil {
		return err
	}
	have := map[string]struct{}{}
	for _, n := range inv {
		have[n] = struct{}{}
	}
	for _, need := range requiredGlobalElements {
		if _, ok := have[need]; !ok {
			return fmt.Errorf("saftao: XSD sem elemento global %q", need)
		}
	}
	meta := Meta()
	if meta.Certified || meta.Status != "pending_validation" {
		return fmt.Errorf("saftao: meta de conformidade incorrecta")
	}
	return nil
}
