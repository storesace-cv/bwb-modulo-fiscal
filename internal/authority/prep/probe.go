// Package prep prepares authority connectivity checks without calling AGT.
//
// Simulator probe ≠ HML/PRD. external_verified must stay false.
package prep

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
)

var (
	// ErrFailClosed rejects unsafe runtime combinations.
	ErrFailClosed = errors.New("authority/prep: fail-closed")
	// ErrNotSimulator means probe only runs against the internal simulator.
	ErrNotSimulator = errors.New("authority/prep: probe só com simulator (≠ AGT)")
)

// Readiness mirrors AuthorityProfile flags (sanitized).
type Readiness struct {
	ConfigReady      bool
	SecretsReady     bool
	OfflineValidated bool
	ExternalVerified bool // must be false for prep
}

// ProbeResult is a sanitized connection/config test result.
type ProbeResult struct {
	OK                 bool
	Mode               string
	SimulatorReachable bool
	Outcome            string
	AuthorityRequestID string
	ExternalVerified   bool // always false
	Notes              []string
	ProbedAt           string
}

// FailClosedProduction rejects FISCAL_AUTHORITY=simulator when FISCAL_ENV=production.
func FailClosedProduction(fiscalEnv, authorityMode string) error {
	env := strings.ToLower(strings.TrimSpace(fiscalEnv))
	mode := strings.ToLower(strings.TrimSpace(authorityMode))
	if env == "production" && mode == config.AuthoritySimulator {
		return fmt.Errorf("%w: produção não pode usar FISCAL_AUTHORITY=simulator", ErrFailClosed)
	}
	if mode == config.AuthorityAGTHML || mode == config.AuthorityAGTPRD {
		return fmt.Errorf("%w: %s reservado até AGT real (GAP-006 / RM-FE-001)", ErrFailClosed, mode)
	}
	return nil
}

// RequirePrepForReservedModes documents that agt-* stay closed even with readiness.
func RequirePrepForReservedModes(mode string, r Readiness) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != config.AuthorityAGTHML && mode != config.AuthorityAGTPRD {
		return nil
	}
	if r.ExternalVerified {
		return fmt.Errorf("%w: external_verified não pode ser true sem AGT", ErrFailClosed)
	}
	if !r.ConfigReady || !r.SecretsReady || !r.OfflineValidated {
		return fmt.Errorf("%w: %s exige config_ready+secrets_ready+offline_validated (ainda assim ≠ AGT até FE-001)", ErrFailClosed, mode)
	}
	return fmt.Errorf("%w: %s ainda reservado (sem transporte AGT)", ErrFailClosed, mode)
}

// ProbeSimulator runs a synthetic Submit against the in-process simulator.
func ProbeSimulator(ctx context.Context, authorityMode string, client *simulator.Client) (ProbeResult, error) {
	mode := strings.ToLower(strings.TrimSpace(authorityMode))
	if mode != config.AuthoritySimulator {
		return ProbeResult{}, ErrNotSimulator
	}
	if client == nil {
		return ProbeResult{}, fmt.Errorf("%w: client nil", ErrFailClosed)
	}
	if err := ctx.Err(); err != nil {
		return ProbeResult{}, err
	}
	res, err := client.Submit(ctx, simulator.Request{
		SubmissionID: "prep-probe-" + time.Now().UTC().Format("20060102"),
		DocumentID:   "prep-probe-doc",
	})
	out := ProbeResult{
		Mode:             config.AuthoritySimulator,
		ExternalVerified: false,
		ProbedAt:         time.Now().UTC().Format(time.RFC3339),
		Notes:            []string{"simulator≠AGT", "external_verified=false"},
	}
	if err != nil {
		out.OK = false
		out.Notes = append(out.Notes, "submit falhou")
		return out, nil // report soft failure without leaking internals
	}
	out.OK = true
	out.SimulatorReachable = true
	out.Outcome = string(res.Outcome)
	out.AuthorityRequestID = res.AuthorityRequestID
	return out, nil
}
