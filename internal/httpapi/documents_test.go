package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/httpapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/series"
)

const (
	devToken      = "0123456789abcdef0123456789abcdef"
	forbiddenTok  = "fedcba9876543210fedcba9876543210"
	effectiveCode = "EFF-A"
)

func TestCreateDocumentSQLite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "docs.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runDocumentsHTTPSuite(t, persistence.NewStore(sqlDB, persistence.DialectSQLite))
}

func TestCreateDocumentPostgres(t *testing.T) {
	dsn := os.Getenv("FISCAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FISCAL_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runDocumentsHTTPSuite(t, persistence.NewStore(sqlDB, persistence.DialectPostgres))
}

func runDocumentsHTTPSuite(t *testing.T, store *persistence.Store) {
	t.Helper()
	scope := fmt.Sprintf("http-scope-%d", time.Now().UnixNano())
	authenticator, err := auth.NewDevStatic(auth.DevStaticConfig{
		Token:          devToken,
		ScopeID:        scope,
		ForbiddenToken: forbiddenTok,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := series.NewStatic(series.StaticConfig{EffectiveCode: effectiveCode})
	if err != nil {
		t.Fatal(err)
	}
	h := httpapi.WithRequestID(&httpapi.DocumentsHandler{
		Store:  store,
		Auth:   authenticator,
		Series: resolver,
	})

	t.Run("201_and_replay", func(t *testing.T) {
		key := "11111111-1111-4111-8111-111111111111"
		body := minimalBody("ext-http-1", "HACK-SERIES")
		code, first, hdr := doPOST(t, h, key, body, devToken, "application/json")
		if code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", code, first)
		}
		if hdr.Get("X-Request-Id") == "" {
			t.Fatal("missing request id")
		}
		var a httpapi.CreateDocumentResponse
		if err := json.Unmarshal(first, &a); err != nil {
			t.Fatal(err)
		}
		if a.Status != "sealed_locally" || a.ID == "" || a.SubmissionID == "" || a.CreatedAt == "" {
			t.Fatalf("%+v", a)
		}
		if a.ExternalID != "ext-http-1" {
			t.Fatalf("external_id=%q", a.ExternalID)
		}
		var probe map[string]any
		_ = json.Unmarshal(first, &probe)
		if _, ok := probe["fiscal_number"]; ok {
			t.Fatal("fiscal_number must be absent")
		}
		if _, ok := probe["authority_request_id"]; ok {
			t.Fatal("authority_request_id must be absent")
		}

		code2, second, _ := doPOST(t, h, key, body, devToken, "application/json")
		if code2 != http.StatusCreated {
			t.Fatalf("replay status=%d body=%s", code2, second)
		}
		var b httpapi.CreateDocumentResponse
		if err := json.Unmarshal(second, &b); err != nil {
			t.Fatal(err)
		}
		if a != b {
			t.Fatalf("replay mismatch\n%+v\n%+v", a, b)
		}
	})

	t.Run("series_not_controlled_by_requested", func(t *testing.T) {
		key := "33333333-3333-4333-8333-333333333333"
		code, raw, _ := doPOST(t, h, key, minimalBody("ext-http-series", "POS-CHOICE"), devToken, "application/json")
		if code != http.StatusCreated {
			t.Fatalf("%d %s", code, raw)
		}
		r, err := store.SealInTx(context.Background(), persistence.SealRequest{
			IdempotencyKey: "44444444-4444-4444-8444-444444444444",
			SeriesCode:     effectiveCode,
			Intent:         sampleIntent(scope, "ext-http-series-2"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if r.SeriesCode != effectiveCode {
			t.Fatalf("series=%q", r.SeriesCode)
		}
		if r.FiscalSeq < 2 {
			t.Fatalf("expected EFF-A seq advanced after HTTP seal, got %d", r.FiscalSeq)
		}
	})

	t.Run("409_idempotency", func(t *testing.T) {
		key := "55555555-5555-4555-8555-555555555555"
		code, _, _ := doPOST(t, h, key, minimalBody("ext-http-idem", "R"), devToken, "application/json")
		if code != http.StatusCreated {
			t.Fatal(code)
		}
		bad := strings.Replace(minimalBody("ext-http-idem", "R"), `"10.50"`, `"11.00"`, 1)
		code, raw, _ := doPOST(t, h, key, bad, devToken, "application/json")
		assertProblem(t, code, raw, http.StatusConflict, "FISCAL_IDEMPOTENCY_CONFLICT")
	})

	t.Run("409_external_id", func(t *testing.T) {
		code, _, _ := doPOST(t, h, "66666666-6666-4666-8666-666666666666", minimalBody("ext-http-dup", "R"), devToken, "application/json")
		if code != http.StatusCreated {
			t.Fatal(code)
		}
		code, raw, _ := doPOST(t, h, "77777777-7777-4777-8777-777777777777", minimalBody("ext-http-dup", "R"), devToken, "application/json")
		assertProblem(t, code, raw, http.StatusConflict, "FISCAL_EXTERNAL_ID_CONFLICT")
	})

	t.Run("401_www_authenticate", func(t *testing.T) {
		code, raw, hdr := doPOST(t, h, "88888888-8888-4888-8888-888888888888", minimalBody("ext-401", "R"), "wrong-token-wrong-token-wrong!!", "application/json")
		assertProblem(t, code, raw, http.StatusUnauthorized, "FISCAL_UNAUTHORIZED")
		if hdr.Get("WWW-Authenticate") == "" {
			t.Fatal("missing WWW-Authenticate")
		}
	})

	t.Run("403_forbidden_token", func(t *testing.T) {
		code, raw, hdr := doPOST(t, h, "99999999-9999-4999-8999-999999999999", minimalBody("ext-403", "R"), forbiddenTok, "application/json")
		assertProblem(t, code, raw, http.StatusForbidden, "FISCAL_FORBIDDEN")
		if hdr.Get("WWW-Authenticate") != "" {
			t.Fatal("403 must not set WWW-Authenticate")
		}
	})

	t.Run("415_content_type", func(t *testing.T) {
		code, raw, _ := doPOST(t, h, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1", minimalBody("ext-415", "R"), devToken, "text/plain")
		assertProblem(t, code, raw, http.StatusUnsupportedMediaType, "FISCAL_UNSUPPORTED_MEDIA_TYPE")
	})

	t.Run("422_unknown_field_and_trailing", func(t *testing.T) {
		key := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1"
		code, raw, _ := doPOST(t, h, key, `{"external_id":"x","document_type":"invoice","currency":"AOA","issued_at":"2026-07-21T10:00:00Z","seller":{"tax_id":"1","name":"N"},"lines":[{"line_id":"L1","description":"D","quantity":"1","unit_price":"1.00","tax_code":"NOR"}],"extra":1}`,
			devToken, "application/json")
		assertProblem(t, code, raw, http.StatusUnprocessableEntity, "FISCAL_VALIDATION_FAILED")

		trailing := minimalBody("ext-trail", "R") + `{"a":1}`
		code, raw, _ = doPOST(t, h, "cccccccc-cccc-4ccc-8ccc-ccccccccccc1", trailing, devToken, "application/json")
		assertProblem(t, code, raw, http.StatusUnprocessableEntity, "FISCAL_VALIDATION_FAILED")
	})

	t.Run("422_invalid_uuid", func(t *testing.T) {
		code, raw, _ := doPOST(t, h, "not-a-uuid", minimalBody("ext-uuid", "R"), devToken, "application/json")
		assertProblem(t, code, raw, http.StatusUnprocessableEntity, "FISCAL_VALIDATION_FAILED")
	})

	t.Run("422_scope_id_in_body", func(t *testing.T) {
		body := `{"scope_id":"evil","external_id":"ext-scope","document_type":"invoice","currency":"AOA","issued_at":"2026-07-21T10:00:00Z","seller":{"tax_id":"1","name":"N"},"lines":[{"line_id":"L1","description":"D","quantity":"1","unit_price":"1.00","tax_code":"NOR"}]}`
		code, raw, _ := doPOST(t, h, "dddddddd-dddd-4ddd-8ddd-ddddddddddd1", body, devToken, "application/json")
		assertProblem(t, code, raw, http.StatusUnprocessableEntity, "FISCAL_VALIDATION_FAILED")
		if !strings.Contains(string(raw), "scope_id") {
			t.Fatalf("%s", raw)
		}
	})

	t.Run("422_external_id_max_length", func(t *testing.T) {
		longID := strings.Repeat("e", 101)
		body := fmt.Sprintf(`{"external_id":%q,"document_type":"invoice","currency":"AOA","issued_at":"2026-07-21T10:00:00Z","seller":{"tax_id":"1","name":"N"},"lines":[{"line_id":"L1","description":"D","quantity":"1","unit_price":"1.00","tax_code":"NOR"}]}`, longID)
		code, raw, _ := doPOST(t, h, "ffffffff-ffff-4fff-8fff-fffffffffff1", body, devToken, "application/json")
		assertProblem(t, code, raw, http.StatusUnprocessableEntity, "FISCAL_VALIDATION_FAILED")
		if !strings.Contains(string(raw), "external_id") {
			t.Fatalf("%s", raw)
		}
	})

	t.Run("413_body_too_large", func(t *testing.T) {
		big := `{"external_id":"ext-big","document_type":"invoice","currency":"AOA","issued_at":"2026-07-21T10:00:00Z","seller":{"tax_id":"1","name":"N"},"lines":[{"line_id":"L1","description":"` +
			strings.Repeat("X", 1<<20) + `","quantity":"1","unit_price":"1.00","tax_code":"NOR"}]}`
		code, raw, _ := doPOST(t, h, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeee1", big, devToken, "application/json")
		assertProblem(t, code, raw, http.StatusRequestEntityTooLarge, "FISCAL_PAYLOAD_TOO_LARGE")
	})
}

func doPOST(t *testing.T, h http.Handler, idem, body, token, contentType string) (int, []byte, http.Header) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/documents", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Idempotency-Key", idem)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes(), rr.Header()
}

func assertProblem(t *testing.T, code int, raw []byte, wantStatus int, wantCode string) {
	t.Helper()
	if code != wantStatus {
		t.Fatalf("status=%d want %d body=%s", code, wantStatus, raw)
	}
	var p httpapi.Problem
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != wantCode || p.Status != wantStatus || p.RequestID == "" {
		t.Fatalf("%+v", p)
	}
	if !strings.HasPrefix(p.Type, "urn:bwb:fiscal:error:") {
		t.Fatalf("type=%q", p.Type)
	}
}

func minimalBody(externalID, requestedSeries string) string {
	return fmt.Sprintf(`{
  "external_id": %q,
  "document_type": "invoice",
  "currency": "AOA",
  "issued_at": "2026-07-21T10:00:00Z",
  "requested_series": %q,
  "seller": {"tax_id": "0000000000", "name": "Seller Demo"},
  "lines": [{"line_id": "L1", "description": "Item", "quantity": "1", "unit_price": "10.50", "tax_code": "NOR"}]
}`, externalID, requestedSeries)
}

func sampleIntent(scope, external string) canonical.DocumentIntent {
	qty, _ := quantity.ParseCanonical("1")
	price, _ := money.ParseCanonical("1.00")
	return canonical.DocumentIntent{
		ScopeID:      scope,
		ExternalID:   external,
		DocumentType: "invoice",
		Currency:     "AOA",
		IssuedAtUTC:  time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		SellerTaxID:  "0000000000",
		SellerName:   "Seller Demo",
		Lines: []canonical.Line{{
			LineID: "L1", Description: "Item", Quantity: qty, UnitPrice: price, TaxCode: "NOR",
		}},
	}
}
