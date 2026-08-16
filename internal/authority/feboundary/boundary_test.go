package feboundary_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

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
	// Sign locally without Process — enqueue only.
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
