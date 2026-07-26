// Package doctype resolve tipos documentais OpenAPI → canónico do catálogo (DEC-REG-003).
//
// Não confirma AO-DOC-*. Não chama AGT. O material canonical_v2 continua a usar
// o enum OpenAPI (invoice / credit_note) para não partir goldens.
package doctype

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed document_catalog.md
var catalogMarkdown []byte

// API enums estáveis do OpenAPI (slice actual).
const (
	APIInvoice    = "invoice"
	APICreditNote = "credit_note"
	CanonicalFT   = "bwb.ao.vendas.ft"
	CanonicalNC   = "bwb.ao.vendas.nc"
	ActiveOn      = "on"
	ActiveOff     = "off"
)

var (
	// ErrUnknownAPI is returned when document_type is not a known OpenAPI enum.
	ErrUnknownAPI = errors.New("doctype: document_type desconhecido")
	// ErrInactive is returned when the mapped canonical type is not activo=on.
	ErrInactive = errors.New("doctype: tipo canónico inactivo no slice")
	// ErrCatalog is returned when the embedded catalog cannot be parsed or is inconsistent.
	ErrCatalog = errors.New("doctype: catálogo inválido")
)

// ChannelAdapters holds read-only FE/SAF-T mappings from the catalog (no AGT calls).
type ChannelAdapters struct {
	FECode        string // L4, empty if ∅
	SAFTType      string // e.g. InvoiceType=FT, empty if ∅
	SAFTStructure string // L3, empty if ∅
	Eligibility   string // SAF-T | FE | ambos
}

// Entry is one catalog seed row.
type Entry struct {
	Grupo           string
	CodigoCanonico  string
	Designacao      string
	Activo          string
	ChannelAdapters ChannelAdapters
}

// Resolved is the result of ResolveAPI for an active slice type.
type Resolved struct {
	APIType   string
	Canonical string
	Entry     Entry
}

// Registry is a fail-closed index of catalog entries.
type Registry struct {
	byCanonical map[string]Entry
}

var (
	defaultRegistry *Registry
	defaultLoadErr  error
)

func init() {
	defaultRegistry, defaultLoadErr = ParseCatalog(catalogMarkdown)
}

// Default returns the embedded catalog registry (fail-closed at init).
func Default() (*Registry, error) {
	if defaultLoadErr != nil {
		return nil, defaultLoadErr
	}
	return defaultRegistry, nil
}

// ParseCatalog parses the seed table from DOCUMENT-CATALOG markdown.
func ParseCatalog(md []byte) (*Registry, error) {
	lines := bytes.Split(md, []byte("\n"))
	var rows [][]string
	headerSeen := false
	for _, raw := range lines {
		line := string(raw)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if strings.Contains(line, "codigo_canonico") && strings.Contains(line, "grupo") {
			headerSeen = true
			continue
		}
		if !headerSeen {
			continue
		}
		if strings.HasPrefix(strings.ReplaceAll(line, " ", ""), "|---") {
			continue
		}
		cells := splitCells(line)
		if len(cells) < 13 {
			continue
		}
		canon := strings.Trim(cells[1], "`")
		if !strings.HasPrefix(canon, "bwb.ao.") {
			continue
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: nenhuma linha seed", ErrCatalog)
	}

	reg := &Registry{byCanonical: make(map[string]Entry, len(rows))}
	for _, cells := range rows {
		e, err := entryFromCells(cells)
		if err != nil {
			return nil, err
		}
		if _, dup := reg.byCanonical[e.CodigoCanonico]; dup {
			return nil, fmt.Errorf("%w: canónico duplicado %s", ErrCatalog, e.CodigoCanonico)
		}
		reg.byCanonical[e.CodigoCanonico] = e
	}

	ft, okFT := reg.byCanonical[CanonicalFT]
	nc, okNC := reg.byCanonical[CanonicalNC]
	if !okFT || !okNC {
		return nil, fmt.Errorf("%w: FT/NC ausentes no seed", ErrCatalog)
	}
	if ft.Activo != ActiveOn || nc.Activo != ActiveOn {
		return nil, fmt.Errorf("%w: DEC-REG-003 exige FT e NC activo=on", ErrCatalog)
	}
	return reg, nil
}

func splitCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func entryFromCells(cells []string) (Entry, error) {
	canon := strings.Trim(cells[1], "`")
	activo := cells[len(cells)-1]
	if activo != ActiveOn && activo != ActiveOff && activo != "pending_dec_reg_003" {
		return Entry{}, fmt.Errorf("%w: activo inválido %q em %s", ErrCatalog, activo, canon)
	}
	canal := cells[3]
	fe, saft := parseChannelCodes(canal)
	estrutura := strings.Trim(cells[4], "`")
	if estrutura == "∅" {
		estrutura = ""
	}
	return Entry{
		Grupo:          cells[0],
		CodigoCanonico: canon,
		Designacao:     cells[2],
		Activo:         activo,
		ChannelAdapters: ChannelAdapters{
			FECode:        fe,
			SAFTType:      saft,
			SAFTStructure: estrutura,
			Eligibility:   cells[5],
		},
	}, nil
}

func parseChannelCodes(s string) (fe, saft string) {
	fe = extractBacktickAfter(s, "FE:")
	saft = extractBacktickAfter(s, "SAFT:")
	return fe, saft
}

func extractBacktickAfter(s, prefix string) string {
	i := strings.Index(s, prefix)
	if i < 0 {
		return ""
	}
	rest := s[i+len(prefix):]
	if !strings.HasPrefix(rest, "`") {
		return ""
	}
	rest = rest[1:]
	j := strings.IndexByte(rest, '`')
	if j < 0 {
		return ""
	}
	v := rest[:j]
	if v == "∅" {
		return ""
	}
	return v
}

// Lookup returns a catalog entry by canonical id.
func (r *Registry) Lookup(canonical string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	e, ok := r.byCanonical[canonical]
	return e, ok
}

// ResolveAPI maps OpenAPI document_type to an active canonical entry.
// Does not change the OpenAPI string used in canonical_v2 hashing.
func (r *Registry) ResolveAPI(documentType string) (Resolved, error) {
	if r == nil {
		return Resolved{}, ErrCatalog
	}
	var canonical string
	switch documentType {
	case APIInvoice:
		canonical = CanonicalFT
	case APICreditNote:
		canonical = CanonicalNC
	default:
		return Resolved{}, fmt.Errorf("%w: %q", ErrUnknownAPI, documentType)
	}
	e, ok := r.byCanonical[canonical]
	if !ok {
		return Resolved{}, fmt.Errorf("%w: canónico %s ausente", ErrCatalog, canonical)
	}
	if e.Activo != ActiveOn {
		return Resolved{}, fmt.Errorf("%w: %s", ErrInactive, canonical)
	}
	return Resolved{APIType: documentType, Canonical: canonical, Entry: e}, nil
}

// ResolveAPI is a convenience over Default().
func ResolveAPI(documentType string) (Resolved, error) {
	reg, err := Default()
	if err != nil {
		return Resolved{}, err
	}
	return reg.ResolveAPI(documentType)
}
