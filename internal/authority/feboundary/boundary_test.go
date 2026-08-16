package feboundary_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fehub"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/femock"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

const (
	mockUser = "bwb-mock-user-synth"
	mockPass = "bwb-mock-pass-synth"
)

func setup(t *testing.T) (*feboundary.Engine, agttestkit.IdentityProvider, *femock.Server, func()) {
	t.Helper()
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		cleanupWB()
		t.Fatal(err)
	}
	mock, err := femock.New(femock.Config{Username: mockUser, Password: mockPass, Provider: provider})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mock.Handler())
	eng, err := feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: provider,
		BaseURL: ts.URL, Username: mockUser, Password: mockPass, Client: ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		_ = eng.Close()
		ts.Close()
		_ = mock.Close()
		_ = provider.Close()
		cleanupWB()
	}
	return eng, provider, mock, cleanup
}

func TestClientCheckRedirectNotMutatingShared(t *testing.T) {
	p, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	shared := &http.Client{Timeout: 2 * time.Second}
	orig := shared.CheckRedirect
	_, err = feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: p,
		BaseURL: "http://127.0.0.1:9", Username: "u", Password: "p", Client: shared,
	})
	if err != nil {
		t.Fatal(err)
	}
	if shared.CheckRedirect != nil && orig == nil {
		t.Fatal("New must not mutate shared http.Client.CheckRedirect")
	}
}

func TestHMLHubDenied(t *testing.T) {
	h, err := fehub.NewReserved(fehub.KindHML)
	if err != nil {
		t.Fatal(err)
	}
	p, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	_, err = feboundary.New(feboundary.Config{
		Hub: h, Provider: p, BaseURL: "http://127.0.0.1", Username: "u", Password: "p",
	})
	if !errors.Is(err, fehub.ErrTransportDenied) {
		t.Fatalf("%v", err)
	}
}

func TestRejectNonLoopbackBaseURL(t *testing.T) {
	p, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	cases := []string{
		"https://127.0.0.1",
		"http://example.com",
		"http://user:pass@127.0.0.1",
		"http://127.0.0.1/mock/agt-fe/v1",
		"http://127.0.0.1?x=1",
		"ftp://127.0.0.1",
		"",
	}
	for _, u := range cases {
		_, err := feboundary.New(feboundary.Config{
			Hub: fehub.NewFixture(), Provider: p, BaseURL: u, Username: "u", Password: "p",
		})
		if !errors.Is(err, feboundary.ErrBaseURLRejected) {
			t.Fatalf("%q: want ErrBaseURLRejected, got %v", u, err)
		}
	}
}

func TestRedirectDenied(t *testing.T) {
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	// Live success target: if redirects were followed, Process would reach StateOK.
	okBody := `{"simulated":true,"mock":"BWB-MOCK","status":"ok","requestID":"mock-redir-ok","operation":"obterEstado"}`
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, okBody)
	}))
	defer okSrv.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, okSrv.URL+femock.PathPrefix+"/obterEstado", http.StatusFound)
	}))
	defer redir.Close()
	eng, err := feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: p,
		BaseURL: redir.URL, Username: mockUser, Password: mockPass,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	out, err := eng.Process(context.Background(), feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: p.List()[0].Ref, IdempotencyKey: "redir",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "R",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != feboundary.StateFailed {
		t.Fatalf("want Failed when redirect denied (not OK via follow): %+v", out)
	}
}

func TestHTTP200RequiresBWBMockEnvelope(t *testing.T) {
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	cases := []struct {
		name string
		ct   string
		body string
	}{
		{"empty", "application/json", ""},
		{"no envelope", "application/json", `{"status":"ok","requestID":"r1"}`},
		{"wrong mock", "application/json", `{"simulated":true,"mock":"JWT","status":"ok","requestID":"r1"}`},
		{"no status", "application/json", `{"simulated":true,"mock":"BWB-MOCK","requestID":"r1"}`},
		{"bad ct", "text/plain", `{"simulated":true,"mock":"BWB-MOCK","status":"ok","requestID":"r1"}`},
		{"jsonp ct", "application/jsonp", `{"simulated":true,"mock":"BWB-MOCK","status":"ok","requestID":"r1"}`},
		{"op mismatch", "application/json", `{"simulated":true,"mock":"BWB-MOCK","status":"ok","requestID":"r1","operation":"consultarFactura"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.ct != "" {
					w.Header().Set("Content-Type", tc.ct)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer ts.Close()
			eng, err := feboundary.New(feboundary.Config{
				Hub: fehub.NewFixture(), Provider: p,
				BaseURL: ts.URL, Username: "u", Password: "p", Client: ts.Client(),
			})
			if err != nil {
				t.Fatal(err)
			}
			defer eng.Close()
			sub, _ := eng.Enqueue(feboundary.OpObterEstado)
			out, err := eng.Process(context.Background(), feboundary.ProcessInput{
				SubmissionID: sub.ID, IdentityRef: p.List()[0].Ref, IdempotencyKey: "bad-" + tc.name,
				ObterEstado: &feprofile.ObterEstadoRequestClaims{
					TaxRegistrationNumber: "9100000000", RequestID: "R",
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if out.State != feboundary.StateFailed || out.IsAGTAccepted() {
				t.Fatalf("%+v", out)
			}
		})
	}
}

func TestConcurrentProcessSameSubmission(t *testing.T) {
	eng, provider, _, cleanup := setup(t)
	defer cleanup()
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	ref := provider.List()[0].Ref
	in := feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: ref, IdempotencyKey: "conc",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "R",
		},
	}
	// First Process wins; second must see ErrInFlight or ErrInvalidTransition.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	states := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := eng.Process(context.Background(), in)
			if err != nil {
				errs <- err
				return
			}
			states <- out.State
		}()
	}
	wg.Wait()
	close(errs)
	close(states)
	var errN, okN int
	for err := range errs {
		if errors.Is(err, feboundary.ErrInFlight) || errors.Is(err, feboundary.ErrInvalidTransition) {
			errN++
		} else {
			t.Fatalf("unexpected err %v", err)
		}
	}
	for st := range states {
		if st == feboundary.StateOK {
			okN++
		}
	}
	if okN != 1 || errN != 1 {
		t.Fatalf("ok=%d err=%d want 1/1", okN, errN)
	}
	got, _ := eng.Get(sub.ID)
	if got.Attempts != 1 {
		t.Fatalf("attempts=%d want 1 (no duplicate POST counting)", got.Attempts)
	}
}

func TestRejectRepeatProcessAfterTerminal(t *testing.T) {
	eng, provider, _, cleanup := setup(t)
	defer cleanup()
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	in := feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: provider.List()[0].Ref, IdempotencyKey: "once",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "R",
		},
	}
	if _, err := eng.Process(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	_, err := eng.Process(context.Background(), in)
	if !errors.Is(err, feboundary.ErrInvalidTransition) {
		t.Fatalf("%v", err)
	}
}

func TestSignInvalidInputNotTransportFailedNil(t *testing.T) {
	eng, provider, _, cleanup := setup(t)
	defer cleanup()
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	_, err := eng.Process(context.Background(), feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: provider.List()[0].Ref, IdempotencyKey: "nil-payload",
		ObterEstado: nil,
	})
	if !errors.Is(err, feboundary.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	got, _ := eng.Get(sub.ID)
	if got.State != feboundary.StateFailed {
		t.Fatalf("state=%s", got.State)
	}
	if got.Note == "fixture_boundary_transport_failed" || strings.Contains(got.Note, "transport failed") {
		t.Fatalf("must not mislabel validation as transport: %s", got.Note)
	}
}

func TestCloseRaceSnapshot(t *testing.T) {
	eng, provider, _, cleanup := setup(t)
	defer cleanup()
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	in := feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: provider.List()[0].Ref, IdempotencyKey: "close-race",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "R",
		},
	}
	done := make(chan struct{})
	go func() {
		_, _ = eng.Process(context.Background(), in)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	_ = eng.Close()
	<-done
	// No panic / race under -race; closed engine rejects further enqueue.
	if _, err := eng.Enqueue(feboundary.OpObterEstado); !errors.Is(err, feboundary.ErrClosed) {
		t.Fatalf("%v", err)
	}
}

func TestProfileBlockedBranchViaStub(t *testing.T) {
	path, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"simulated": true,
			"mock":      femock.TypMock,
			"requestID": "mock-blocked-1",
			"code":      femock.CodeProfileBlocked,
		})
	}))
	defer ts.Close()
	eng, err := feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: p,
		BaseURL: ts.URL, Username: "u", Password: "p", Client: ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	out, err := eng.Process(context.Background(), feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: p.List()[0].Ref, IdempotencyKey: "blocked",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "R",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != feboundary.StateBlocked || out.IsAGTAccepted() {
		t.Fatalf("%+v", out)
	}
}

func TestObterEstadoToBoundaryOKNeverAGTAccepted(t *testing.T) {
	eng, provider, _, cleanup := setup(t)
	defer cleanup()
	sub, err := eng.Enqueue(feboundary.OpObterEstado)
	if err != nil || sub.State != feboundary.StateQueued {
		t.Fatalf("%+v %v", sub, err)
	}
	ref := provider.List()[0].Ref
	out, err := eng.Process(context.Background(), feboundary.ProcessInput{
		SubmissionID:   sub.ID,
		IdentityRef:    ref,
		IdempotencyKey: "idem-1",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "SYNTHETIC_REQ",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != feboundary.StateOK {
		t.Fatalf("%+v", out)
	}
	if out.IsAGTAccepted() {
		t.Fatal("must never label AGT accepted")
	}
	if !strings.Contains(out.Note, "≠ AGT") {
		t.Fatalf("%s", out.Note)
	}
	dump := out.ID + out.State + out.Note + out.MockCode
	if strings.Contains(dump, "BEGIN") || strings.Contains(dump, "9100000") {
		t.Fatalf("leak in state: %s", dump)
	}
}

func TestLocalSignatureAloneNotAccepted(t *testing.T) {
	eng, provider, _, cleanup := setup(t)
	defer cleanup()
	ref := provider.List()[0].Ref
	_, err := femock.SignObterEstadoMock(provider, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000", RequestID: "R",
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := eng.Enqueue(feboundary.OpObterEstado)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := eng.Get(sub.ID)
	if got.State != feboundary.StateQueued || got.IsAGTAccepted() {
		t.Fatalf("signature alone must not accept: %+v", got)
	}
}

func TestSoftwareInfoSyntheticMock(t *testing.T) {
	prod, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer prod.Close()
	mock, err := femock.New(femock.Config{Username: mockUser, Password: mockPass, Provider: prod})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mock.Handler())
	defer ts.Close()
	defer mock.Close()
	eng, err := feboundary.New(feboundary.Config{
		Hub: fehub.NewFixture(), Provider: prod,
		BaseURL: ts.URL, Username: mockUser, Password: mockPass, Client: ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	sub, _ := eng.Enqueue(feboundary.OpSoftwareInfo)
	out, err := eng.Process(context.Background(), feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: prod.List()[0].Ref, IdempotencyKey: "sw",
		Software: &feprofile.SoftwareInfoClaims{
			ProductID: "SYNTHETIC_PRODUCT", ProductVersion: "0.0.1-test",
			SoftwareValidationNumber: "SYNTHETIC_SVN_PLACEHOLDER",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != feboundary.StateOK || out.IsAGTAccepted() {
		t.Fatalf("%+v", out)
	}
	v := eng.HubView()
	if v.Kind != string(fehub.KindFixture) || v.ExternalVerified {
		t.Fatalf("%+v", v)
	}
}

func TestFERNGRejectState(t *testing.T) {
	eng, provider, mock, cleanup := setup(t)
	defer cleanup()
	if err := mock.ScriptFERNG("obterEstado", "FE-RNG-032"); err != nil {
		t.Fatal(err)
	}
	sub, _ := eng.Enqueue(feboundary.OpObterEstado)
	out, err := eng.Process(context.Background(), feboundary.ProcessInput{
		SubmissionID: sub.ID, IdentityRef: provider.List()[0].Ref, IdempotencyKey: "ferng",
		ObterEstado: &feprofile.ObterEstadoRequestClaims{
			TaxRegistrationNumber: "9100000000", RequestID: "R",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != feboundary.StateReject || out.MockCode != "FE-RNG-032" {
		t.Fatalf("%+v", out)
	}
	if out.SourceID != "AO-FE-SNAP-HML-2026-07-25-CONSULTAR" {
		t.Fatalf("%s", out.SourceID)
	}
	if out.IsAGTAccepted() {
		t.Fatal("AGT accept")
	}
}

func TestUnknownOpAndClose(t *testing.T) {
	eng, _, _, cleanup := setup(t)
	defer cleanup()
	if _, err := eng.Enqueue("registarFactura"); !errors.Is(err, feboundary.ErrUnknownOp) {
		t.Fatalf("%v", err)
	}
	_ = eng.Close()
	if _, err := eng.Enqueue(feboundary.OpObterEstado); !errors.Is(err, feboundary.ErrClosed) {
		t.Fatalf("%v", err)
	}
}
