package sandboxmeasure_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

const validReplayBody = `{"id":"doc1","external_id":"E","status":"sealed_locally","submission_id":"sub1","created_at":"2026-01-01T00:00:00Z"}`

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

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
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

func ok201(body string) *http.Response {
	if body == "" {
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}
	}
	return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
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
			Body:       io.NopCloser(strings.NewReader(validReplayBody)),
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
	if rep.Attempted != 60 || rep.HTTPResponses != 60 || rep.TransportErrors != 0 {
		t.Fatalf("counts %+v", rep)
	}
	if rep.Status201+rep.Status409+rep.Status429+rep.Status5xx+rep.StatusOther != 60 {
		t.Fatalf("class sum mismatch: %+v", rep)
	}
}

func TestSustainedPacingOnSchedule(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	clk := newFakeClock(time.Unix(0, 0).UTC())

	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return ok201(validReplayBody), nil
	})}

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
	if rep.Attempted != 300 || rep.HTTPResponses != 300 {
		t.Fatalf("attempted=%d http=%d", rep.Attempted, rep.HTTPResponses)
	}
	if len(clk.wait) < 299 {
		t.Fatalf("expected pacing waits, got %d", len(clk.wait))
	}
	for i, w := range clk.wait {
		if w < 90*time.Millisecond || w > 110*time.Millisecond {
			t.Fatalf("wait[%d]=%v", i, w)
		}
	}
}

func TestSustainedNoCatchUpAfterArtificialDelay(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	clk := newFakeClock(time.Unix(0, 0).UTC())

	var starts []time.Time
	var mu sync.Mutex
	var n atomic.Int64
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		mu.Lock()
		starts = append(starts, clk.Now())
		mu.Unlock()
		if n.Add(1) == 1 {
			// Artificial delay longer than one interval — must not compress overdue slots.
			clk.Advance(250 * time.Millisecond)
		}
		return ok201(validReplayBody), nil
	})}

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
	if rep.Attempted != 300 {
		t.Fatalf("attempted=%d", rep.Attempted)
	}
	if len(starts) < 3 {
		t.Fatal("need starts")
	}
	// After delayed first request, gap to second must be >= interval (no recovery burst).
	gap01 := starts[1].Sub(starts[0])
	if gap01 < 90*time.Millisecond {
		t.Fatalf("catch-up burst detected: gap01=%v", gap01)
	}
	if gap01 < 340*time.Millisecond || gap01 > 360*time.Millisecond {
		// 250ms delay + 100ms interval from "now"
		t.Fatalf("expected ~350ms gap after delay, got %v", gap01)
	}
	zeroWaits := 0
	for _, w := range clk.wait {
		if w == 0 {
			zeroWaits++
		}
	}
	if zeroWaits != 0 {
		t.Fatalf("compressed waits (catch-up): zeroWaits=%d waits=%v", zeroWaits, clk.wait[:min(5, len(clk.wait))])
	}
	// Subsequent waits remain full intervals (no compressed recovery).
	for i, w := range clk.wait {
		if w < 90*time.Millisecond || w > 110*time.Millisecond {
			t.Fatalf("wait[%d]=%v after delay (expected ~100ms)", i, w)
		}
	}
}

func TestLatencyUsesInjectedClock(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	clk := newFakeClock(time.Unix(0, 0).UTC())
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		clk.Advance(120 * time.Millisecond)
		return ok201(validReplayBody), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		Clock:              clk,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rep.P50MS201 < 100 || rep.P50MS201 > 140 {
		t.Fatalf("p50_ms_201=%d (clock not used?)", rep.P50MS201)
	}
}

func TestBurstThresholdsFailWithReport(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return ok201(validReplayBody), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileBurst,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if !errors.Is(err, sandboxmeasure.ErrThresholds) {
		t.Fatalf("err=%v", err)
	}
	if rep.Passed {
		t.Fatal("expected passed=false")
	}
	if !contains(rep.FailureCodes, "burst_201_out_of_range") {
		t.Fatalf("codes=%v", rep.FailureCodes)
	}
	var buf bytes.Buffer
	if err := sandboxmeasure.WriteReport(&buf, rep); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["passed"] != false {
		t.Fatalf("json passed=%v", decoded["passed"])
	}
	if _, ok := decoded["failure_codes"]; !ok {
		t.Fatal("missing failure_codes")
	}
}

func TestTransportErrorPreservesMetrics(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	var n atomic.Int64
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if n.Add(1) > 5 {
			return nil, errors.New("connection reset")
		}
		return ok201(validReplayBody), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileSustained,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  true,
		Clock:              newFakeClock(time.Unix(0, 0).UTC()),
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if !errors.Is(err, sandboxmeasure.ErrTransport) {
		t.Fatalf("err=%v", err)
	}
	if rep.Passed {
		t.Fatal("expected failed")
	}
	if rep.Attempted != 6 || rep.HTTPResponses != 5 || rep.TransportErrors != 1 {
		t.Fatalf("attempted=%d http=%d transport=%d", rep.Attempted, rep.HTTPResponses, rep.TransportErrors)
	}
	// Throughput is over HTTP responses only.
	if rep.HTTPResponses == 0 || rep.RequestThroughput <= 0 {
		t.Fatalf("throughput %+v", rep)
	}
	if !contains(rep.FailureCodes, "transport_error") {
		t.Fatalf("codes=%v", rep.FailureCodes)
	}
	raw, _ := json.Marshal(rep)
	out := string(raw)
	for _, bad := range []string{"Bearer", "Authorization", "/v1/documents"} {
		if strings.Contains(out, bad) {
			t.Fatalf("sensitive %q in report", bad)
		}
	}
	if strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "Bearer") {
		t.Fatalf("error leaked: %v", err)
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
		return ok201(validReplayBody), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed || !rep.ReplayIdentical || rep.Attempted != 2 || rep.Status201 != 2 {
		t.Fatalf("%+v", rep)
	}
	if len(bodies) != 2 || !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatal("replay request bodies must match")
	}
}

func TestReplayEmptyBody(t *testing.T) {
	runReplayBodyCase(t, "", true)
}

func TestReplayInvalidJSON(t *testing.T) {
	runReplayBodyCase(t, "{", true)
}

func TestReplayMissingFields(t *testing.T) {
	runReplayBodyCase(t, `{"id":"x","status":"sealed_locally"}`, true)
}

func TestReplayProhibitedFiscalField(t *testing.T) {
	runReplayBodyCase(t, `{"id":"doc1","external_id":"E","status":"sealed_locally","submission_id":"sub1","created_at":"2026-01-01T00:00:00Z","fiscal_number":"FT 1"}`, true)
}

func TestReplayTrailingData(t *testing.T) {
	runReplayBodyCase(t, validReplayBody+"\n{}", true)
}

func TestReplayDifferentPayloads(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	var n atomic.Int64
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if n.Add(1) == 1 {
			return ok201(validReplayBody), nil
		}
		alt := `{"id":"doc2","external_id":"E","status":"sealed_locally","submission_id":"sub1","created_at":"2026-01-01T00:00:00Z"}`
		return ok201(alt), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if !errors.Is(err, sandboxmeasure.ErrThresholds) {
		t.Fatalf("err=%v", err)
	}
	if rep.Passed || !contains(rep.FailureCodes, "replay_invalid") {
		t.Fatalf("%+v", rep)
	}
}

func TestReplayTwoEmptyObjectsNotEqual(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return ok201(`{}`), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if !errors.Is(err, sandboxmeasure.ErrThresholds) {
		t.Fatalf("err=%v", err)
	}
	if rep.Passed || !contains(rep.FailureCodes, "replay_invalid") {
		t.Fatalf("%+v", rep)
	}
}

func runReplayBodyCase(t *testing.T, body string, wantFail bool) {
	t.Helper()
	dir := t.TempDir()
	tokenPath := writeToken(t, dir, "measure.token", "tok")
	fix := writeFixture(t, dir)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return ok201(body), nil
	})}
	cfg := sandboxmeasure.Config{
		Profile:            sandboxmeasure.ProfileReplay,
		BaseURL:            "http://127.0.0.1:9",
		TokenPath:          tokenPath,
		FixturePath:        fix,
		AllowNonFixedPaths: true,
		EnforceAcceptance:  true,
		HTTPClient:         client,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if wantFail {
		if !errors.Is(err, sandboxmeasure.ErrThresholds) {
			t.Fatalf("err=%v", err)
		}
		if rep.Passed || !contains(rep.FailureCodes, "replay_invalid") {
			t.Fatalf("%+v", rep)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
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
	if strings.Contains(err.Error(), "tok-secret") || strings.Contains(strings.ToLower(err.Error()), "bearer") || strings.Contains(err.Error(), real) {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func TestTokenDirSymlinkRejected(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.Mkdir(realDir, 0o700); err != nil {
		t.Fatal(err)
	}
	tok := writeToken(t, realDir, "measure.token", "secret-token-value")
	linkDir := filepath.Join(root, "linkdir")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("symlink not supported")
	}
	err := sandboxmeasure.ValidateTokenFileForTest(linkDir, filepath.Join(linkDir, "measure.token"), true)
	if err == nil {
		t.Fatal("expected dir symlink rejection")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), tok) {
		t.Fatalf("leaked: %v", err)
	}
}

func TestTokenFile0644Rejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "measure.token")
	if err := os.WriteFile(path, []byte("tok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := sandboxmeasure.ValidateTokenFileForTest(dir, path, true); err == nil {
		t.Fatal("expected 0644 rejection")
	}
}

func TestTokenRegular0600Accepted(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := writeToken(t, dir, "measure.token", "tok")
	if err := sandboxmeasure.ValidateTokenFileForTest(dir, path, true); err != nil {
		t.Fatal(err)
	}
}

func TestTokenDirGroupPermsRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	path := writeToken(t, dir, "measure.token", "tok")
	if err := sandboxmeasure.ValidateTokenFileForTest(dir, path, true); err == nil {
		t.Fatal("expected group-writable/readable dir rejection")
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
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if !errors.Is(err, sandboxmeasure.ErrTransport) {
		t.Fatalf("err=%v", err)
	}
	if rep.TransportErrors < 1 {
		t.Fatalf("%+v", rep)
	}
}

func TestReportNoSecrets(t *testing.T) {
	rep := sandboxmeasure.Report{
		Profile: "burst", Attempted: 1, HTTPResponses: 1, Status201: 1,
		Passed: true, FailureCodes: []string{},
	}
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
	if !strings.Contains(out, `"passed":true`) || !strings.Contains(out, `"failure_codes"`) {
		t.Fatalf("missing passed/failure_codes: %s", out)
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

func contains(codes []string, c string) bool {
	for _, x := range codes {
		if x == c {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
