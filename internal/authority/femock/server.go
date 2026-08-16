package femock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

// Config constructs a local mock FE handler. Credentials must be synthetic and injected.
type Config struct {
	Username string
	Password string
	Provider agttestkit.IdentityProvider
	MaxBody  int64
	// InjectedDelay is waited (respecting request context) before processing — tests only.
	InjectedDelay time.Duration
}

// Server is an in-memory mock FE AGT HTTP handler (httptest / Handler only).
type Server struct {
	cfg      Config
	mu       sync.Mutex
	closed   bool
	replay   *replayStore
	scripted map[string]string // operation → FE-RNG code for next successful-auth call
	logCodes []string          // allowlisted codes observed (tests)
}

// New validates config and returns a Server. Does not listen on a network port.
func New(cfg Config) (*Server, error) {
	if cfg.Username == "" || cfg.Password == "" {
		return nil, errBadConfig("credentials required")
	}
	if cfg.Provider == nil {
		return nil, errBadConfig("provider required")
	}
	if cfg.MaxBody <= 0 {
		cfg.MaxBody = DefaultMaxBody
	}
	return &Server{
		cfg:      cfg,
		replay:   newReplayStore(),
		scripted: make(map[string]string),
	}, nil
}

func errBadConfig(msg string) error {
	return &configError{msg: msg}
}

type configError struct{ msg string }

func (e *configError) Error() string { return "femock: " + e.msg }

// ScriptFERNG queues a simulated FE-RNG response for the next call to op.
// code must be allowlisted; unknown codes are rejected (harness misconfig).
func (s *Server) ScriptFERNG(op, code string) error {
	if _, err := ferngSourceID(code); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errClosed()
	}
	s.scripted[op] = code
	return nil
}

func errClosed() error { return &configError{msg: CodeClosed} }

// Close clears in-memory state. Safe for tests.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.replay.reset()
	s.scripted = make(map[string]string)
	s.logCodes = nil
	return nil
}

// LogCodes returns a copy of allowlisted error/success codes recorded (no secrets).
func (s *Server) LogCodes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logCodes))
	copy(out, s.logCodes)
	return out
}

func (s *Server) record(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logCodes = append(s.logCodes, code)
}

// Handler returns http.Handler for PathPrefix routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(PathPrefix+"/softwareInfo", s.wrap("softwareInfo", s.handleSoftwareInfo))
	mux.HandleFunc(PathPrefix+"/obterEstado", s.wrap("obterEstado", s.handleObterEstado))
	mux.HandleFunc(PathPrefix+"/consultarFactura", s.wrap("consultarFactura", s.handleConsultarFactura))
	// Blocked wire operations — auth required, then profile blocked.
	for _, op := range []struct {
		path     string
		conflict string
	}{
		{"/registarFactura", "C-FE-JWS-DOC-001"},
		{"/solicitarSerie", "C-FE-JWS-REQ-001"},
		{"/listarSeries", "C-FE-JWS-REQ-002"},
		{"/validarDocumento", "C-FE-JWS-REQ-003"},
		{"/listarFacturas", "C-FE-JWS-REQ-004"},
	} {
		op := op
		mux.HandleFunc(PathPrefix+op.path, s.wrap(strings.TrimPrefix(op.path, "/"), func(w http.ResponseWriter, r *http.Request, reqID string) {
			s.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"simulated": true,
				"mock":      TypMock,
				"requestID": reqID,
				"code":      CodeProfileBlocked,
				"conflict":  op.conflict,
				"note":      "BWB-MOCK≠JWT/JOSE AGT; profile wire blocked",
			})
			s.record(CodeProfileBlocked)
		}))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.writeErr(w, http.StatusNotFound, newMockID(), CodeNotFound)
	})
	return mux
}

type opHandler func(w http.ResponseWriter, r *http.Request, reqID string)

type wireRequest struct {
	IdentityRef     string `json:"identityRef"`
	JWS             string `json:"jws"`
	IdempotencyKey  string `json:"idempotencyKey"`
	ClientRequestID string `json:"clientRequestID"` // ignored for canonical requestID
}

func (s *Server) wrap(op string, h opHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := newMockID()
		s.mu.Lock()
		closed := s.closed
		delay := s.cfg.InjectedDelay
		s.mu.Unlock()
		if closed {
			s.writeErr(w, http.StatusServiceUnavailable, reqID, CodeClosed)
			return
		}
		if r.Method != http.MethodPost {
			s.writeErr(w, http.StatusMethodNotAllowed, reqID, CodeMethodNotAllowed)
			return
		}
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
			s.writeErr(w, http.StatusUnsupportedMediaType, reqID, CodeContentType)
			return
		}
		if !checkBasicAuth(r, s.cfg.Username, s.cfg.Password) {
			s.writeErr(w, http.StatusUnauthorized, reqID, CodeUnauthorized)
			return
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-r.Context().Done():
				s.writeErr(w, http.StatusRequestTimeout, reqID, CodeCancelled)
				return
			case <-timer.C:
			}
		} else if err := r.Context().Err(); err != nil {
			s.writeErr(w, http.StatusRequestTimeout, reqID, CodeCancelled)
			return
		}
		_ = op
		h(w, r, reqID)
	}
}

func (s *Server) readWire(w http.ResponseWriter, r *http.Request, reqID string) (wireRequest, bool) {
	limited := io.LimitReader(r.Body, s.cfg.MaxBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return wireRequest{}, false
	}
	if int64(len(body)) > s.cfg.MaxBody {
		s.writeErr(w, http.StatusRequestEntityTooLarge, reqID, CodeBodyTooLarge)
		return wireRequest{}, false
	}
	var wr wireRequest
	if err := decodeStrictJSON(body, &wr); err != nil {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return wireRequest{}, false
	}
	if wr.IdentityRef == "" || wr.JWS == "" || wr.IdempotencyKey == "" {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return wireRequest{}, false
	}
	return wr, true
}

func (s *Server) handleReplay(w http.ResponseWriter, op, key, jws, reqID string) (done bool) {
	hash := payloadHash(op, jws)
	replayKey := op + ":" + key
	if resp, status, conflict, ok := s.replay.lookup(replayKey, hash); ok {
		if conflict {
			s.writeJSON(w, http.StatusConflict, map[string]any{
				"simulated": true,
				"mock":      TypMock,
				"requestID": reqID,
				"code":      CodeIdempotencyConflict,
			})
			s.record(CodeIdempotencyConflict)
			return true
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(resp)
		return true
	}
	return false
}

func (s *Server) storeReplay(op, key, jws string, status int, payload []byte) {
	s.replay.store(op+":"+key, payloadHash(op, jws), status, payload)
}

func (s *Server) takeScript(op string) (code, sourceID string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok = s.scripted[op]
	if !ok {
		return "", "", false
	}
	delete(s.scripted, op)
	sourceID, _ = ferngSourceID(code)
	return code, sourceID, true
}

func (s *Server) handleSoftwareInfo(w http.ResponseWriter, r *http.Request, reqID string) {
	wr, ok := s.readWire(w, r, reqID)
	if !ok {
		return
	}
	if s.handleReplay(w, "softwareInfo", wr.IdempotencyKey, wr.JWS, reqID) {
		return
	}
	payload, _, err := verifyMockJWS(s.cfg.Provider, wr.IdentityRef, wr.JWS)
	if err != nil {
		s.writeErr(w, http.StatusUnauthorized, reqID, err.Error())
		return
	}
	if err := s.cfg.Provider.RequireRole(wr.IdentityRef, agttestkit.RoleProducerEphemeral); err != nil {
		s.writeErr(w, http.StatusForbidden, reqID, CodeRoleMismatch)
		return
	}
	var claims feprofile.SoftwareInfoClaims
	if err := decodeStrictJSON(payload, &claims); err != nil || claims.ProductID == "" {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return
	}
	// Ensure payload matches builder canonical form.
	canon, err := feprofile.MarshalSoftwareInfoPayload(claims)
	if err != nil || !bytesEqual(canon, payload) {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return
	}
	s.respondOKOrFERNG(w, r.Context(), "softwareInfo", wr, reqID, map[string]any{
		"operation": "softwareInfo",
		"productId": claims.ProductID,
	})
}

func (s *Server) handleObterEstado(w http.ResponseWriter, r *http.Request, reqID string) {
	wr, ok := s.readWire(w, r, reqID)
	if !ok {
		return
	}
	if s.handleReplay(w, "obterEstado", wr.IdempotencyKey, wr.JWS, reqID) {
		return
	}
	payload, _, err := verifyMockJWS(s.cfg.Provider, wr.IdentityRef, wr.JWS)
	if err != nil {
		s.writeErr(w, http.StatusUnauthorized, reqID, err.Error())
		return
	}
	if err := s.cfg.Provider.RequireRole(wr.IdentityRef, agttestkit.RoleTaxpayerTest); err != nil {
		s.writeErr(w, http.StatusForbidden, reqID, CodeRoleMismatch)
		return
	}
	var claims feprofile.ObterEstadoRequestClaims
	if err := decodeStrictJSON(payload, &claims); err != nil {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return
	}
	if err := s.cfg.Provider.ValidateTaxpayerBinding(wr.IdentityRef, claims.TaxRegistrationNumber); err != nil {
		s.writeErr(w, http.StatusForbidden, reqID, CodeBindingMismatch)
		return
	}
	canon, err := feprofile.MarshalObterEstadoRequestPayload(claims)
	if err != nil || !bytesEqual(canon, payload) {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return
	}
	s.respondOKOrFERNG(w, r.Context(), "obterEstado", wr, reqID, map[string]any{
		"operation": "obterEstado",
		"state":     "SIMULATED_PENDING",
	})
}

func (s *Server) handleConsultarFactura(w http.ResponseWriter, r *http.Request, reqID string) {
	wr, ok := s.readWire(w, r, reqID)
	if !ok {
		return
	}
	if s.handleReplay(w, "consultarFactura", wr.IdempotencyKey, wr.JWS, reqID) {
		return
	}
	payload, _, err := verifyMockJWS(s.cfg.Provider, wr.IdentityRef, wr.JWS)
	if err != nil {
		s.writeErr(w, http.StatusUnauthorized, reqID, err.Error())
		return
	}
	if err := s.cfg.Provider.RequireRole(wr.IdentityRef, agttestkit.RoleTaxpayerTest); err != nil {
		s.writeErr(w, http.StatusForbidden, reqID, CodeRoleMismatch)
		return
	}
	var claims feprofile.ConsultarFacturaRequestClaims
	if err := decodeStrictJSON(payload, &claims); err != nil {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return
	}
	if err := s.cfg.Provider.ValidateTaxpayerBinding(wr.IdentityRef, claims.TaxRegistrationNumber); err != nil {
		s.writeErr(w, http.StatusForbidden, reqID, CodeBindingMismatch)
		return
	}
	canon, err := feprofile.MarshalConsultarFacturaRequestPayload(claims)
	if err != nil || !bytesEqual(canon, payload) {
		s.writeErr(w, http.StatusBadRequest, reqID, CodeBadRequest)
		return
	}
	s.respondOKOrFERNG(w, r.Context(), "consultarFactura", wr, reqID, map[string]any{
		"operation":  "consultarFactura",
		"documentNo": "SIMULATED",
	})
}

func (s *Server) respondOKOrFERNG(w http.ResponseWriter, ctx context.Context, op string, wr wireRequest, reqID string, extra map[string]any) {
	if err := ctx.Err(); err != nil {
		s.writeErr(w, http.StatusRequestTimeout, reqID, CodeCancelled)
		return
	}
	if code, src, ok := s.takeScript(op); ok {
		body := map[string]any{
			"simulated": true,
			"mock":      TypMock,
			"requestID": reqID,
			"code":      code,
			"source_id": src,
			"kind":      "FE-RNG-simulated",
			"note":      "simulated FE-RNG ≠ AGT live response",
		}
		raw, _ := json.Marshal(body)
		s.storeReplay(op, wr.IdempotencyKey, wr.JWS, http.StatusUnprocessableEntity, raw)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write(raw)
		s.record(code)
		return
	}
	body := map[string]any{
		"simulated": true,
		"mock":      TypMock,
		"requestID": reqID,
		"status":    "ok",
		"note":      "BWB-MOCK success ≠ AGT homologation",
	}
	for k, v := range extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	s.storeReplay(op, wr.IdempotencyKey, wr.JWS, http.StatusOK, raw)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	s.record("ok")
}

func (s *Server) writeErr(w http.ResponseWriter, status int, reqID, code string) {
	s.writeJSON(w, status, map[string]any{
		"simulated": true,
		"mock":      TypMock,
		"requestID": reqID,
		"code":      code,
	})
	s.record(code)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newMockID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "mock-" + hex.EncodeToString(b[:])
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
