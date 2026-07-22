// Package sandboxmeasure implements closed S3C1 measurement profiles against loopback :18080.
package sandboxmeasure

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// FixedBaseURL is the only allowed target for S3C1 measurement.
	FixedBaseURL = "http://127.0.0.1:18080"
	// FixedTokenDir is the only allowed token directory.
	FixedTokenDir = "/var/lib/bwb-fiscal-admin/tokens"
	// FixedTokenFile is the measure credential path (regular file, not symlink).
	FixedTokenFile = FixedTokenDir + "/measure.token"
	// FixedFixtureName is the synthetic create-document fixture basename.
	FixedFixtureName = "create-document.min.json"

	maxResponseBody = 64 << 10
	httpTimeout     = 10 * time.Second
)

// Profile is a closed measurement profile.
type Profile string

const (
	ProfileSustained Profile = "sustained"
	ProfileBurst     Profile = "burst"
	ProfileReplay    Profile = "replay"
)

// ProfileSpec holds immutable closed limits for a profile.
type ProfileSpec struct {
	Name         Profile
	Total        int
	Concurrency  int
	RatePerSec   float64 // sustained only; 0 = no pacing
	WallMin      time.Duration
	WallMax      time.Duration
	OverallLimit time.Duration
}

// Spec returns the closed spec for a profile.
func Spec(p Profile) (ProfileSpec, error) {
	switch p {
	case ProfileSustained:
		return ProfileSpec{
			Name:         ProfileSustained,
			Total:        300,
			Concurrency:  1,
			RatePerSec:   10,
			WallMin:      28 * time.Second,
			WallMax:      33 * time.Second,
			OverallLimit: 45 * time.Second,
		}, nil
	case ProfileBurst:
		return ProfileSpec{
			Name:         ProfileBurst,
			Total:        60,
			Concurrency:  5,
			OverallLimit: 30 * time.Second,
		}, nil
	case ProfileReplay:
		return ProfileSpec{
			Name:         ProfileReplay,
			Total:        2,
			Concurrency:  1,
			OverallLimit: 30 * time.Second,
		}, nil
	default:
		return ProfileSpec{}, fmt.Errorf("sandboxmeasure: unknown profile %q", p)
	}
}

// ParseProfile validates a profile name.
func ParseProfile(s string) (Profile, error) {
	p := Profile(strings.TrimSpace(s))
	if _, err := Spec(p); err != nil {
		return "", err
	}
	return p, nil
}

// Clock abstracts time for tests.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	NewTimer(d time.Duration) Timer
}

// Timer is a one-shot timer.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
}

type realClock struct{}

func (realClock) Now() time.Time                  { return time.Now() }
func (realClock) Since(t time.Time) time.Duration { return time.Since(t) }
func (realClock) NewTimer(d time.Duration) Timer  { return realTimer{time.NewTimer(d)} }

type realTimer struct{ t *time.Timer }

func (r realTimer) C() <-chan time.Time { return r.t.C }
func (r realTimer) Stop() bool          { return r.t.Stop() }

// Config for a measurement run. Production CLI fills fixed paths only.
type Config struct {
	Profile     Profile
	BaseURL     string // must be FixedBaseURL in production
	TokenPath   string
	FixturePath string
	Clock       Clock
	HTTPClient  *http.Client
	// AllowNonFixedPaths enables test-only overrides of URL/token/fixture.
	AllowNonFixedPaths bool
	// EnforceAcceptance applies sustained wall/throughput gates (CLI production).
	EnforceAcceptance bool
}

// Report is the sanitised machine-readable result (no secrets).
type Report struct {
	Profile            string  `json:"profile"`
	Sent               int     `json:"sent"`
	Status201          int     `json:"status_201"`
	Status409          int     `json:"status_409"`
	Status429          int     `json:"status_429"`
	Status5xx          int     `json:"status_5xx"`
	StatusOther        int     `json:"status_other"`
	DurationMS         int64   `json:"duration_ms"`
	RequestThroughput  float64 `json:"request_throughput_rps"`
	AcceptedThroughput float64 `json:"accepted_throughput_rps"`
	P50MS201           int64   `json:"p50_ms_201"`
	P95MS201           int64   `json:"p95_ms_201"`
	P99MS201           int64   `json:"p99_ms_201"`
	P50MS429           int64   `json:"p50_ms_429"`
	P95MS429           int64   `json:"p95_ms_429"`
	P99MS429           int64   `json:"p99_ms_429"`
	ReplayIdentical    bool    `json:"replay_identical,omitempty"`
}

type class int

const (
	class201 class = iota
	class409
	class429
	class5xx
	classOther
)

type sample struct {
	class class
	us    int64
	body  []byte // only retained for replay compare; never logged
}

// Run executes a closed profile.
func Run(ctx context.Context, cfg Config) (Report, error) {
	spec, err := Spec(cfg.Profile)
	if err != nil {
		return Report{}, err
	}
	if err := cfg.validate(); err != nil {
		return Report{}, err
	}
	clk := cfg.Clock
	if clk == nil {
		clk = realClock{}
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: httpTimeout}
	}

	token, err := readTokenFile(cfg.TokenPath)
	if err != nil {
		return Report{}, err
	}
	defer clearBytes(token)

	fixture, err := loadFixture(cfg.FixturePath)
	if err != nil {
		return Report{}, err
	}

	runCtx, cancel := context.WithTimeout(ctx, spec.OverallLimit)
	defer cancel()

	start := clk.Now()
	var samples []sample
	var seq atomic.Uint64

	switch cfg.Profile {
	case ProfileSustained:
		samples, err = runSustained(runCtx, clk, client, cfg.BaseURL, token, fixture, spec, &seq)
	case ProfileBurst:
		samples, err = runBurst(runCtx, client, cfg.BaseURL, token, fixture, spec, &seq)
	case ProfileReplay:
		samples, err = runReplay(runCtx, client, cfg.BaseURL, token, fixture, &seq)
	}
	if err != nil {
		return Report{}, sanitizeErr(err)
	}
	dur := clk.Since(start)
	rep := buildReport(cfg.Profile, samples, dur)
	if cfg.Profile == ProfileReplay {
		if len(samples) != 2 || samples[0].class != class201 || samples[1].class != class201 {
			return rep, fmt.Errorf("sandboxmeasure: replay expected two 201 responses")
		}
		if !bytes.Equal(stableReplayPayload(samples[0].body), stableReplayPayload(samples[1].body)) {
			return rep, fmt.Errorf("sandboxmeasure: replay payloads differ")
		}
		rep.ReplayIdentical = true
	}
	if cfg.Profile == ProfileSustained && cfg.EnforceAcceptance {
		if dur < spec.WallMin || dur > spec.WallMax {
			return rep, fmt.Errorf("sandboxmeasure: sustained wall duration out of range")
		}
		if rep.RequestThroughput < 9.0 || rep.RequestThroughput > 11.0 {
			return rep, fmt.Errorf("sandboxmeasure: sustained request throughput out of range")
		}
	}
	return rep, nil
}

func (c Config) validate() error {
	if !c.AllowNonFixedPaths {
		if c.BaseURL != FixedBaseURL {
			return fmt.Errorf("sandboxmeasure: base URL is not controllable")
		}
		if c.TokenPath != FixedTokenFile {
			return fmt.Errorf("sandboxmeasure: token path is not controllable")
		}
	}
	if c.TokenPath == "" || c.FixturePath == "" || c.BaseURL == "" {
		return fmt.Errorf("sandboxmeasure: incomplete config")
	}
	return nil
}

func runSustained(ctx context.Context, clk Clock, client *http.Client, base string, token, fixture []byte, spec ProfileSpec, seq *atomic.Uint64) ([]sample, error) {
	out := make([]sample, 0, spec.Total)
	interval := time.Duration(float64(time.Second) / spec.RatePerSec)
	t0 := clk.Now()
	for i := 0; i < spec.Total; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		target := t0.Add(time.Duration(i) * interval)
		now := clk.Now()
		if wait := target.Sub(now); wait > 0 {
			tm := clk.NewTimer(wait)
			select {
			case <-ctx.Done():
				tm.Stop()
				return out, ctx.Err()
			case <-tm.C():
			}
		}
		// No catch-up burst: if late, send immediately and continue agenda without compressing future gaps.
		s, err := doOne(ctx, client, base, token, fixture, seq.Add(1), false)
		if err != nil {
			return out, err
		}
		out = append(out, s)
	}
	return out, nil
}

func runBurst(ctx context.Context, client *http.Client, base string, token, fixture []byte, spec ProfileSpec, seq *atomic.Uint64) ([]sample, error) {
	type result struct {
		s   sample
		err error
	}
	ch := make(chan result, spec.Total)
	sem := make(chan struct{}, spec.Concurrency)
	for i := 0; i < spec.Total; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sem <- struct{}{}
		n := seq.Add(1)
		go func(n uint64) {
			defer func() { <-sem }()
			s, err := doOne(ctx, client, base, token, fixture, n, false)
			ch <- result{s: s, err: err}
		}(n)
	}
	out := make([]sample, 0, spec.Total)
	for i := 0; i < spec.Total; i++ {
		r := <-ch
		if r.err != nil {
			return out, r.err
		}
		out = append(out, r.s)
	}
	return out, nil
}

func runReplay(ctx context.Context, client *http.Client, base string, token, fixture []byte, seq *atomic.Uint64) ([]sample, error) {
	n := seq.Add(1)
	ext, idem, err := uniqueKeys(n)
	if err != nil {
		return nil, err
	}
	body, err := mutateFixture(fixture, ext)
	if err != nil {
		return nil, err
	}
	s1, err := doOnePrepared(ctx, client, base, token, body, idem, true)
	if err != nil {
		return nil, err
	}
	s2, err := doOnePrepared(ctx, client, base, token, body, idem, true)
	if err != nil {
		return nil, err
	}
	return []sample{s1, s2}, nil
}

func doOne(ctx context.Context, client *http.Client, base string, token, fixture []byte, n uint64, keepBody bool) (sample, error) {
	ext, idem, err := uniqueKeys(n)
	if err != nil {
		return sample{}, err
	}
	body, err := mutateFixture(fixture, ext)
	if err != nil {
		return sample{}, err
	}
	return doOnePrepared(ctx, client, base, token, body, idem, keepBody)
}

func doOnePrepared(ctx context.Context, client *http.Client, base string, token, body []byte, idem string, keepBody bool) (sample, error) {
	url := strings.TrimRight(base, "/") + "/v1/documents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return sample{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+string(token))
	req.Header.Set("Idempotency-Key", idem)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Microseconds()
	if err != nil {
		return sample{}, err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBody+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return sample{}, err
	}
	if len(raw) > maxResponseBody {
		return sample{}, fmt.Errorf("sandboxmeasure: response body too large")
	}
	cl := classify(resp.StatusCode)
	s := sample{class: cl, us: latency}
	if keepBody {
		s.body = append([]byte(nil), raw...)
	}
	return s, nil
}

func classify(code int) class {
	switch {
	case code == http.StatusCreated:
		return class201
	case code == http.StatusConflict:
		return class409
	case code == http.StatusTooManyRequests:
		return class429
	case code >= 500 && code <= 599:
		return class5xx
	default:
		return classOther
	}
}

func buildReport(profile Profile, samples []sample, dur time.Duration) Report {
	var c201, c409, c429, c5xx, cOther int
	var lat201, lat429 []int64
	for _, s := range samples {
		switch s.class {
		case class201:
			c201++
			lat201 = append(lat201, s.us)
		case class409:
			c409++
		case class429:
			c429++
			lat429 = append(lat429, s.us)
		case class5xx:
			c5xx++
		default:
			cOther++
		}
	}
	sec := dur.Seconds()
	if sec <= 0 {
		sec = 1e-9
	}
	rep := Report{
		Profile:            string(profile),
		Sent:               len(samples),
		Status201:          c201,
		Status409:          c409,
		Status429:          c429,
		Status5xx:          c5xx,
		StatusOther:        cOther,
		DurationMS:         dur.Milliseconds(),
		RequestThroughput:  float64(len(samples)) / sec,
		AcceptedThroughput: float64(c201) / sec,
		P50MS201:           usToMS(NearestRank(lat201, 50)),
		P95MS201:           usToMS(NearestRank(lat201, 95)),
		P99MS201:           usToMS(NearestRank(lat201, 99)),
		P50MS429:           usToMS(NearestRank(lat429, 50)),
		P95MS429:           usToMS(NearestRank(lat429, 95)),
		P99MS429:           usToMS(NearestRank(lat429, 99)),
	}
	return rep
}

func usToMS(us int64) int64 {
	if us <= 0 {
		return 0
	}
	return (us + 500) / 1000
}

// NearestRank computes percentile with nearest-rank method on a copy of values (µs).
// percent in 0..100; empty returns 0.
func NearestRank(values []int64, percent int) int64 {
	if len(values) == 0 {
		return 0
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	cp := append([]int64(nil), values...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	if percent == 0 {
		return cp[0]
	}
	// nearest-rank: rank = ceil(p/100 * N) (1-based)
	rank := (percent*len(cp) + 99) / 100
	if rank < 1 {
		rank = 1
	}
	if rank > len(cp) {
		rank = len(cp)
	}
	return cp[rank-1]
}

func uniqueKeys(n uint64) (externalID, idempotencyKey string, err error) {
	var rnd [8]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", "", err
	}
	externalID = fmt.Sprintf("S3C-MEAS-%016x-%s", n, hex.EncodeToString(rnd[:]))

	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return "", "", err
	}
	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80
	idempotencyKey = fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
	return externalID, idempotencyKey, nil
}

func loadFixture(path string) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: fixture unavailable")
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("sandboxmeasure: fixture must be a regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: fixture unavailable")
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("sandboxmeasure: fixture malformed")
	}
	return raw, nil
}

func mutateFixture(fixture []byte, externalID string) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(fixture, &m); err != nil {
		return nil, fmt.Errorf("sandboxmeasure: fixture malformed")
	}
	b, err := json.Marshal(externalID)
	if err != nil {
		return nil, err
	}
	m["external_id"] = b
	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func stableReplayPayload(raw []byte) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	// Compare stable fiscal fields only (ignore request_id if present at top level).
	keys := []string{"id", "external_id", "status", "submission_id", "created_at"}
	stable := make(map[string]json.RawMessage, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			stable[k] = v
		}
	}
	b, err := json.Marshal(stable)
	if err != nil {
		return nil
	}
	return b
}

func readTokenFile(path string) ([]byte, error) {
	if err := validateTokenPath(path); err != nil {
		return nil, err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
		return nil, fmt.Errorf("sandboxmeasure: token must be a regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	tok := bytes.TrimSpace(raw)
	if len(tok) == 0 {
		return nil, fmt.Errorf("sandboxmeasure: token unavailable")
	}
	return tok, nil
}

func validateTokenPath(path string) error {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	if base == "" || base == "." || base == ".." || strings.Contains(path, "..") {
		return fmt.Errorf("sandboxmeasure: token path invalid")
	}
	return nil
}

func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func sanitizeErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "bearer") || strings.Contains(lower, "bwb_sbox_") ||
		strings.Contains(lower, "postgres://") || strings.Contains(lower, "fixture-nif") {
		return fmt.Errorf("sandboxmeasure: request failed")
	}
	return err
}

// WriteReport writes a single-line machine-readable report to w.
func WriteReport(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

// DefaultFixturePath resolves fixtures/sandbox/create-document.min.json next to the binary.
func DefaultFixturePath(argv0 string) (string, error) {
	abs, err := filepath.Abs(argv0)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	p := filepath.Join(dir, "fixtures", "sandbox", FixedFixtureName)
	return p, nil
}
