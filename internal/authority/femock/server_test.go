package femock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/femock"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fejws"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

const (
	mockUser = "bwb-mock-user-synth"
	mockPass = "bwb-mock-pass-synth"
)

func newMock(t *testing.T) (*femock.Server, agttestkit.IdentityProvider, *httptest.Server, func()) {
	t.Helper()
	producer, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	taxpayer, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		cleanupWB()
		t.Fatal(err)
	}
	// Prefer taxpayer provider for most routes; software uses producer via separate helpers.
	_ = producer
	srv, err := femock.New(femock.Config{
		Username: mockUser,
		Password: mockPass,
		Provider: taxpayer,
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	cleanup := func() {
		ts.Close()
		_ = srv.Close()
		_ = taxpayer.Close()
		_ = producer.Close()
		cleanupWB()
	}
	return srv, taxpayer, ts, cleanup
}

func basicReq(t *testing.T, method, url, user, pass string, body any) *http.Request {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestBasicAuth(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	jws, err := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "SYNTHETIC_REQ",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{"identityRef": ref, "jws": jws, "idempotencyKey": "k1"}

	res, err := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", "", "", body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
	assertSanitized(t, res)

	res2, err := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, "wrong", body))
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res2.StatusCode)
	}
	assertSanitized(t, res2)

	res3, err := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res3.Body)
		t.Fatalf("status %d %s", res3.StatusCode, b)
	}
}

func TestBWBMockJWSAcceptedRejectsJWTJOSEOmit(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	claims := feprofile.ObterEstadoRequestClaims{TaxRegistrationNumber: "9100000000", RequestID: "R1"}
	payload, err := feprofile.MarshalObterEstadoRequestPayload(claims)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := provider.Signer(ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []string{"JWT", "JOSE", ""} {
		hdr := fejws.ProtectedHeader{Alg: fejws.Algorithm}
		if typ != "" {
			hdr.Typ = typ
		}
		jws, err := fejws.SignCompact(signer, payload, hdr)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
			"identityRef": ref, "jws": jws, "idempotencyKey": "typ-" + typ,
		}))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			t.Fatalf("typ %q must fail: %s", typ, b)
		}
		if !strings.Contains(string(b), femock.CodeJWSTypRejected) && !strings.Contains(string(b), femock.CodeJWSInvalid) {
			t.Fatalf("typ %q: %s", typ, b)
		}
		assertBodySanitized(t, b)
	}
	okJWS, err := femock.SignObterEstadoMock(provider, ref, claims)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": okJWS, "idempotencyKey": "ok-typ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%d", res.StatusCode)
	}
}

func TestTamperAndRoleBinding(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	jws, err := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Tamper payload segment.
	parts := strings.Split(jws, ".")
	parts[1] = parts[1] + "x"
	bad := strings.Join(parts, ".")
	res, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": bad, "idempotencyKey": "tamper",
	}))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode == http.StatusOK || strings.Contains(string(b), "9100000000") {
		t.Fatalf("%s", b)
	}

	// Wrong NIF binding via crafted claims signed... cannot sign wrong NIF with binding in SignObterEstadoMock.
	// Sign with wrong NIF directly:
	signer, _ := provider.Signer(ref)
	payload, _ := json.Marshal(feprofile.ObterEstadoRequestClaims{TaxRegistrationNumber: "9100000999", RequestID: "R"})
	wrong, err := fejws.SignCompact(signer, payload, fejws.ProtectedHeader{Alg: fejws.Algorithm, Typ: femock.TypMock})
	if err != nil {
		t.Fatal(err)
	}
	res2, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": wrong, "idempotencyKey": "bind",
	}))
	b2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	if res2.StatusCode == http.StatusOK {
		t.Fatal("binding should fail")
	}
	if strings.Contains(string(b2), "9100000999") || strings.Contains(string(b2), "9100000000") {
		t.Fatalf("nif leak: %s", b2)
	}

	// Role: producer on obterEstado
	prod, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer prod.Close()
	pref := prod.List()[0].Ref
	// Rebuild server with producer? Use taxpayer server but sign with producer ref unknown → JWS invalid
	// Mount dual: create server with producer for software role test on obterEstado.
	srv2, err := femock.New(femock.Config{Username: mockUser, Password: mockPass, Provider: prod})
	if err != nil {
		t.Fatal(err)
	}
	defer srv2.Close()
	ts2 := httptest.NewServer(srv2.Handler())
	defer ts2.Close()
	psigner, err := prod.Signer(pref)
	if err != nil {
		t.Fatal(err)
	}
	pjws, err := fejws.SignCompact(psigner, mustMarshalOE(t, "9100000000"), fejws.ProtectedHeader{Alg: fejws.Algorithm, Typ: femock.TypMock})
	if err != nil {
		t.Fatal(err)
	}
	res3, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts2.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
		"identityRef": pref, "jws": pjws, "idempotencyKey": "role",
	}))
	b3, _ := io.ReadAll(res3.Body)
	res3.Body.Close()
	if !strings.Contains(string(b3), femock.CodeRoleMismatch) {
		t.Fatalf("%s", b3)
	}
}

func mustMarshalOE(t *testing.T, nif string) []byte {
	t.Helper()
	b, err := feprofile.MarshalObterEstadoRequestPayload(feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: nif, RequestID: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSupportedOperationsAndBlocked(t *testing.T) {
	srv, taxpayer, ts, cleanup := newMock(t)
	defer cleanup()
	prod, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer prod.Close()

	// softwareInfo needs producer provider on server
	srvProd, err := femock.New(femock.Config{Username: mockUser, Password: mockPass, Provider: prod})
	if err != nil {
		t.Fatal(err)
	}
	defer srvProd.Close()
	tsProd := httptest.NewServer(srvProd.Handler())
	defer tsProd.Close()

	pref := prod.List()[0].Ref
	sjws, err := femock.SignSoftwareMock(prod, pref, feprofile.SoftwareInfoClaims{
		ProductID: "SYNTHETIC_PRODUCT", ProductVersion: "0.0.1-test", SoftwareValidationNumber: "SYNTHETIC_SVN_PLACEHOLDER",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, tsProd.URL+femock.PathPrefix+"/softwareInfo", mockUser, mockPass, map[string]string{
		"identityRef": pref, "jws": sjws, "idempotencyKey": "sw1",
	}))
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("%d %s", res.StatusCode, b)
	}
	res.Body.Close()

	ref := taxpayer.List()[0].Ref
	cjws, err := femock.SignConsultarFacturaMock(taxpayer, ref, feprofile.ConsultarFacturaRequestClaims{
		TaxRegistrationNumber: "9100000000", DocumentNo: "FT SYNTHETIC/1",
	})
	if err != nil {
		t.Fatal(err)
	}
	res2, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/consultarFactura", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": cjws, "idempotencyKey": "cf1", "clientRequestID": "CLIENT-MUST-NOT-ECHO",
	}))
	b2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", res2.StatusCode, b2)
	}
	if strings.Contains(string(b2), "CLIENT-MUST-NOT-ECHO") {
		t.Fatal("client request id reflected")
	}

	for _, path := range []string{"/registarFactura", "/solicitarSerie", "/listarSeries", "/validarDocumento", "/listarFacturas"} {
		res, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+path, mockUser, mockPass, map[string]string{
			"identityRef": ref, "jws": "x", "idempotencyKey": "b",
		}))
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if !strings.Contains(string(b), femock.CodeProfileBlocked) {
			t.Fatalf("%s: %s", path, b)
		}
		assertBodySanitized(t, b)
	}
	_ = srv
}

func TestFERNGContextualCatalog(t *testing.T) {
	srv, provider, ts, cleanup := newMock(t)
	defer cleanup()

	if err := srv.ScriptFERNG("obterEstado", "FE-RNG-031"); err == nil {
		t.Fatal("FE-RNG-031 must not script on obterEstado")
	}
	if err := srv.ScriptFERNG("consultarFactura", "FE-RNG-051"); err == nil {
		t.Fatal("FE-RNG-051 must not script on consultarFactura")
	}
	if err := srv.ScriptFERNG("registarFactura", "FE-RNG-031"); err == nil {
		t.Fatal("blocked route must not script FE-RNG emit")
	}
	if err := srv.ScriptFERNG("noSuchOp", "FE-RNG-032"); err == nil {
		t.Fatal("unknown op")
	}
	if err := srv.ScriptFERNG("obterEstado", "FE-RNG-032"); err != nil {
		t.Fatal(err)
	}

	ref := provider.List()[0].Ref
	jws, err := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": jws, "idempotencyKey": "ferng-ok",
	}))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("%d %s", res.StatusCode, b)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["code"] != "FE-RNG-032" {
		t.Fatalf("%v", got)
	}
	if got["source_id"] != "AO-FE-SNAP-HML-2026-07-25-CONSULTAR" {
		t.Fatalf("source_id=%v", got["source_id"])
	}

	// consultarFactura valid code
	if err := srv.ScriptFERNG("consultarFactura", "FE-RNG-010"); err != nil {
		t.Fatal(err)
	}
	cjws, err := femock.SignConsultarFacturaMock(provider, ref, feprofile.ConsultarFacturaRequestClaims{
		TaxRegistrationNumber: "9100000000", DocumentNo: "FT SYNTHETIC/1",
	})
	if err != nil {
		t.Fatal(err)
	}
	res2, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/consultarFactura", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": cjws, "idempotencyKey": "ferng-cf",
	}))
	b2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	var got2 map[string]any
	_ = json.Unmarshal(b2, &got2)
	if got2["source_id"] != "AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA" {
		t.Fatalf("%v", got2)
	}

	// Blocked HTTP route never emits FE-RNG after validation of request
	res3, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/registarFactura", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": "x", "idempotencyKey": "b",
	}))
	b3, _ := io.ReadAll(res3.Body)
	res3.Body.Close()
	if !strings.Contains(string(b3), femock.CodeProfileBlocked) || strings.Contains(string(b3), "FE-RNG-") {
		t.Fatalf("%s", b3)
	}
}

func TestReplayReautenticatesAndNewRequestID(t *testing.T) {
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()
	provider, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := femock.New(femock.Config{Username: mockUser, Password: mockPass, Provider: provider})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ref := provider.List()[0].Ref
	jws, err := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R1",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{"identityRef": ref, "jws": jws, "idempotencyKey": "same", "clientRequestID": "CLIENT-ID-X"}
	res1, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	b1, _ := io.ReadAll(res1.Body)
	res1.Body.Close()
	var m1 map[string]any
	_ = json.Unmarshal(b1, &m1)
	id1, _ := m1["requestID"].(string)
	if id1 == "" || strings.Contains(string(b1), "CLIENT-ID-X") {
		t.Fatalf("%s", b1)
	}

	res2, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	b2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	var m2 map[string]any
	_ = json.Unmarshal(b2, &m2)
	id2, _ := m2["requestID"].(string)
	if id2 == "" || id2 == id1 {
		t.Fatalf("requestID must change on replay: %q vs %q", id1, id2)
	}
	if m1["state"] != m2["state"] || m1["status"] != m2["status"] {
		t.Fatalf("functional mismatch %v vs %v", m1, m2)
	}

	// Conflict with different JWS
	jws2, _ := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R2",
	})
	body["jws"] = jws2
	res3, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	b3, _ := io.ReadAll(res3.Body)
	res3.Body.Close()
	if !strings.Contains(string(b3), femock.CodeIdempotencyConflict) {
		t.Fatalf("%s", b3)
	}

	// Provider closed → replay must not return cache
	body["jws"] = jws
	_ = provider.Close()
	res4, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	b4, _ := io.ReadAll(res4.Body)
	res4.Body.Close()
	if res4.StatusCode == http.StatusOK || strings.Contains(string(b4), `"status":"ok"`) {
		t.Fatalf("closed provider must not replay cache: %s", b4)
	}
	_ = srv.Close()
}

func TestContentTypeStrict(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	jws, _ := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	url := ts.URL + femock.PathPrefix + "/obterEstado"
	payload := map[string]string{"identityRef": ref, "jws": jws, "idempotencyKey": "ct"}

	// empty Content-Type
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.SetBasicAuth(mockUser, mockPass)
	res, _ := http.DefaultClient.Do(req)
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("empty CT: %d", res.StatusCode)
	}

	for _, ct := range []string{"application/jsonx", "text/json", "application/json; charset=latin1", "not-a-type"} {
		req := basicReq(t, http.MethodPost, url, mockUser, mockPass, payload)
		req.Header.Set("Content-Type", ct)
		res, _ := http.DefaultClient.Do(req)
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusUnsupportedMediaType {
			t.Fatalf("%s → %d", ct, res.StatusCode)
		}
	}

	req = basicReq(t, http.MethodPost, url, mockUser, mockPass, payload)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(res.Body)
		t.Fatalf("charset utf-8: %d %s", res.StatusCode, bb)
	}
	res.Body.Close()
}

func TestMethodBodyLimitCancelClose(t *testing.T) {
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()
	provider, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	srv, err := femock.New(femock.Config{
		Username: mockUser, Password: mockPass, Provider: provider,
		MaxBody: 64, InjectedDelay: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ref := provider.List()[0].Ref
	jws, _ := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	url := ts.URL + femock.PathPrefix + "/obterEstado"

	req := basicReq(t, http.MethodGet, url, mockUser, mockPass, map[string]string{"a": "b"})
	res, _ := http.DefaultClient.Do(req)
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("%d", res.StatusCode)
	}

	big := map[string]string{"identityRef": ref, "jws": strings.Repeat("a", 200), "idempotencyKey": "big"}
	res, _ = http.DefaultClient.Do(basicReq(t, http.MethodPost, url, mockUser, mockPass, big))
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("%d", res.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(`{"identityRef":"`+ref+`","jws":"`+jws+`","idempotencyKey":"cancel"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(mockUser, mockPass)
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("cancel transport err: %v", err)
	} else {
		bb, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusRequestTimeout && !strings.Contains(string(bb), femock.CodeCancelled) {
			t.Fatalf("cancel: status=%d body=%s", res.StatusCode, bb)
		}
	}

	_ = srv.Close()
	res, _ = http.DefaultClient.Do(basicReq(t, http.MethodPost, url, mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": jws, "idempotencyKey": "after-close",
	}))
	bb, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(bb), femock.CodeClosed) && !strings.Contains(string(bb), femock.CodeUnauthorized) {
		t.Fatalf("after Close: %s", bb)
	}
}

func TestNoExternalDial(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	jws, _ := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	client := ts.Client()
	res, err := client.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": jws, "idempotencyKey": "local",
	}))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%d", res.StatusCode)
	}
}

func TestProductionSignStillBlocked(t *testing.T) {
	_, err := feprofile.SignObterEstadoRequest(nil, "", feprofile.ObterEstadoRequestClaims{})
	if err == nil || !strings.Contains(err.Error(), feprofile.ConflictTyp) {
		t.Fatalf("%v", err)
	}
}

func TestFERNGCatalogHasSources(t *testing.T) {
	cat := femock.FERNGCatalog()
	if len(cat) == 0 {
		t.Fatal("empty")
	}
	emit := 0
	for _, e := range cat {
		if !strings.HasPrefix(e.Code, "FE-RNG-") || !strings.HasPrefix(e.SourceID, "AO-FE-SNAP") {
			t.Fatalf("%+v", e)
		}
		if e.Status == femock.FERNGEmitActive {
			emit++
		}
	}
	if emit < 6 {
		t.Fatalf("emit_active count %d", emit)
	}
}

func assertSanitized(t *testing.T, res *http.Response) {
	t.Helper()
	b, _ := io.ReadAll(res.Body)
	assertBodySanitized(t, b)
}

func assertBodySanitized(t *testing.T, b []byte) {
	t.Helper()
	s := string(b)
	for _, bad := range []string{"BEGIN ", "Authorization", "sourceLabel", "9100000", mockPass} {
		if strings.Contains(s, bad) {
			t.Fatalf("leak %q in %s", bad, s)
		}
	}
}
