package prep

import (
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fepath"
)

// Endpoint path validation states (scaffolding; ≠ AGT confirmation).
const (
	PathStatusAligned         = "aligned"
	PathStatusConflictOpen    = "conflict_open"
	PathStatusPendingExternal = "pending_external"
)

// EndpointCatalogEntry is a non-secret FE operation binding candidate from the
// provisional matrix / fepath inventory. Never invents paths.
type EndpointCatalogEntry struct {
	Operation   string
	Environment string // homologation | production
	PathStatus  string
	DeclaredURL string // empty when conflict_open or pending_external
	Host        string
	ConflictID  string // e.g. C-FE-001
	SourceNote  string
	Bindable    bool // true when admitted in AuthorityProfile KnownAuthorityOperations
}

// knownMatrixOperations is the inventory cited in FE-SERVICES-MATRIX (pending_validation).
// Includes ops beyond AuthorityProfile bindables when matrix lists them.
var knownMatrixOperations = []string{
	"registarFactura",
	"solicitarSerie",
	"listarSeries",
	"obterEstado",
	"listarFacturas",
	"consultarFactura",
	"validarDocumento",
}

// EndpointCatalog returns fail-closed catalog rows for one environment.
// Homologation/production hosts come from fepath snapshot constants only.
func EndpointCatalog(environment string) ([]EndpointCatalogEntry, error) {
	env := strings.TrimSpace(environment)
	var host string
	switch env {
	case adminregistry.EnvHomologation:
		host = fepath.HostHML
	case adminregistry.EnvProduction:
		host = fepath.HostPRD
	default:
		return nil, fmt.Errorf("prep: environment deve ser homologation|production")
	}

	out := make([]EndpointCatalogEntry, 0, len(knownMatrixOperations))
	for _, op := range knownMatrixOperations {
		_, bindable := adminregistry.KnownAuthorityOperations[op]
		entry := EndpointCatalogEntry{
			Operation:   op,
			Environment: env,
			Host:        host,
			Bindable:    bindable,
			SourceNote:  "FE-SERVICES-MATRIX + fepath; pending_validation; ≠ AO-AGT",
		}
		switch {
		case fepath.ServiceHasPathConflict(op):
			entry.PathStatus = PathStatusConflictOpen
			entry.ConflictID = "C-FE-001"
			entry.DeclaredURL = ""
			entry.SourceNote = "C-FE-001 aberto — URL não construída (fail-closed)"
		case fepath.ServiceIsAligned(op):
			url, err := fepath.BuildAlignedURL(host, op)
			if err != nil {
				entry.PathStatus = PathStatusPendingExternal
				entry.DeclaredURL = ""
				entry.SourceNote = "alinhado no inventário mas BuildAlignedURL recusou: " + err.Error()
			} else {
				entry.PathStatus = PathStatusAligned
				entry.DeclaredURL = url
			}
		default:
			entry.PathStatus = PathStatusPendingExternal
			entry.DeclaredURL = ""
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Operation < out[j].Operation })
	return out, nil
}
