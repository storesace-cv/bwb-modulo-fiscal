// Package fepath encodes C-FE-001 fail-closed path handling:
// HML snapshot `/sigt/fe/ws/v1` ≠ PRD/aligned `/sigt/fe/v1`.
//
// Does not invent the «correct» AGT path and does not confirm AO-AGT-*.
package fepath

import (
	"fmt"
	"strings"
)

// Snapshot path prefixes (facts from FE HML HTML — pending_validation).
const (
	PrefixV1   = "/sigt/fe/v1"    // aligned on many services
	PrefixWSV1 = "/sigt/fe/ws/v1" // appears on some HML pages (C-FE-001)
)

// Hosts cited in FE HML snapshots (pending_validation; not AGT confirmation).
const (
	HostHML = "https://sifphml.minfin.gov.ao"
	HostPRD = "https://sifp.minfin.gov.ao"
)

// ConflictOpen remains true until AGT/docs close C-FE-001.
// Engineering mitigations must not flip this to false.
const ConflictOpen = true

// Services with documented HML vs PRD path/operation inconsistency (C-FE-001).
const (
	ServiceSolicitarSerie = "solicitarSerie"
	ServiceListarFacturas = "listarFacturas"
)

// Aligned services (HML and PRD both use PrefixV1 in the inventory snapshot).
var alignedServices = map[string]struct{}{
	"registarFactura":  {},
	"listarSeries":     {},
	"obterEstado":      {},
	"consultarFactura": {},
	"validarDocumento": {},
}

// Violation is a C-FE-001 path-separation breach.
type Violation struct {
	Code   string
	Reason string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s", v.Code, v.Reason)
}

// CheckInvariants verifies package-level separation used by the module.
func CheckInvariants() []Violation {
	out := make([]Violation, 0, 4)
	if PrefixV1 == PrefixWSV1 {
		out = append(out, Violation{Code: "prefix_ids", Reason: "prefixos /v1 e /ws/v1 colidem"})
	}
	if !strings.Contains(PrefixWSV1, "/ws/") {
		out = append(out, Violation{Code: "ws_marker", Reason: "PrefixWSV1 deve conter /ws/"})
	}
	if strings.Contains(PrefixV1, "/ws/") {
		out = append(out, Violation{Code: "v1_clean", Reason: "PrefixV1 não pode conter /ws/"})
	}
	if HostHML == HostPRD {
		out = append(out, Violation{Code: "hosts", Reason: "hosts HML e PRD colidem"})
	}
	if !ConflictOpen {
		out = append(out, Violation{
			Code:   "conflict_flag",
			Reason: "ConflictOpen=false exige fecho documentado C-FE-001 + revisão compliance",
		})
	}
	return out
}

// ServiceHasPathConflict reports services that must not get a constructed URL
// while ConflictOpen (solicitarSerie HML URL/op mismatch; listarFacturas /ws/).
func ServiceHasPathConflict(service string) bool {
	switch strings.TrimSpace(service) {
	case ServiceSolicitarSerie, ServiceListarFacturas:
		return true
	default:
		return false
	}
}

// ServiceIsAligned reports services whose HML and PRD snapshot paths both use PrefixV1.
func ServiceIsAligned(service string) bool {
	_, ok := alignedServices[strings.TrimSpace(service)]
	return ok
}

// RejectAmbiguousURL rejects a candidate URL that mixes /ws/v1 and /v1 semantics
// or targets a conflicted service while C-FE-001 is open.
func RejectAmbiguousURL(rawURL, service string) error {
	u := strings.TrimSpace(rawURL)
	svc := strings.TrimSpace(service)
	if u == "" || svc == "" {
		return fmt.Errorf("fepath: url/serviço vazios")
	}
	hasV1 := strings.Contains(u, PrefixV1)
	hasWS := strings.Contains(u, PrefixWSV1)
	if hasV1 && hasWS {
		return fmt.Errorf("fepath: URL mistura %s e %s (C-FE-001)", PrefixV1, PrefixWSV1)
	}
	if ConflictOpen && ServiceHasPathConflict(svc) {
		return fmt.Errorf("fepath: C-FE-001 aberto — recusar URL para %s", svc)
	}
	if ServiceHasPathConflict(svc) && hasWS {
		return fmt.Errorf("fepath: path /ws/ em serviço conflituoso %s sem fecho C-FE-001", svc)
	}
	return nil
}

// BuildAlignedURL builds a FE URL only for snapshot-aligned services using PrefixV1.
// Conflicted services always fail while ConflictOpen.
// Does not confirm AO-AGT-001 and does not choose between /ws/ and /v1 for conflicted ops.
func BuildAlignedURL(host, service string) (string, error) {
	h := strings.TrimRight(strings.TrimSpace(host), "/")
	svc := strings.TrimSpace(service)
	if h == "" || svc == "" {
		return "", fmt.Errorf("fepath: host/serviço vazios")
	}
	if ConflictOpen && ServiceHasPathConflict(svc) {
		return "", fmt.Errorf("fepath: C-FE-001 aberto — não construir URL para %s", svc)
	}
	if !ServiceIsAligned(svc) {
		return "", fmt.Errorf("fepath: serviço %q não está na lista alinhada do inventário", svc)
	}
	if h != HostHML && h != HostPRD {
		return "", fmt.Errorf("fepath: host %q fora dos hosts citados no snapshot FE", h)
	}
	return h + PrefixV1 + "/" + svc, nil
}
