package taxao

import (
	"math"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

// LineInput is one line for tax calculation (post money/quantity parse).
type LineInput struct {
	Quantity  quantity.Qty
	UnitPrice money.Amount
	TaxCode   string
}

// LineResult holds computed amounts in cents (AOA scale 2).
type LineResult struct {
	TaxCode    string
	TaxType    TaxType
	RateBP     int64
	NetCents   int64
	TaxCents   int64
	GrossCents int64
}

// DocumentTotals aggregates line results.
type DocumentTotals struct {
	Lines           []LineResult
	NetTotalCents   int64
	TaxTotalCents   int64
	GrossTotalCents int64
}

// LineNetCents computes line net = quantity × unit_price with half-up rounding to cents.
func LineNetCents(q quantity.Qty, unit money.Amount) (int64, error) {
	if q.Scaled() <= 0 || unit.Cents() < 0 {
		return 0, ErrInvalidInput
	}
	// net = round(qty_scaled * unit_cents / quantity.Factor)
	num := q.Scaled() * unit.Cents()
	if q.Scaled() > 0 && unit.Cents() > math.MaxInt64/q.Scaled() {
		return 0, ErrOverflow
	}
	return divHalfUp(num, quantity.Factor), nil
}

// LineTaxCents computes tax on net using rate basis points (half-up).
func LineTaxCents(netCents, rateBP int64) (int64, error) {
	if netCents < 0 || rateBP < 0 || rateBP > 10000 {
		return 0, ErrInvalidInput
	}
	if rateBP == 0 {
		return 0, nil
	}
	num := netCents * rateBP
	if netCents > 0 && rateBP > math.MaxInt64/netCents {
		return 0, ErrOverflow
	}
	return divHalfUp(num, 10000), nil
}

func divHalfUp(num, den int64) int64 {
	if den <= 0 {
		return 0
	}
	if num >= 0 {
		return (num + den/2) / den
	}
	return (num - den/2) / den
}

// CalculateLine returns net/tax/gross for one line.
func CalculateLine(in LineInput) (LineResult, error) {
	e, err := Lookup(in.TaxCode)
	if err != nil {
		return LineResult{}, err
	}
	net, err := LineNetCents(in.Quantity, in.UnitPrice)
	if err != nil {
		return LineResult{}, err
	}
	tax, err := LineTaxCents(net, e.rateBP)
	if err != nil {
		return LineResult{}, err
	}
	gross, err := addCents(net, tax)
	if err != nil {
		return LineResult{}, err
	}
	return LineResult{
		TaxCode: in.TaxCode, TaxType: e.typ, RateBP: e.rateBP,
		NetCents: net, TaxCents: tax, GrossCents: gross,
	}, nil
}

// CalculateDocument sums all lines; fails on first unknown tax code.
func CalculateDocument(lines []LineInput) (DocumentTotals, error) {
	if len(lines) == 0 {
		return DocumentTotals{}, ErrInvalidInput
	}
	out := DocumentTotals{Lines: make([]LineResult, 0, len(lines))}
	for _, ln := range lines {
		lr, err := CalculateLine(ln)
		if err != nil {
			return DocumentTotals{}, err
		}
		out.Lines = append(out.Lines, lr)
		var err2 error
		out.NetTotalCents, err2 = addCents(out.NetTotalCents, lr.NetCents)
		if err2 != nil {
			return DocumentTotals{}, err2
		}
		out.TaxTotalCents, err2 = addCents(out.TaxTotalCents, lr.TaxCents)
		if err2 != nil {
			return DocumentTotals{}, err2
		}
		out.GrossTotalCents, err2 = addCents(out.GrossTotalCents, lr.GrossCents)
		if err2 != nil {
			return DocumentTotals{}, err2
		}
	}
	return out, nil
}

func addCents(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, ErrOverflow
	}
	return a + b, nil
}

// FormatCents returns OpenAPI Money canonical string for totals helpers/tests.
func FormatCents(cents int64) string {
	a, err := money.FromCents(cents)
	if err != nil {
		return "0.00"
	}
	return a.FormatCanonical()
}
