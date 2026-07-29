package prep

import (
	"strings"
	"time"
)

// Connectivity validation states (scaffolding; ≠ AGT reachability).
const (
	ConnectivityAbsent     = "absent"
	ConnectivitySimulator  = "simulator_only"
	ConnectivityFailClosed = "fail_closed"
	ConnectivityBlockedAGT = "blocked_until_agt"
)

// ConnectivityStatus is sanitized owner-facing connectivity prep state.
type ConnectivityStatus struct {
	FiscalEnv        string
	AuthorityMode    string
	Status           string
	ExternalVerified bool
	SimulatorAllowed bool
	LastProbeAt      *time.Time
	LastProbeResult  string // success|denied|error|""
	LastProbeMode    string
	Notes            []string
}

// BuildConnectivityStatus aggregates fail-closed runtime flags + optional last probe audit.
func BuildConnectivityStatus(fiscalEnv, authorityMode, lastProbeResult, lastProbeMode string, lastProbeAt *time.Time) ConnectivityStatus {
	env := strings.TrimSpace(fiscalEnv)
	mode := strings.ToLower(strings.TrimSpace(authorityMode))
	if mode == "" {
		mode = "simulator"
	}
	out := ConnectivityStatus{
		FiscalEnv:        env,
		AuthorityMode:    mode,
		ExternalVerified: false,
		LastProbeResult:  strings.TrimSpace(lastProbeResult),
		LastProbeMode:    strings.TrimSpace(lastProbeMode),
		LastProbeAt:      lastProbeAt,
		Notes:            []string{"external_verified permanece false até probe AGT real (GAP-006 / RM-FE-001)"},
	}
	switch mode {
	case "agt-hml", "agt-prd":
		out.Status = ConnectivityBlockedAGT
		out.SimulatorAllowed = false
		out.Notes = append(out.Notes, "modo AGT reservado — sem chamada real (FailClosedProduction)")
		return out
	}
	if err := FailClosedProduction(env, mode); err != nil {
		out.Status = ConnectivityFailClosed
		out.SimulatorAllowed = false
		out.Notes = append(out.Notes, "produção+simulator ou modo inválido: fail-closed")
		return out
	}
	switch mode {
	case "simulator":
		out.Status = ConnectivitySimulator
		out.SimulatorAllowed = true
		out.Notes = append(out.Notes, "probe via POST /admin/v1/authority/probe-config (simulator≠AGT)")
	default:
		out.Status = ConnectivityAbsent
		out.SimulatorAllowed = false
		out.Notes = append(out.Notes, "modo autoridade desconhecido")
	}
	return out
}
