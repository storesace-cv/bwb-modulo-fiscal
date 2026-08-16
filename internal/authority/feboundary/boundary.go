// Package feboundary drives FE submissions to the local BWB-MOCK boundary (RM-FEFIX-005).
//
// States stop at fixture_boundary_*; never authority_accepted / homologação AGT.
// Requires fehub.KindFixture + femock. Distinct from persistence outbox→simulator (fiscaljws).
package feboundary

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fehub"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/femock"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

// Document/submission states until the fixture boundary (≠ DEC-PROD-009 accepted).
const (
	StateQueued   = "queued_for_fixture_boundary"
	StateInFlight = "fixture_boundary_in_flight"
	StateOK       = "fixture_boundary_ok"
	StateReject   = "fixture_boundary_reject"
	StateBlocked  = "fixture_boundary_profile_blocked"
	StateFailed   = "fixture_boundary_transport_failed"
)

// Supported mock operations (payload builders + BWB-MOCK). softwareInfo is synthetic mock-only.
const (
	OpSoftwareInfo     = "softwareInfo"
	OpObterEstado      = "obterEstado"
	OpConsultarFactura = "consultarFactura"
)

var (
	ErrHubRequired  = errors.New("feboundary: hub required")
	ErrClosed       = errors.New("feboundary: closed")
	ErrUnknownOp    = errors.New("feboundary: unknown operation")
	ErrNotFound     = errors.New("feboundary: submission not found")
	ErrInvalidInput = errors.New("feboundary: invalid input")
)

// Submission is in-memory FE prep state (sanitized; no PEM/NIF/JWS full dump).
type Submission struct {
	ID            string
	Operation     string
	State         string
	Attempts      int
	MockRequestID string
	MockCode      string
	SourceID      string
	Note          string
	UpdatedAt     time.Time
}

// IsAGTAccepted is always false — fixture/mock success ≠ AGT acceptance.
func (s Submission) IsAGTAccepted() bool { return false }

// Engine orchestrates queue→sign(BWB-MOCK)→femock until boundary.
type Engine struct {
	hub      fehub.Hub
	provider agttestkit.IdentityProvider
	baseURL  string // e.g. httptest.URL + no trailing slash issues
	user     string
	pass     string
	client   *http.Client

	mu     sync.Mutex
	closed bool
	byID   map[string]*Submission
}

// Config wires fixture hub + identities + femock base URL (httptest only).
type Config struct {
	Hub      fehub.Hub
	Provider agttestkit.IdentityProvider
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
}

// New validates config (fixture transport only).
func New(cfg Config) (*Engine, error) {
	if cfg.Provider == nil {
		return nil, ErrInvalidInput
	}
	if err := cfg.Hub.AssertTransportAllowed(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || cfg.Username == "" || cfg.Password == "" {
		return nil, ErrInvalidInput
	}
	cli := cfg.Client
	if cli == nil {
		cli = &http.Client{Timeout: 5 * time.Second}
	}
	return &Engine{
		hub:      cfg.Hub,
		provider: cfg.Provider,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		user:     cfg.Username,
		pass:     cfg.Password,
		client:   cli,
		byID:     make(map[string]*Submission),
	}, nil
}

// Close clears submissions and credentials (best-effort).
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	e.byID = make(map[string]*Submission)
	e.user, e.pass = "", ""
	return nil
}

// Enqueue creates a queued submission (does not call AGT).
func (e *Engine) Enqueue(op string) (Submission, error) {
	switch op {
	case OpSoftwareInfo, OpObterEstado, OpConsultarFactura:
	default:
		return Submission{}, fmt.Errorf("%w: %s", ErrUnknownOp, op)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return Submission{}, ErrClosed
	}
	id := newID()
	s := &Submission{
		ID:        id,
		Operation: op,
		State:     StateQueued,
		Note:      "queued local fixture boundary; ≠ AGT",
		UpdatedAt: time.Now().UTC(),
	}
	e.byID[id] = s
	return *s, nil
}

// Get returns a copy of submission state.
func (e *Engine) Get(id string) (Submission, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s, ok := e.byID[id]
	if !ok {
		return Submission{}, ErrNotFound
	}
	return *s, nil
}

// ProcessInput carries operation-specific synthetic claims + identity ref.
type ProcessInput struct {
	SubmissionID   string
	IdentityRef    string
	IdempotencyKey string
	Software       *feprofile.SoftwareInfoClaims
	ObterEstado    *feprofile.ObterEstadoRequestClaims
	Consultar      *feprofile.ConsultarFacturaRequestClaims
}

// Process advances one attempt: sign BWB-MOCK → POST femock → update state.
// Never sets IsAGTAccepted / authority_accepted.
func (e *Engine) Process(ctx context.Context, in ProcessInput) (Submission, error) {
	if err := ctx.Err(); err != nil {
		return Submission{}, err
	}
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return Submission{}, ErrClosed
	}
	s, ok := e.byID[in.SubmissionID]
	if !ok {
		e.mu.Unlock()
		return Submission{}, ErrNotFound
	}
	if in.IdentityRef == "" || in.IdempotencyKey == "" {
		e.mu.Unlock()
		return Submission{}, ErrInvalidInput
	}
	s.State = StateInFlight
	s.Attempts++
	s.UpdatedAt = time.Now().UTC()
	op := s.Operation
	e.mu.Unlock()

	jws, err := e.sign(op, in)
	if err != nil {
		return e.fail(in.SubmissionID, StateFailed, "", "", "sign failed (sanitized)")
	}
	status, body, err := e.postMock(ctx, op, in.IdentityRef, jws, in.IdempotencyKey)
	if err != nil {
		return e.fail(in.SubmissionID, StateFailed, "", "", "transport failed (sanitized)")
	}
	code, reqID, src := extractMockFields(body)
	switch {
	case status == http.StatusOK:
		return e.finish(in.SubmissionID, StateOK, reqID, code, src, "fixture_boundary_ok ≠ AGT homologation")
	case strings.Contains(code, femock.CodeProfileBlocked) || code == femock.CodeProfileBlocked:
		return e.finish(in.SubmissionID, StateBlocked, reqID, code, src, "wire profile blocked")
	case status == http.StatusUnprocessableEntity && strings.HasPrefix(code, "FE-RNG-"):
		return e.finish(in.SubmissionID, StateReject, reqID, code, src, "simulated FE-RNG ≠ AGT live")
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return e.finish(in.SubmissionID, StateReject, reqID, code, src, "mock authz reject ≠ AGT")
	default:
		return e.finish(in.SubmissionID, StateReject, reqID, code, src, "mock non-OK ≠ AGT")
	}
}

func (e *Engine) sign(op string, in ProcessInput) (string, error) {
	switch op {
	case OpSoftwareInfo:
		if in.Software == nil {
			return "", ErrInvalidInput
		}
		return femock.SignSoftwareMock(e.provider, in.IdentityRef, *in.Software)
	case OpObterEstado:
		if in.ObterEstado == nil {
			return "", ErrInvalidInput
		}
		return femock.SignObterEstadoMock(e.provider, in.IdentityRef, *in.ObterEstado)
	case OpConsultarFactura:
		if in.Consultar == nil {
			return "", ErrInvalidInput
		}
		return femock.SignConsultarFacturaMock(e.provider, in.IdentityRef, *in.Consultar)
	default:
		return "", ErrUnknownOp
	}
}

func (e *Engine) postMock(ctx context.Context, op, ref, jws, idem string) (int, map[string]any, error) {
	payload, _ := json.Marshal(map[string]string{
		"identityRef":    ref,
		"jws":            jws,
		"idempotencyKey": idem,
	})
	url := e.baseURL + femock.PathPrefix + "/" + op
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(e.user, e.pass)
	res, err := e.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return res.StatusCode, nil, err
	}
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	return res.StatusCode, body, nil
}

func extractMockFields(body map[string]any) (code, reqID, src string) {
	if body == nil {
		return "", "", ""
	}
	code, _ = body["code"].(string)
	reqID, _ = body["requestID"].(string)
	src, _ = body["source_id"].(string)
	if code == "" {
		if st, _ := body["status"].(string); st == "ok" {
			code = "ok"
		}
	}
	return code, reqID, src
}

func (e *Engine) finish(id, state, reqID, code, src, note string) (Submission, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s, ok := e.byID[id]
	if !ok {
		return Submission{}, ErrNotFound
	}
	s.State = state
	s.MockRequestID = reqID
	s.MockCode = code
	s.SourceID = src
	s.Note = note
	s.UpdatedAt = time.Now().UTC()
	return *s, nil
}

func (e *Engine) fail(id, state, reqID, code, note string) (Submission, error) {
	return e.finish(id, state, reqID, code, "", note)
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "feb-" + hex.EncodeToString(b[:])
}

// HubView exposes sanitized hub metadata.
func (e *Engine) HubView() fehub.PublicView {
	return e.hub.View()
}
