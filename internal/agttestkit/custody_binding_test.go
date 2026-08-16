package agttestkit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
	"github.com/xuri/excelize/v2"
)

func TestHeldBindingNIFLabelKey(t *testing.T) {
	dir := t.TempDir()
	path := writeLabeledWorkbook(t, dir, []labeledRow{
		{nif: "9100000001", label: "SYNTHETIC_PROFILE_ALPHA"},
		{nif: "9100000002", label: "SYNTHETIC_PROFILE_BETA"},
	})
	p, err := OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	mp, ok := p.(*memoryProvider)
	if !ok {
		t.Fatal("expected memoryProvider")
	}

	type snap struct {
		nif, label, n string
	}
	byRef := map[string]snap{}
	mp.mu.RLock()
	for ref, h := range mp.byRef {
		if h.taxpayerNIF == "" || h.sourceLabel == "" || h.priv == nil {
			mp.mu.RUnlock()
			t.Fatalf("incomplete binding for %s", ref)
		}
		byRef[ref] = snap{nif: h.taxpayerNIF, label: h.sourceLabel, n: h.priv.N.String()}
	}
	mp.mu.RUnlock()

	if len(byRef) != 2 {
		t.Fatalf("want 2, got %d", len(byRef))
	}
	labels := map[string]string{}
	for ref, s := range byRef {
		if other, ok := labels[s.label]; ok && other != ref {
			t.Fatal("label reused across refs")
		}
		labels[s.label] = ref
		signer, err := p.Signer(ref)
		if err != nil {
			t.Fatal(err)
		}
		msg := []byte("bind-" + s.label)
		sig, err := SignMessageRSA(signer, msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Verify(ref, msg, sig); err != nil {
			t.Fatal(err)
		}
		for otherRef, other := range byRef {
			if otherRef == ref {
				continue
			}
			if err := p.Verify(otherRef, msg, sig); !errors.Is(err, ErrVerifyFailed) {
				t.Fatalf("cross verify: %v", err)
			}
			if other.label == s.label || other.nif == s.nif || other.n == s.n {
				t.Fatal("identities collapsed")
			}
		}
	}
	if _, ok := labels["SYNTHETIC_PROFILE_ALPHA"]; !ok {
		t.Fatal("missing alpha label binding")
	}
	if _, ok := labels["SYNTHETIC_PROFILE_BETA"]; !ok {
		t.Fatal("missing beta label binding")
	}

	for _, e := range p.List() {
		dump := fmt.Sprintf("%+v", e)
		if strings.Contains(dump, "9100000001") || strings.Contains(dump, "SYNTHETIC_PROFILE") {
			t.Fatalf("list leaked: %s", dump)
		}
	}
	_, err = p.Signer("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "9100000001") || strings.Contains(err.Error(), "SYNTHETIC_PROFILE") {
		t.Fatalf("error leaked: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	if mp.byRef != nil {
		t.Fatal("byRef not cleared")
	}
}

func TestOpaqueSignerNotPrivateKey(t *testing.T) {
	path, cleanup, err := WriteSyntheticWorkbook(t.TempDir(), SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	s, err := p.Signer(p.List()[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*rsa.PrivateKey); ok {
		t.Fatal("Signer must not return *rsa.PrivateKey")
	}
	if _, ok := s.(*opaqueSigner); !ok {
		t.Fatalf("want *opaqueSigner, got %T", s)
	}
	pub := s.Public()
	rk, ok := pub.(*rsa.PublicKey)
	if !ok || rk == nil || rk.N == nil {
		t.Fatalf("public: %T", pub)
	}
	msg := []byte("opaque-ok")
	sig, err := SignMessageRSA(s, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(p.List()[0].Ref, msg, sig); err != nil {
		t.Fatal(err)
	}
}

func TestOpaqueSignerFailsAfterClose(t *testing.T) {
	path, cleanup, err := WriteSyntheticWorkbook(t.TempDir(), SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.Signer(p.List()[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = SignMessageRSA(s, []byte("after-close"))
	if !errors.Is(err, ErrProviderClosed) {
		t.Fatalf("got %v", err)
	}
	if pub := s.Public(); pub != nil {
		t.Fatalf("public after close: %v", pub)
	}
}

func TestOpaqueSignerConcurrentWithClose(t *testing.T) {
	path, cleanup, err := WriteSyntheticWorkbook(t.TempDir(), SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.Signer(p.List()[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = SignMessageRSA(s, []byte(fmt.Sprintf("c-%d", i)))
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Close()
	}()
	wg.Wait()
	_, err = SignMessageRSA(s, []byte("final"))
	if err != nil && !errors.Is(err, ErrProviderClosed) && !errors.Is(err, ErrSignerUnavailable) {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestAllProvidersReturnOpaqueSigner(t *testing.T) {
	path, cleanup, err := WriteSyntheticWorkbook(t.TempDir(), SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	wb, err := OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = wb.Close() }()
	assertOpaque(t, wb)

	eph, err := OpenEphemeralProducerProvider(MinRSABits)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = eph.Close() }()
	assertOpaque(t, eph)
}

func TestSecretStoreAdapterCompatibility(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, MinRSABits)
	if err != nil {
		t.Fatal(err)
	}
	privPEM, err := marshalPKCS8Private(priv)
	if err != nil {
		t.Fatal(err)
	}
	wantRef := opaqueRefFromPublic(&priv.PublicKey)

	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	storeRef := secretstore.Ref{
		Kind:        secretstore.KindTaxpayerKey,
		Environment: secretstore.EnvHomologation,
		SubjectID:   "platform",
		Name:        "adapter-test",
	}
	if _, err := mem.Put(context.Background(), storeRef, privPEM, nil); err != nil {
		t.Fatal(err)
	}
	zeroBytes(privPEM)

	p, err := OpenSecretStorePEMProvider(context.Background(), mem, []SecretStorePEMBinding{{
		OpaqueRef: wantRef,
		StoreRef:  storeRef,
		Role:      RoleSecretStoreAdapter,
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	assertOpaque(t, p)
	s, err := p.Signer(wantRef)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("via-secretstore-adapter")
	sig, err := SignMessageRSA(s, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(wantRef, msg, sig); err != nil {
		t.Fatal(err)
	}
}

func TestSecretStoreAdapterRejectsOpaqueMismatch(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, MinRSABits)
	if err != nil {
		t.Fatal(err)
	}
	privPEM, err := marshalPKCS8Private(priv)
	if err != nil {
		t.Fatal(err)
	}
	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	storeRef := secretstore.Ref{
		Kind:        secretstore.KindTaxpayerKey,
		Environment: secretstore.EnvHomologation,
		SubjectID:   "platform",
		Name:        "mismatch",
	}
	if _, err := mem.Put(context.Background(), storeRef, privPEM, nil); err != nil {
		t.Fatal(err)
	}
	zeroBytes(privPEM)
	p, err := OpenSecretStorePEMProvider(context.Background(), mem, []SecretStorePEMBinding{{
		OpaqueRef: "agt-test-0000000000000000",
		StoreRef:  storeRef,
	}})
	if err == nil || p != nil {
		t.Fatal("want opaque mismatch fail-closed")
	}
	if !errors.Is(err, ErrRefAmbiguous) {
		t.Fatalf("got %v", err)
	}
}

func assertOpaque(t *testing.T, p IdentityProvider) {
	t.Helper()
	s, err := p.Signer(p.List()[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*rsa.PrivateKey); ok {
		t.Fatal("*rsa.PrivateKey leaked")
	}
	if _, ok := s.(*opaqueSigner); !ok {
		t.Fatalf("got %T", s)
	}
}

type labeledRow struct {
	nif, label string
}

func writeLabeledWorkbook(t *testing.T, dir string, rows []labeledRow) string {
	t.Helper()
	path := filepath.Join(dir, "custom.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetSheetName(sheet, "Folha1")
	sheet = "Folha1"
	headers := []string{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for r, row := range rows {
		priv, err := rsa.GenerateKey(rand.Reader, MinRSABits)
		if err != nil {
			t.Fatal(err)
		}
		privPEM, err := marshalPKCS8Private(priv)
		if err != nil {
			t.Fatal(err)
		}
		pubPEM, err := marshalPKIXPublic(&priv.PublicKey)
		if err != nil {
			t.Fatal(err)
		}
		vals := []string{row.nif, row.label, string(privPEM), string(pubPEM)}
		for c, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
		zeroBytes(privPEM)
		zeroBytes(pubPEM)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return path
}
