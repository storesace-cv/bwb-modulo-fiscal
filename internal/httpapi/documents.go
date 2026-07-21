// Package httpapi implementa os handlers HTTP públicos do módulo fiscal.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/series"
)

const (
	maxBodyBytes          = 1 << 20 // 1 MiB
	maxCorrelationIDLen   = 128
	headerRequestID       = "X-Request-Id"
	headerCorrelationID   = "X-Correlation-Id"
	statusSealedLocally   = "sealed_locally"
	wwwAuthenticateBearer = `Bearer realm="fiscal-api"`
)

var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

// DocumentsHandler serves POST /v1/documents.
type DocumentsHandler struct {
	Store  *persistence.Store
	Auth   auth.Authenticator
	Series series.Resolver
	Log    *slog.Logger
}

// CreateDocumentResponse matches OpenAPI CreateDocumentResponse.
type CreateDocumentResponse struct {
	ID           string `json:"id"`
	ExternalID   string `json:"external_id"`
	Status       string `json:"status"`
	SubmissionID string `json:"submission_id"`
	CreatedAt    string `json:"created_at"`
}

type documentIntentBody struct {
	ExternalID      string         `json:"external_id"`
	DocumentType    string         `json:"document_type"`
	Currency        string         `json:"currency"`
	IssuedAt        string         `json:"issued_at"`
	RequestedSeries string         `json:"requested_series"`
	Seller          *partyBody     `json:"seller"`
	Customer        *partyBody     `json:"customer"`
	Lines           []documentLine `json:"lines"`
	ScopeID         *string        `json:"scope_id"` // rejected if present
}

type partyBody struct {
	TaxID string `json:"tax_id"`
	Name  string `json:"name"`
}

type documentLine struct {
	LineID      string `json:"line_id"`
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	TaxCode     string `json:"tax_code"`
}

type fieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Problem is the stable error body.
type Problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Code      string       `json:"code"`
	RequestID string       `json:"request_id"`
	Errors    []fieldError `json:"errors,omitempty"`
}

type ctxKey int

const requestIDKey ctxKey = 1

// WithRequestID middleware generates request_id and optionally accepts a limited correlation ID.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := newRequestID()
		w.Header().Set(headerRequestID, reqID)
		if corr := strings.TrimSpace(r.Header.Get(headerCorrelationID)); corr != "" {
			if len(corr) <= maxCorrelationIDLen && correlationIDPattern.MatchString(corr) {
				w.Header().Set(headerCorrelationID, corr)
			}
			// Invalid correlation IDs are ignored (not echoed); request continues.
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, reqID)))
	})
}

func requestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
		return v
	}
	return newRequestID()
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "req_" + hex.EncodeToString(b[:])
}

func (h *DocumentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.writeProblem(w, r, http.StatusMethodNotAllowed, "urn:bwb:fiscal:error:method-not-allowed", "Método não permitido", "FISCAL_METHOD_NOT_ALLOWED", nil)
		return
	}
	reqID := requestIDFrom(r.Context())

	principal, err := h.Auth.Authenticate(r.Context(), r)
	if errors.Is(err, auth.ErrForbidden) {
		h.writeProblem(w, r, http.StatusForbidden, "urn:bwb:fiscal:error:forbidden", "Não autorizado", "FISCAL_FORBIDDEN", nil)
		return
	}
	if err != nil {
		w.Header().Set("WWW-Authenticate", wwwAuthenticateBearer)
		h.writeProblem(w, r, http.StatusUnauthorized, "urn:bwb:fiscal:error:unauthorized", "Não autenticado", "FISCAL_UNAUTHORIZED", nil)
		return
	}

	if err := requireJSONContentType(r.Header.Get("Content-Type")); err != nil {
		h.writeProblem(w, r, http.StatusUnsupportedMediaType, "urn:bwb:fiscal:error:unsupported-media-type", "Tipo de média não suportado", "FISCAL_UNSUPPORTED_MEDIA_TYPE", nil)
		return
	}

	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if _, err := uuid.Parse(idemKey); err != nil {
		h.writeProblem(w, r, http.StatusUnprocessableEntity, "urn:bwb:fiscal:error:validation", "Documento inválido", "FISCAL_VALIDATION_FAILED",
			[]fieldError{{Field: "Idempotency-Key", Code: "INVALID_UUID", Message: "deve ser UUID"}})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := decodeDocumentIntent(r.Body)
	if errors.Is(err, errPayloadTooLarge) {
		h.writeProblem(w, r, http.StatusRequestEntityTooLarge, "urn:bwb:fiscal:error:payload-too-large", "Corpo demasiado grande", "FISCAL_PAYLOAD_TOO_LARGE", nil)
		return
	}
	var verr *validationFailure
	if errors.As(err, &verr) {
		h.writeProblem(w, r, http.StatusUnprocessableEntity, "urn:bwb:fiscal:error:validation", "Documento inválido", "FISCAL_VALIDATION_FAILED", verr.Errors)
		return
	}
	if err != nil {
		h.writeProblem(w, r, http.StatusUnprocessableEntity, "urn:bwb:fiscal:error:validation", "Documento inválido", "FISCAL_VALIDATION_FAILED",
			[]fieldError{{Field: "body", Code: "INVALID_JSON", Message: "JSON inválido"}})
		return
	}

	seriesCode, err := h.Series.Resolve(principal.ScopeID, body.RequestedSeries)
	if err != nil {
		h.writeProblem(w, r, http.StatusUnprocessableEntity, "urn:bwb:fiscal:error:validation", "Documento inválido", "FISCAL_VALIDATION_FAILED",
			[]fieldError{{Field: "requested_series", Code: "UNRESOLVED", Message: "referência de série não autorizada"}})
		return
	}

	intent, ferrs := mapIntent(principal.ScopeID, body)
	if len(ferrs) > 0 {
		h.writeProblem(w, r, http.StatusUnprocessableEntity, "urn:bwb:fiscal:error:validation", "Documento inválido", "FISCAL_VALIDATION_FAILED", ferrs)
		return
	}

	res, err := h.Store.SealInTx(r.Context(), persistence.SealRequest{
		IdempotencyKey: idemKey,
		SeriesCode:     seriesCode,
		Intent:         intent,
	})
	if errors.Is(err, persistence.ErrIdempotencyConflict) {
		h.writeProblem(w, r, http.StatusConflict, "urn:bwb:fiscal:error:idempotency-conflict", "Conflito de idempotência", "FISCAL_IDEMPOTENCY_CONFLICT", nil)
		return
	}
	if errors.Is(err, persistence.ErrExternalIDConflict) {
		h.writeProblem(w, r, http.StatusConflict, "urn:bwb:fiscal:error:external-id-conflict", "Conflito de identificador externo", "FISCAL_EXTERNAL_ID_CONFLICT", nil)
		return
	}
	var pverr *persistence.ValidationError
	if errors.As(err, &pverr) {
		h.writeProblem(w, r, http.StatusUnprocessableEntity, "urn:bwb:fiscal:error:validation", "Documento inválido", "FISCAL_VALIDATION_FAILED",
			[]fieldError{{Field: pverr.Field, Code: pverr.Code, Message: pverr.Message}})
		return
	}
	if err != nil {
		if h.Log != nil {
			h.Log.Error("seal_failed", "request_id", reqID, "error", "internal")
		}
		h.writeProblem(w, r, http.StatusInternalServerError, "urn:bwb:fiscal:error:internal", "Erro interno", "FISCAL_INTERNAL_ERROR", nil)
		return
	}

	out := CreateDocumentResponse{
		ID:           res.DocumentID,
		ExternalID:   res.ExternalID,
		Status:       statusSealedLocally,
		SubmissionID: res.SubmissionID,
		CreatedAt:    res.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func requireJSONContentType(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errUnsupportedMedia
	}
	mediatype, params, err := mime.ParseMediaType(raw)
	if err != nil {
		return errUnsupportedMedia
	}
	if !strings.EqualFold(mediatype, "application/json") {
		return errUnsupportedMedia
	}
	if cs, ok := params["charset"]; ok {
		cs = strings.ToLower(strings.TrimSpace(cs))
		if cs != "" && cs != "utf-8" && cs != "utf8" {
			return errUnsupportedMedia
		}
	}
	return nil
}

var (
	errUnsupportedMedia = errors.New("unsupported media type")
	errPayloadTooLarge  = errors.New("payload too large")
)

type validationFailure struct {
	Errors []fieldError
}

func (e *validationFailure) Error() string { return "validation failed" }

func decodeDocumentIntent(r io.Reader) (documentIntentBody, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var body documentIntentBody
	if err := dec.Decode(&body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return documentIntentBody{}, errPayloadTooLarge
		}
		return documentIntentBody{}, err
	}
	if dec.More() {
		return documentIntentBody{}, &validationFailure{Errors: []fieldError{{
			Field: "body", Code: "TRAILING_DATA", Message: "conteúdo JSON adicional não permitido",
		}}}
	}
	if body.ScopeID != nil {
		return documentIntentBody{}, &validationFailure{Errors: []fieldError{{
			Field: "scope_id", Code: "FORBIDDEN_FIELD", Message: "scope_id não é aceite no body",
		}}}
	}
	return body, nil
}

func mapIntent(scopeID string, body documentIntentBody) (canonical.DocumentIntent, []fieldError) {
	var errs []fieldError
	requireNonEmpty := func(field, v string) {
		if strings.TrimSpace(v) == "" {
			errs = append(errs, fieldError{Field: field, Code: "REQUIRED", Message: "obrigatório e non-empty"})
		}
	}
	requireNonEmpty("external_id", body.ExternalID)
	if body.DocumentType != "invoice" && body.DocumentType != "credit_note" {
		errs = append(errs, fieldError{Field: "document_type", Code: "INVALID_ENUM", Message: "valor não permitido"})
	}
	if body.Currency != "AOA" {
		errs = append(errs, fieldError{Field: "currency", Code: "INVALID_ENUM", Message: "valor não permitido"})
	}
	requireNonEmpty("issued_at", body.IssuedAt)
	if body.Seller == nil {
		errs = append(errs, fieldError{Field: "seller", Code: "REQUIRED", Message: "obrigatório"})
	} else {
		requireNonEmpty("seller.tax_id", body.Seller.TaxID)
		requireNonEmpty("seller.name", body.Seller.Name)
	}
	if len(body.Lines) == 0 {
		errs = append(errs, fieldError{Field: "lines", Code: "REQUIRED", Message: "pelo menos uma linha"})
	}

	intent := canonical.DocumentIntent{
		ScopeID:         scopeID,
		ExternalID:      body.ExternalID,
		DocumentType:    body.DocumentType,
		Currency:        body.Currency,
		IssuedAtUTC:     body.IssuedAt,
		RequestedSeries: body.RequestedSeries,
	}
	if body.Seller != nil {
		intent.SellerTaxID = body.Seller.TaxID
		intent.SellerName = body.Seller.Name
	}
	if body.Customer != nil {
		intent.CustomerTaxID = body.Customer.TaxID
		intent.CustomerName = body.Customer.Name
	}

	for i, ln := range body.Lines {
		prefix := "lines[" + itoa(i) + "]"
		requireNonEmpty(prefix+".line_id", ln.LineID)
		requireNonEmpty(prefix+".description", ln.Description)
		requireNonEmpty(prefix+".tax_code", ln.TaxCode)
		qty, err := quantity.ParseCanonical(ln.Quantity)
		if err != nil {
			errs = append(errs, fieldError{Field: prefix + ".quantity", Code: "INVALID_FORMAT", Message: "quantidade inválida"})
			continue
		}
		price, err := money.ParseCanonical(ln.UnitPrice)
		if err != nil {
			errs = append(errs, fieldError{Field: prefix + ".unit_price", Code: "INVALID_FORMAT", Message: "preço inválido"})
			continue
		}
		intent.Lines = append(intent.Lines, canonical.Line{
			LineID:      ln.LineID,
			Description: ln.Description,
			Quantity:    qty,
			UnitPrice:   price,
			TaxCode:     ln.TaxCode,
		})
	}
	if len(errs) > 0 {
		return canonical.DocumentIntent{}, errs
	}
	return intent, nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func (h *DocumentsHandler) writeProblem(w http.ResponseWriter, r *http.Request, status int, typ, title, code string, errs []fieldError) {
	reqID := requestIDFrom(r.Context())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	p := Problem{
		Type:      typ,
		Title:     title,
		Status:    status,
		Code:      code,
		RequestID: reqID,
		Errors:    errs,
	}
	_ = json.NewEncoder(w).Encode(p)
}
