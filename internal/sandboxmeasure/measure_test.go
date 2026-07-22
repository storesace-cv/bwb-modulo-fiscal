package sandboxmeasure_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/sandboxmeasure"
)

const fixtureJSON = `{
  "external_id": "FIXTURE-SBOX-EXT-001",
  "document_type": "invoice",
  "currency": "AOA",
  "issued_at": "2026-07-21T10:00:00+01:00",
  "requested_series": "IGNORED",
  "seller": {"tax_id": "FIXTURE-NIF-AO-0001", "name": "Fixture Seller Synthetic AO"},
  "lines": [{"line_id": "L1", "description": "Fixture line", "quantity": "1", "unit_price": "10.50", "tax_code": "NOR"}]
}`

type fakeClock struct {
	mu   sync.Mutex
	now  time.Time
	wait []time.Duration
}

func newFakeClock(t0 time.Time) *fakeClock {
	return &fakeClock{now: t0}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Since(t time.Time) time.Duration {
	return f.Now().Sub(t)
}

func (f *fakeClock) NewTimer(d time.Duration) sandboxmeasure.Timer {
	f.mu.Lock()
	f.wait = append(f.wait, d)
	f.now = f.now.Add(d)
	f.mu.Unlock()
	ch := make(chan time.Time, 1)
	ch <- f.Now()
	return instantTimer{ch: ch}
}

type instantTimer struct{ ch chan time.Time }

func (i instantTimer) C() <-chan time.Time { return i.ch }
func (i instantTimer) Stop() bool          { return true }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func writeToken(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "create-document.min.json")
	if err := os.WriteFile(path, []byte(fixtureJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNearestRank(t *testing.T) {
	vals := []int64{100, 200, 300, 400, 500}
	if got := sandboxmeasure.NearestRank(vals, 50); got != 300 {
		t.Fatalf("p50=%d", got)
	}
	if got := sandboxmeasure.NearestRank(vals, 95); got != 500 {
		t.Fatalf("p95=%d", got)
	}
	if got := sandboxmeasure.NearestRank(nil, 50); got != 0 {
		t.Fatalf("empty=%d", got)
	}
}

func TestUniqueKeysAndClassifyViaBurst(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok-secret-value")
	fix := writeFixture(t, dir)

	var seenExt sync.Map
	var seenIdem sync.Map
	var calls atomic.Int64
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if ah := r.Header.Get("Authorization"); !strings.HasPrefix(ah, "Bearer ") {
			t.Error("missing bearer")
		}
		if strings.Contains(r.Header.Get("Authorization"), "tok-secret") {
			// ok present in request; must never appear in report/errors
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		ext, _ := m["external_id"].(string)
		if ext == "" || ext == "FIXTURE-SBOX-EXT-001" {
			t.Fatalf("external_id not unique: %q", ext)
		}
		if _, ok := seenExt.LoadOrStore(ext, true); ok {
			t.Fatalf("duplicate external_id %q", ext)
		}
		idem := r.Header.Get("Idempotency-Key")
		if idem == "" {
			t.Fatal("missing idempotency")
		}
		if _, ok := seenIdem.LoadOrStore(idem, true); ok {
			t.Fatalf("duplicate idem %q", idem)
		}
		// preserve NIF
		seller := m["seller"].(map[string]any)
		if seller["tax_id"] != "FIXTURE-NIF-AO-0001" {
			t.Fatalf("nif mutated")
		}
		code := 201
		n := calls.Load()
		switch {
		case n%17 == 0:
			code = 429
		case n%19 == 0:
			code = 409
		case n%23 == 0:
			code = 500
		case n%29 == 0:
			code = 422
		}
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(`{"id":"x","external_id":"` + ext + `","status":"sealed_locally","submission_id":"s","created_at":"t"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileBurst,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Sent != 60 {
		t.Fatalf("sent=%d", rep.Sent)
	}
	if rep.Status201+rep.Status409+rep.Status429+rep.Status5xx+rep.StatusOther != 60 {
		t.Fatalf("class sum mismatch: %+v", rep)
	}
	if rep.RequestThroughput <= 0 || rep.AcceptedThroughput < 0 {
		t.Fatalf("throughput %+v", rep)
	}
}

func TestSustainedMonotonicNoCatchUp(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	clk := newFakeClock(time.Unix(0, 0).UTC())

	var starts []time.Time
	var mu sync.Mutex
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		starts = append(starts, clk.Now())
		mu.Unlock()
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"id":"a","external_id":"e","status":"sealed_locally","submission_id":"s","created_at":"t"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	// Use a reduced sustained via hacking: we can't change Total without Spec.
	// Instead verify Spec values and that wait durations are non-decreasing agenda gaps.
	spec, err := sandboxmeasure.Spec(sandboxmeasure.ProfileSustained)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Total != 300 || spec.RatePerSec != 10 || spec.Concurrency != 1 {
		t.Fatalf("%+v", spec)
	}

	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileSustained,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  false,
		Clock:              clk,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Sent != 300 {
		t.Fatalf("sent=%d", rep.Sent)
	}
	if len(clk.wait) < 299 {
		t.Fatalf("expected pacing waits, got %d", len(clk.wait))
	}
	// No catch-up: each wait should be ~100ms when on schedule (fake clock advances exactly).
	for i, w := range clk.wait {
		if w < 90*time.Millisecond || w > 110*time.Millisecond {
			t.Fatalf("wait[%d]=%v (catch-up or wrong pace?)", i, w)
		}
	}
	if rep.Status201 != 300 {
		t.Fatalf("201=%d", rep.Status201)
	}
}

func TestReplayIdentical(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	var bodies [][]byte
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, b)
		idem := r.Header.Get("Idempotency-Key")
		if len(bodies) == 2 && idem == "" {
			t.Fatal("missing idem")
		}
		payload := `{"id":"doc1","external_id":"E","status":"sealed_locally","submission_id":"sub1","created_at":"2026-01-01T00:00:00Z"}`
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ReplayIdentical || rep.Sent != 2 || rep.Status201 != 2 {
		t.Fatalf("%+v", rep)
	}
	if len(bodies) != 2 || !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatal("replay request bodies must match")
	}
}

func TestTokenSymlinkRejected(t *testing.T) {
	dir := t.TempDir()
	real := writeToken(t, dir, "real.token", "tok")
	link := filepath.Join(dir, "measure.token")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink not supported")
	}
	fix := writeFixture(t, dir)
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          link,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		HTTPClient:         &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("should not call"); return nil, nil })},
	}
	_, err := sandboxmeasure.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "tok-secret") || strings.Contains(strings.ToLower(err.Error()), "bearer") {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func TestTokenMissing(t *testing.T) {
	dir := t.TempDir()
	fix := writeFixture(t, dir)
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          filepath.Join(dir, "missing.token"),
		FixturePath:        fix,
		AllowNonFixedPaths: true,
	}
	_, err := sandboxmeasure.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFixedURLNotControllable(t *testing.T) {
	cfg := sandboxmeasure.Config{
		Profile:     sandboxmeasure.ProfileBurst,
		BaseURL:     "http://evil.example",
		TokenPath:   sandboxmeasure.FixedTokenFile,
		FixturePath: "/tmp/x",
	}
	_, err := sandboxmeasure.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "not controllable") {
		t.Fatalf("got %v", err)
	}
}

func TestCancelContext(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileBurst,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		HTTPClient:         client,
	}
	_, err := sandboxmeasure.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected cancel")
	}
}

func TestBodyTooLarge(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	big := strings.Repeat("a", 65<<10)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(big)), Header: make(http.Header)}, nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		HTTPClient:         client,
	}
	_, err := sandboxmeasure.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReportNoSecrets(t *testing.T) {
	rep := sandboxmeasure.Report{Profile: "burst", Sent: 1, Status201: 1}
	var buf bytes.Buffer
	if err := sandboxmeasure.WriteReport(&buf, rep); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, bad := range []string{"bwb_sbox_", "Bearer", "postgres://", "FIXTURE-NIF", "Authorization"} {
		if strings.Contains(out, bad) {
			t.Fatalf("secret pattern %q in output", bad)
		}
	}
}

func TestMalformedFixture(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	bad := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(bad, []byte("{"), 0o644)
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        bad,
		AllowNonFixedPaths: true,
	}
	_, err := sandboxmeasure.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}
