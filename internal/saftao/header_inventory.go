package saftao

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// RequiredHeaderChildren are the XSD Header sequence children (minOccurs default 1
// unless noted). source_id AO-SAFT-XSD-1.01_01 — structural only; ≠ AO-* confirmed.
var RequiredHeaderChildren = []string{
	"AuditFileVersion",
	"CompanyID",
	"TaxRegistrationNumber",
	"TaxAccountingBasis",
	"CompanyName",
	// BusinessName optional
	"CompanyAddress",
	"FiscalYear",
	"StartDate",
	"EndDate",
	"CurrencyCode",
	"DateCreated",
	"TaxEntity",
	"ProductCompanyTaxID",
	"SoftwareValidationNumber",
	"ProductID",
	"ProductVersion",
}

// HeaderFieldInventory extracts ordered child element names/refs under xs:element name=Header.
func HeaderFieldInventory() ([]string, error) {
	raw, err := XSDBytes()
	if err != nil {
		return nil, err
	}
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	inHeader := false
	depth := 0
	var out []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("saftao: header inventory: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if !inHeader && t.Name.Local == "element" {
				for _, a := range t.Attr {
					if a.Name.Local == "name" && a.Value == "Header" {
						inHeader = true
						depth = 1
						break
					}
				}
				continue
			}
			if !inHeader {
				continue
			}
			depth++
			if t.Name.Local == "element" {
				var name, ref string
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "name":
						name = a.Value
					case "ref":
						ref = a.Value
					}
				}
				if name != "" {
					out = append(out, name)
				} else if ref != "" {
					out = append(out, ref)
				}
			}
		case xml.EndElement:
			if inHeader {
				depth--
				if depth == 0 {
					return out, nil
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("saftao: Header não encontrado no XSD")
	}
	return out, nil
}

// EnsureHeaderShape checks required Header children exist in the embedded XSD order inventory.
func EnsureHeaderShape() error {
	inv, err := HeaderFieldInventory()
	if err != nil {
		return err
	}
	have := map[string]struct{}{}
	for _, n := range inv {
		have[n] = struct{}{}
	}
	for _, need := range RequiredHeaderChildren {
		if _, ok := have[need]; !ok {
			return fmt.Errorf("saftao: Header XSD sem %q", need)
		}
	}
	return nil
}
