// Package feboundary drives FE submissions to the local BWB-MOCK boundary (RM-FEFIX-005/006).
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
	"net"
	"net/http"
	"net/url"
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
	ErrHubRequired       = errors.New("feboundary: hub required")
	ErrClosed            = errors.New("feboundary: closed")
	ErrUnknownOp         = errors.New("feboundary: unknown operation")
	ErrNotFound          = errors.New("feboundary: submission not found")
	ErrInvalidInput      = errors.New("feboundary: invalid input")
	ErrBaseURLRejected   = errors.New("feboundary: base URL rejected")
	ErrRedirectDenied    = errors.New("feboundary: HTTP redirect denied")
	ErrInFlight          = errors.New("feboundary: submission already in flight")
	ErrInvalidTransition = errors.New("feboundary: invalid state transition")
	ErrMockResponse      = errors.New("feboundary: mock response rejected")
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

type dialSnapshot struct {
	baseURL string
	user    string
	pass    string
	client  *http.Client
}

// Engine orchestrates queue→sign(BWB-MOCK)→femock until boundary.
type Engine struct {
	hub      fehub.Hub
	provider agttestkit.IdentityProvider
	baseURL  string
	user     string
	pass     string
	client   *http.Client

	mu     sync.Mutex
	closed bool
	byID   map[string]*Submission
}

// Config wires fixture hub + identities + femock base URL (loopback HTTP only).
type Config struct {
	Hub      fehub.Hub
	Provider agttestkit.IdentityProvider
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
}

// New validates config (fixture transport only; BaseURL loopback-only).
func New(cfg Config) (*Engine, error) {
	if cfg.Provider == nil {
		return nil, ErrInvalidInput
	}
	if err := cfg.Hub.AssertTransportAllowed(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
		return nil, ErrInvalidInput
	}
	base, err := normalizeFixtureBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	cli := cfg.Client
	if cli == nil {
		cli = &http.Client{Timeout: 5 * time.Second}
	}
	// Always deny redirects (SSRF / host pivot).
	cli.CheckRedirect = func(*http.Request, []*http.Request) error {
		return ErrRedirectDenied
	}
	return &Engine{
		hub:      cfg.Hub,
		provider: cfg.Provider,
		baseURL:  base,
		user:     cfg.Username,
		pass:     cfg.Password,
		client:   cli,
		byID:     make(map[string]*Submission),
	}, nil
}

func normalizeFixtureBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrBaseURLRejected
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ErrBaseURLRejected
	}
	if !strings.EqualFold(u.Scheme, "http") {
		return "", fmt.Errorf("%w: scheme", ErrBaseURLRejected)
	}
	if u.User != nil {
		return "", fmt.Errorf("%w: userinfo", ErrBaseURLRejected)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%w: query/fragment", ErrBaseURLRejected)
	}
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	if path != "" {
		return "", fmt.Errorf("%w: path", ErrBaseURLRejected)
	}
	host := u.Hostname()
	if !isAuthorizedLoopbackHost(host) {
		return "", fmt.Errorf("%w: host", ErrBaseURLRejected)
	}
	// Rebuild without userinfo/query/fragment; keep port.
	out := &url.URL{Scheme: "http", Host: u.Host}
	return strings.TrimRight(out.String(), "/"), nil
}

func isAuthorizedLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
	if in.IdentityRef == "" || in.IdempotencyKey == "" {
		return Submission{}, ErrInvalidInput
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
	switch s.State {
	case StateQueued:
		// ok
	case StateInFlight:
		e.mu.Unlock()
		return Submission{}, ErrInFlight
	default:
		e.mu.Unlock()
		return Submission{}, ErrInvalidTransition
	}
	s.State = StateInFlight
	s.Attempts++
	s.UpdatedAt = time.Now().UTC()
	op := s.Operation
	snap := dialSnapshot{baseURL: e.baseURL, user: e.user, pass: e.pass, client: e.client}
	e.mu.Unlock()

	jws, err := e.sign(op, in)
	if err != nil {
		_, _ = e.fail(in.SubmissionID, StateFailed, "", "", "sign/validation failed (sanitized)")
		return Submission{}, err
	}
	status, ct, body, err := e.postMock(ctx, snap, op, in.IdentityRef, jws, in.IdempotencyKey)
	if err != nil {
		return e.fail(in.SubmissionID, StateFailed, "", "", "transport failed (sanitized)")
	}

	code, reqID, src, mockOK := classifyMockBody(body)
	switch {
	case status == http.StatusOK:
		if err := assertMockSuccess(op, ct, body, mockOK, reqID); err != nil {
			return e.fail(in.SubmissionID, StateFailed, reqID, code, "mock success body rejected (sanitized)")
		}
		return e.finish(in.SubmissionID, StateOK, reqID, "ok", src, "fixture_boundary_ok ≠ AGT homologation")
	case code == femock.CodeProfileBlocked || strings.Contains(code, femock.CodeProfileBlocked):
		// Defensive: Enqueue does not allow wire-blocked ops; still map if a stub returns it (RM-FEFIX-006 P3).
		if !isMockEnvelope(body) || reqID == "" {
			return e.fail(in.SubmissionID, StateFailed, reqID, code, "mock blocked body rejected (sanitized)")
		}
		return e.finish(in.SubmissionID, StateBlocked, reqID, code, src, "wire profile blocked")
	case status == http.StatusUnprocessableEntity && strings.HasPrefix(code, "FE-RNG-"):
		if !isMockEnvelope(body) || reqID == "" {
			return e.fail(in.SubmissionID, StateFailed, reqID, code, "mock FE-RNG body rejected (sanitized)")
		}
		return e.finish(in.SubmissionID, StateReject, reqID, code, src, "simulated FE-RNG ≠ AGT live")
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		if !isMockEnvelope(body) {
			return e.fail(in.SubmissionID, StateFailed, reqID, code, "mock authz body rejected (sanitized)")
		}
		return e.finish(in.SubmissionID, StateReject, reqID, code, src, "mock authz reject ≠ AGT")
	default:
		if body == nil {
			return e.fail(in.SubmissionID, StateFailed, "", "", "empty/malformed mock response")
		}
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

func (e *Engine) postMock(ctx context.Context, snap dialSnapshot, op, ref, jws, idem string) (int, string, map[string]any, error) {
	payload, err := json.Marshal(map[string]string{
		"identityRef":    ref,
		"jws":            jws,
		"idempotencyKey": idem,
	})
	if err != nil {
		return 0, "", nil, err
	}
	urlStr := snap.baseURL + femock.PathPrefix + "/" + op
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(payload))
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(snap.user, snap.pass)
	res, err := snap.client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer res.Body.Close()
	ct := res.Header.Get("Content-Type")
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return res.StatusCode, ct, nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return res.StatusCode, ct, nil, nil
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return res.StatusCode, ct, nil, nil
	}
	return res.StatusCode, ct, body, nil
}

func classifyMockBody(body map[string]any) (code, reqID, src string, mockOK bool) {
	if body == nil {
		return "", "", "", false
	}
	code, _ = body["code"].(string)
	reqID, _ = body["requestID"].(string)
	src, _ = body["source_id"].(string)
	st, _ := body["status"].(string)
	mockOK = strings.EqualFold(st, "ok")
	if code == "" && mockOK {
		code = "ok"
	}
	return code, reqID, src, mockOK
}

func isMockEnvelope(body map[string]any) bool {
	if body == nil {
		return false
	}
	sim, _ := body["simulated"].(bool)
	mock, _ := body["mock"].(string)
	return sim && mock == femock.TypMock
}

func assertMockSuccess(op, contentType string, body map[string]any, mockOK bool, reqID string) error {
	if body == nil {
		return ErrMockResponse
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" || !strings.HasPrefix(ct, "application/json") {
		return fmt.Errorf("%w: content-type", ErrMockResponse)
	}
	if !isMockEnvelope(body) {
		return fmt.Errorf("%w: envelope", ErrMockResponse)
	}
	if !mockOK {
		return fmt.Errorf("%w: status field", ErrMockResponse)
	}
	if strings.TrimSpace(reqID) == "" {
		return fmt.Errorf("%w: requestID", ErrMockResponse)
	}
	if opName, ok := body["operation"].(string); ok && opName != "" && opName != op {
		return fmt.Errorf("%w: operation mismatch", ErrMockResponse)
	}
	return nil
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
