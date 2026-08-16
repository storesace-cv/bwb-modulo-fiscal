package agttestkit

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// OpenWorkbookProvider loads a validated AGT test workbook into memory custody.
// path must be explicit (no default). The workbook is closed before return.
// Each row keeps taxpayerNIF + sourceLabel (NOME origin designation) bound to the
// key pair in private memory only — never listed, logged, or persisted.
// Signer returns an opaque crypto.Signer proxy (never *rsa.PrivateKey).
func OpenWorkbookProvider(path string) (IdentityProvider, error) {
	held, err := loadHeldFromWorkbook(path)
	if err != nil {
		return nil, err
	}
	p, err := newMemoryProvider(held)
	if err != nil {
		wipeHeld(held)
		return nil, err
	}
	return p, nil
}

func loadHeldFromWorkbook(path string) ([]heldIdentity, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrPathRequired
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("%w: path inaccessible", ErrOpenWorkbook)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrOpenWorkbook)
	}
	defer func() { _ = f.Close() }()

	_, rows, err := extractRows(f)
	if err != nil {
		return nil, err
	}
	defer zeroRows(rows)

	held, err := heldFromRows(rows, RoleTaxpayerTest)
	zeroRows(rows)
	if err != nil {
		wipeHeld(held)
		return nil, err
	}
	return held, nil
}

func heldFromRows(rows []rawRow, role string) ([]heldIdentity, error) {
	seenNIF := map[string]string{}
	seenNome := map[string]string{}
	seenRef := map[string]struct{}{}
	var held []heldIdentity

	for _, row := range rows {
		priv, err := parseRSAPrivate(row.privPEM)
		if err != nil {
			wipeHeld(held)
			return nil, fmt.Errorf("%w: excel_row=%d", ErrInvalidPrivatePEM, row.excelRow)
		}
		pub, err := parseRSAPublic(row.pubPEM)
		if err != nil {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w: excel_row=%d", ErrInvalidPublicPEM, row.excelRow)
		}
		if !rsaPublicEqual(priv.PublicKey, pub) {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w: excel_row=%d", ErrKeyPairMismatch, row.excelRow)
		}
		bits := priv.N.BitLen()
		if bits < MinRSABits {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w: excel_row=%d bits=%d min=%d", ErrRSATooSmall, row.excelRow, bits, MinRSABits)
		}
		ref := opaqueRefFromPublic(pub)
		if _, ok := seenRef[ref]; ok {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrDuplicateRef)
		}
		if prev, ok := seenNIF[row.nif]; ok {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w: refs=%s,%s", ErrDuplicateNIF, prev, ref)
		}
		if prev, ok := seenNome[row.nome]; ok {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w: refs=%s,%s", ErrDuplicateNome, prev, ref)
		}
		seenNIF[row.nif] = ref
		seenNome[row.nome] = ref
		seenRef[ref] = struct{}{}
		held = append(held, heldIdentity{
			ref:         ref,
			role:        role,
			bits:        bits,
			taxpayerNIF: row.nif,
			sourceLabel: row.nome, // structural trim only; not a tax-regime classification
			priv:        priv,
		})
	}
	if len(held) == 0 {
		return nil, ErrNoIdentities
	}
	return held, nil
}
