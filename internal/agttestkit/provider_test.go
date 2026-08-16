package agttestkit_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/xuri/excelize/v2"
)

func TestWorkbookProviderResolveAndSign(t *testing.T) {
	path, cleanup := synthWorkbook(t, 2)
	defer cleanup()

	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()

	list := p.List()
	if len(list) != 2 {
		t.Fatalf("list=%d", len(list))
	}
	for _, e := range list {
		assertSanitizedListing(t, e)
		if e.Role != agttestkit.RoleTaxpayerTest {
			t.Fatalf("role=%q", e.Role)
		}
	}
	ref := list[0].Ref
	signer, err := p.Signer(ref)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("generic-payload-rm-fefix-002")
	sig, err := agttestkit.SignMessageRSA(signer, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(ref, msg, sig); err != nil {
		t.Fatal(err)
	}
}

func TestWorkbookProviderRefNotFound(t *testing.T) {
	path, cleanup := synthWorkbook(t, 1)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	_, err = p.Signer("agt-test-deadbeefdeadbeef")
	if !errors.Is(err, agttestkit.ErrRefNotFound) {
		t.Fatalf("got %v", err)
	}
	assertNoSecrets(t, err.Error())
}

func TestWorkbookProviderRefRequired(t *testing.T) {
	path, cleanup := synthWorkbook(t, 1)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	_, err = p.Signer("   ")
	if !errors.Is(err, agttestkit.ErrRefRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestOpaqueRefStableAcrossOpens(t *testing.T) {
	path, cleanup := synthWorkbook(t, 2)
	defer cleanup()
	p1, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	refs1 := refsOf(p1.List())
	_ = p1.Close()

	p2, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p2.Close() }()
	refs2 := refsOf(p2.List())
	if len(refs1) != len(refs2) {
		t.Fatal("length")
	}
	for i := range refs1 {
		if refs1[i] != refs2[i] {
			t.Fatalf("unstable refs")
		}
	}
}

func TestOpaqueRefsDifferForDifferentKeys(t *testing.T) {
	path, cleanup := synthWorkbook(t, 3)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	seen := map[string]struct{}{}
	for _, e := range p.List() {
		if _, ok := seen[e.Ref]; ok {
			t.Fatalf("duplicate ref")
		}
		seen[e.Ref] = struct{}{}
	}
	if len(seen) != 3 {
		t.Fatal(len(seen))
	}
}

func TestSignatureDoesNotVerifyWithOtherKey(t *testing.T) {
	path, cleanup := synthWorkbook(t, 2)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	list := p.List()
	s0, err := p.Signer(list[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("cross-key")
	sig, err := agttestkit.SignMessageRSA(s0, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(list[1].Ref, msg, sig); !errors.Is(err, agttestkit.ErrVerifyFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestProviderClosedBlocksOps(t *testing.T) {
	path, cleanup := synthWorkbook(t, 1)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	ref := p.List()[0].Ref
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Signer(ref); !errors.Is(err, agttestkit.ErrProviderClosed) {
		t.Fatalf("got %v", err)
	}
	if err := p.Verify(ref, []byte("x"), []byte("y")); !errors.Is(err, agttestkit.ErrProviderClosed) {
		t.Fatalf("got %v", err)
	}
	if len(p.List()) != 0 {
		t.Fatal("list after close")
	}
}

func TestInvalidWorkbookLeavesNoProvider(t *testing.T) {
	dir := t.TempDir()
	path := writeBrokenWorkbook(t, dir)
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err == nil || p != nil {
		t.Fatalf("want fail-closed, p=%v err=%v", p, err)
	}
	assertNoSecrets(t, err.Error())
}

func TestProviderConcurrentSign(t *testing.T) {
	path, cleanup := synthWorkbook(t, 2)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	ref := p.List()[0].Ref
	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, err := p.Signer(ref)
			if err != nil {
				errCh <- err
				return
			}
			msg := []byte(fmt.Sprintf("msg-%d", i))
			sig, err := agttestkit.SignMessageRSA(s, msg)
			if err != nil {
				errCh <- err
				return
			}
			if err := p.Verify(ref, msg, sig); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

func TestSanitizedSurfacesHaveNoSecrets(t *testing.T) {
	path, cleanup := synthWorkbook(t, 2)
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	for _, e := range p.List() {
		assertSanitizedListing(t, e)
		assertNoSecrets(t, fmt.Sprintf("%+v", e))
	}
	_, err = p.Signer("missing")
	assertNoSecrets(t, err.Error())
}

func TestEphemeralProducerDistinctAndNonPersistent(t *testing.T) {
	a, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close(); _ = b.Close() }()
	ra, rb := a.List()[0], b.List()[0]
	if ra.Role != agttestkit.RoleProducerEphemeral || rb.Role != agttestkit.RoleProducerEphemeral {
		t.Fatal(ra.Role, rb.Role)
	}
	if ra.Ref == rb.Ref {
		t.Fatal("ephemeral refs must differ across runs")
	}
	sa, err := a.Signer(ra.Ref)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("producer")
	siga, err := agttestkit.SignMessageRSA(sa, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Verify(rb.Ref, msg, siga); !errors.Is(err, agttestkit.ErrVerifyFailed) {
		t.Fatalf("cross ephemeral verify: %v", err)
	}
	assertSanitizedListing(t, ra)
}

func synthWorkbook(t *testing.T, count int) (string, func()) {
	t.Helper()
	path, cleanup, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: count})
	if err != nil {
		t.Fatal(err)
	}
	return path, cleanup
}

func writeBrokenWorkbook(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "custom.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "NOT_A_HEADER")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return path
}

func refsOf(list []agttestkit.SanitizedRef) []string {
	out := make([]string, len(list))
	for i, e := range list {
		out[i] = e.Ref
	}
	return out
}

func assertSanitizedListing(t *testing.T, e agttestkit.SanitizedRef) {
	t.Helper()
	if e.Ref == "" || !strings.HasPrefix(e.Ref, "agt-test-") {
		t.Fatalf("ref %+v", e)
	}
	if e.Algorithm != "RSA" || e.RSABits < agttestkit.MinRSABits {
		t.Fatalf("meta %+v", e)
	}
	assertNoSecrets(t, e.Ref)
	assertNoSecrets(t, e.Role)
	assertNoSecrets(t, e.Algorithm)
}

func assertNoSecrets(t *testing.T, s string) {
	t.Helper()
	if agttestkit.ContainsPrivatePEMBlock([]byte(s)) {
		t.Fatal("private pem in string")
	}
	if strings.Contains(strings.ToLower(s), "-----begin") {
		t.Fatal("pem fencing")
	}
}
