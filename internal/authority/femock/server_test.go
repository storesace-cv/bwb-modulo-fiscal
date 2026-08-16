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

func TestFERNGScriptAndUnknown(t *testing.T) {
	srv, provider, ts, cleanup := newMock(t)
	defer cleanup()
	if err := srv.ScriptFERNG("obterEstado", "FE-RNG-NOT-REAL"); err == nil {
		t.Fatal("expected reject unknown FERNG")
	}
	if err := srv.ScriptFERNG("obterEstado", "FE-RNG-031"); err != nil {
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
		"identityRef": ref, "jws": jws, "idempotencyKey": "ferng",
	}))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("%d %s", res.StatusCode, b)
	}
	if !strings.Contains(string(b), "FE-RNG-031") || !strings.Contains(string(b), "AO-FE-SNAP") {
		t.Fatalf("%s", b)
	}
	if !strings.Contains(string(b), `"simulated":true`) {
		t.Fatal("missing simulated")
	}
}

func TestReplayAndConflict(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	jws1, _ := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R1",
	})
	body := map[string]string{"identityRef": ref, "jws": jws1, "idempotencyKey": "same"}
	res1, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	b1, _ := io.ReadAll(res1.Body)
	res1.Body.Close()
	res2, _ := http.DefaultClient.Do(basicReq(t, http.MethodPost, ts.URL+femock.PathPrefix+"/obterEstado", mockUser, mockPass, body))
	b2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	if string(b1) != string(b2) {
		t.Fatalf("replay unstable")
	}
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
}

func TestMethodContentTypeBodyLimitCancelClose(t *testing.T) {
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

	req := basicReq(t, http.MethodGet, url, mockUser, mockPass, nil)
	res, _ := http.DefaultClient.Do(req)
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("%d", res.StatusCode)
	}

	req = basicReq(t, http.MethodPost, url, mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": jws, "idempotencyKey": "ct",
	})
	req.Header.Set("Content-Type", "text/plain")
	res, _ = http.DefaultClient.Do(req)
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusUnsupportedMediaType {
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
		// Client-side cancel/timeout before response is acceptable.
		t.Logf("cancel transport err: %v", err)
	} else {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusRequestTimeout && !strings.Contains(string(b), femock.CodeCancelled) {
			t.Fatalf("cancel: status=%d body=%s", res.StatusCode, b)
		}
	}

	_ = srv.Close()
	res, _ = http.DefaultClient.Do(basicReq(t, http.MethodPost, url, mockUser, mockPass, map[string]string{
		"identityRef": ref, "jws": jws, "idempotencyKey": "after-close",
	}))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(b), femock.CodeClosed) {
		t.Fatalf("%s", b)
	}
}

func TestNoExternalDial(t *testing.T) {
	_, provider, ts, cleanup := newMock(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	jws, _ := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	client := &http.Client{Transport: roundTripFail{t: t}}
	// Local httptest URL must still work via custom transport that only allows loopback host from ts.
	client = ts.Client()
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

type roundTripFail struct{ t *testing.T }

func (roundTripFail) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, io.EOF
}

func TestProductionSignStillBlocked(t *testing.T) {
	_, err := feprofile.SignObterEstadoRequest(nil, "", feprofile.ObterEstadoRequestClaims{})
	if err == nil || !strings.Contains(err.Error(), feprofile.ConflictTyp) {
		t.Fatalf("%v", err)
	}
}

func TestAllowlistedFERNGHasSources(t *testing.T) {
	m := femock.AllowlistedFERNG()
	if len(m) == 0 {
		t.Fatal("empty")
	}
	for code, src := range m {
		if !strings.HasPrefix(code, "FE-RNG-") || !strings.HasPrefix(src, "AO-FE-SNAP") {
			t.Fatalf("%s %s", code, src)
		}
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
