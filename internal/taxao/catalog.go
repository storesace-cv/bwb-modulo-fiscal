package taxao

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownTaxCode = errors.New("taxao: unknown tax code")
	ErrInvalidInput   = errors.New("taxao: invalid input")
	ErrOverflow       = errors.New("taxao: overflow")
)

// TaxType identifies the tax family (IVA MVP slice).
type TaxType string

const (
	TaxTypeIVA TaxType = "IVA"
)

// entry describes a provisional IVA tax code (basis points: 1400 = 14.00%).
type entry struct {
	typ        TaxType
	rateBP     int64 // 0..10000
	exempt     bool
	sourceNote string
}

// mvpCatalog aligns with slice OpenAPI examples and SAFT synthetic TaxTable (≠ AO-* confirmed).
var mvpCatalog = map[string]entry{
	"NOR": {typ: TaxTypeIVA, rateBP: 1400, sourceNote: "IVA taxa normal sintética 14% (provisional)"},
	"RED": {typ: TaxTypeIVA, rateBP: 500, sourceNote: "IVA taxa reduzida 5% (provisional MVP)"},
	"INT": {typ: TaxTypeIVA, rateBP: 700, sourceNote: "IVA taxa intermédia 7% (provisional MVP)"},
	"ISE": {typ: TaxTypeIVA, rateBP: 0, exempt: true, sourceNote: "IVA isento (provisional MVP)"},
}

// KnownTaxCode reports whether code is in the MVP catalog (case-sensitive NOR/ISE/…).
func KnownTaxCode(code string) bool {
	_, ok := mvpCatalog[strings.TrimSpace(code)]
	return ok
}

// Lookup returns catalog metadata for a tax code.
func Lookup(code string) (entry, error) {
	e, ok := mvpCatalog[strings.TrimSpace(code)]
	if !ok {
		return entry{}, fmt.Errorf("%w: %q", ErrUnknownTaxCode, code)
	}
	return e, nil
}
