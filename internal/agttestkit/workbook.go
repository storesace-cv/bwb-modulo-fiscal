package agttestkit

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Validation / inventory errors (sanitized; never embed cell contents).
var (
	ErrPathRequired          = errors.New("agttestkit: workbook path required")
	ErrOpenWorkbook          = errors.New("agttestkit: cannot open workbook")
	ErrNoUsableSheet         = errors.New("agttestkit: no usable sheet with expected headers")
	ErrAmbiguousSheets       = errors.New("agttestkit: multiple sheets match expected headers")
	ErrUnexpectedColumns     = errors.New("agttestkit: unexpected columns")
	ErrMissingRequiredHeader = errors.New("agttestkit: missing required header")
	ErrMissingRequiredCell   = errors.New("agttestkit: missing required cell")
	ErrDuplicateNIF          = errors.New("agttestkit: duplicate nif")
	ErrDuplicateNome         = errors.New("agttestkit: duplicate nome")
	ErrInvalidPrivatePEM     = errors.New("agttestkit: invalid private pem")
	ErrInvalidPublicPEM      = errors.New("agttestkit: invalid public pem")
	ErrKeyPairMismatch       = errors.New("agttestkit: public/private pair mismatch")
	ErrUnsupportedKey        = errors.New("agttestkit: unsupported key algorithm")
	ErrRSATooSmall           = errors.New("agttestkit: rsa key below minimum for rs256")
	ErrNoIdentities          = errors.New("agttestkit: no identity rows")
)

// MinRSABits is the FE-documented minimum for RS256 (AO-FE-SNAP-HML-2026-07-25-ESTRUTURA / GESTAO).
const MinRSABits = 2048

// IdentityInventory is the only public-facing inventory shape (sanitized).
type IdentityInventory struct {
	IdentityCount   int                 `json:"identity_count"`
	SheetName       string              `json:"sheet_name"`
	Algorithm       string              `json:"algorithm"`
	MinRSABits      int                 `json:"min_rsa_bits_required"`
	AllPairsValid   bool                `json:"all_pairs_valid"`
	Identities      []SanitizedIdentity `json:"identities"`
	Notes           []string            `json:"notes"`
	SourceCitations []string            `json:"source_citations"`
}

// SanitizedIdentity never includes NIF, name, or PEM.
type SanitizedIdentity struct {
	OpaqueRef         string `json:"opaque_ref"`
	Algorithm         string `json:"algorithm"`
	RSABits           int    `json:"rsa_bits"`
	PublicFingerprint string `json:"public_fingerprint_sha256"`
	PairMatch         bool   `json:"pair_match"`
	MeetsMinRSABits   bool   `json:"meets_min_rsa_bits"`
	NIFHashTruncated  string `json:"nif_hash_truncated"` // sha256(nif)[:16] hex; domain = inventory
	NomeHashTruncated string `json:"nome_hash_truncated"`
}

type rawRow struct {
	sheet    string
	excelRow int
	nif      string
	nome     string
	privPEM  []byte
	pubPEM   []byte
}

// LoadAndValidate opens path, validates structure and RSA pairs, returns sanitized inventory.
// Private material is zeroed before return.
func LoadAndValidate(path string) (IdentityInventory, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return IdentityInventory{}, ErrPathRequired
	}
	if _, err := os.Stat(path); err != nil {
		return IdentityInventory{}, fmt.Errorf("%w: path inaccessible", ErrOpenWorkbook)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return IdentityInventory{}, fmt.Errorf("%w", ErrOpenWorkbook)
	}
	defer func() { _ = f.Close() }()

	sheet, rows, err := extractRows(f)
	if err != nil {
		return IdentityInventory{}, err
	}
	defer zeroRows(rows)

	inv, err := validateRows(sheet, rows)
	zeroRows(rows)
	return inv, err
}

func extractRows(f *excelize.File) (string, []rawRow, error) {
	sheets := f.GetSheetList()
	type candidate struct {
		name string
		cols map[string]int
	}
	var matched []candidate
	for _, name := range sheets {
		cols, ok, err := detectHeader(f, name)
		if err != nil {
			return "", nil, err
		}
		if ok {
			matched = append(matched, candidate{name: name, cols: cols})
		}
	}
	switch len(matched) {
	case 0:
		return "", nil, ErrNoUsableSheet
	case 1:
		// ok
	default:
		return "", nil, ErrAmbiguousSheets
	}
	c := matched[0]
	rows, err := readDataRows(f, c.name, c.cols)
	return c.name, rows, err
}

func detectHeader(f *excelize.File, sheet string) (map[string]int, bool, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, false, fmt.Errorf("%w", ErrOpenWorkbook)
	}
	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		// Reject unexpected non-empty cells beyond mapped columns after mapping attempt.
		roles := make(map[string]int)
		unexpected := false
		for i, cell := range row {
			tok := normalizeHeader(cell)
			if tok == "" {
				continue
			}
			role, ok := mapNormalizedHeader(tok)
			if !ok {
				unexpected = true
				break
			}
			if _, exists := roles[role]; exists {
				unexpected = true
				break
			}
			roles[role] = i
		}
		if unexpected {
			// Not a header row if it looks like data; keep scanning.
			// If this row has any mapped role but also unexpected tokens → fail.
			if len(roles) > 0 {
				return nil, false, ErrUnexpectedColumns
			}
			continue
		}
		if len(roles) == 0 {
			continue
		}
		for _, req := range expectedHeaderOrder {
			if _, ok := roles[req]; !ok {
				return nil, false, ErrMissingRequiredHeader
			}
		}
		if len(roles) != len(expectedHeaderOrder) {
			return nil, false, ErrUnexpectedColumns
		}
		return roles, true, nil
	}
	return nil, false, nil
}

func readDataRows(f *excelize.File, sheet string, cols map[string]int) ([]rawRow, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrOpenWorkbook)
	}
	headerSeen := false
	var out []rawRow
	for i, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		if !headerSeen {
			// Skip until we pass the header row (same detection as detectHeader).
			if rowLooksLikeHeader(row) {
				headerSeen = true
			}
			continue
		}
		nif := cellAt(row, cols[colNIF])
		nome := cellAt(row, cols[colNome])
		priv := normalizePEMCell(cellAt(row, cols[colPrivada]))
		pub := normalizePEMCell(cellAt(row, cols[colPublica]))
		if nif == "" && nome == "" && priv == "" && pub == "" {
			continue
		}
		if nif == "" || nome == "" || priv == "" || pub == "" {
			return nil, fmt.Errorf("%w: row_index=%d", ErrMissingRequiredCell, i+1)
		}
		out = append(out, rawRow{
			sheet:    sheet,
			excelRow: i + 1,
			nif:      nif,
			nome:     nome,
			privPEM:  []byte(priv),
			pubPEM:   []byte(pub),
		})
	}
	if !headerSeen {
		return nil, ErrNoUsableSheet
	}
	if len(out) == 0 {
		return nil, ErrNoIdentities
	}
	return out, nil
}

func rowLooksLikeHeader(row []string) bool {
	roles := 0
	for _, cell := range row {
		tok := normalizeHeader(cell)
		if tok == "" {
			continue
		}
		if _, ok := mapNormalizedHeader(tok); ok {
			roles++
		}
	}
	return roles == len(expectedHeaderOrder)
}

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// normalizePEMCell strips incidental spreadsheet quoting without altering key material.
func normalizePEMCell(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = strings.TrimSpace(s[1 : len(s)-1])
		}
	}
	// Some spreadsheet exports escape quotes by doubling them inside a quoted field.
	s = strings.ReplaceAll(s, "\"\"", "\"")
	return strings.TrimSpace(s)
}

func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func zeroRows(rows []rawRow) {
	for i := range rows {
		zeroBytes(rows[i].privPEM)
		zeroBytes(rows[i].pubPEM)
		rows[i].nif = ""
		rows[i].nome = ""
	}
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func truncatedHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// opaqueRefDomain separates opaque identity refs from raw public fingerprints.
const opaqueRefDomain = "bwb.agttestkit.identity.v1"

func opaqueRefFromPublic(pub *rsa.PublicKey) string {
	if pub == nil {
		return "agt-test-invalid"
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "agt-test-invalid"
	}
	h := sha256.New()
	_, _ = h.Write([]byte(opaqueRefDomain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(der)
	sum := h.Sum(nil)
	return "agt-test-" + hex.EncodeToString(sum[:8])
}
