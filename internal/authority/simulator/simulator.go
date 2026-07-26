// Package simulator implements an in-process AGT authority stub for CI/demo.
//
// It is NOT the AGT HML/PRD environment. Do not use outcomes as compliance evidence.
package simulator

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Outcome is a scripted simulator result (technical; ≠ certified AGT).
type Outcome string

const (
	OutcomeAccept      Outcome = "accept"
	OutcomeReject      Outcome = "reject"
	OutcomeUnavailable Outcome = "unavailable"
	OutcomeUnknown     Outcome = "unknown"
)

var (
	// ErrUnavailable means the simulator refused the call; outbox must retry.
	ErrUnavailable = errors.New("simulator: unavailable")
)

// Result is returned on a successful transport to the simulator (including reject/unknown).
type Result struct {
	AuthorityRequestID string
	Outcome            Outcome // accept | reject | unknown
}

// Client is a fail-closed scripted authority. Marked non-AGT.
type Client struct {
	mu           sync.Mutex
	Default      Outcome
	BySubmission map[string]Outcome
	calls        map[string]int
}

// New returns a client with default accept when unscripted.
func New(defaultOutcome Outcome) *Client {
	if defaultOutcome == "" {
		defaultOutcome = OutcomeAccept
	}
	return &Client{
		Default:      defaultOutcome,
		BySubmission: make(map[string]Outcome),
		calls:        make(map[string]int),
	}
}

// Script sets the outcome for a stable submission_id.
func (c *Client) Script(submissionID string, outcome Outcome) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.BySubmission == nil {
		c.BySubmission = make(map[string]Outcome)
	}
	c.BySubmission[submissionID] = outcome
}

// Submit simulates an authority submission. Never talks to AGT.
func (c *Client) Submit(ctx context.Context, submissionID string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if submissionID == "" {
		return Result{}, fmt.Errorf("simulator: empty submission_id")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[submissionID]++
	out := c.Default
	if v, ok := c.BySubmission[submissionID]; ok {
		out = v
	}
	switch out {
	case OutcomeUnavailable:
		return Result{}, ErrUnavailable
	case OutcomeAccept, OutcomeReject, OutcomeUnknown:
		return Result{
			AuthorityRequestID: "sim-" + submissionID,
			Outcome:            out,
		}, nil
	default:
		return Result{}, fmt.Errorf("simulator: outcome inválido %q", out)
	}
}

// CallCount returns how many Submit calls were made for submissionID (tests).
func (c *Client) CallCount(submissionID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[submissionID]
}
