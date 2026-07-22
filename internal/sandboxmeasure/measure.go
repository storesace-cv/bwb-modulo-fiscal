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

// Sentinel errors (sanitised; safe to print).
var (
	ErrThresholds = errors.New("sandboxmeasure: thresholds failed")
	ErrTransport  = errors.New("sandboxmeasure: transport failed")
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
	RatePerSec   float64
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

// Config for a measurement run.
type Config struct {
	Profile            Profile
	BaseURL            string
	TokenPath          string
	FixturePath        string
	Clock              Clock
	HTTPClient         *http.Client
	AllowNonFixedPaths bool
	EnforceAcceptance  bool
	// SkipTokenOwnerChecks is test-only (cannot chown in unit tests).
	SkipTokenOwnerChecks bool
}

// Report is the sanitised machine-readable result (no secrets).
type Report struct {
	Profile            string   `json:"profile"`
	Passed             bool     `json:"passed"`
	FailureCodes       []string `json:"failure_codes"`
	Attempted          int      `json:"attempted"`
	HTTPResponses      int      `json:"http_responses"`
	TransportErrors    int      `json:"transport_errors"`
	Status201          int      `json:"status_201"`
	Status409          int      `json:"status_409"`
	Status429          int      `json:"status_429"`
	Status5xx          int      `json:"status_5xx"`
	StatusOther        int      `json:"status_other"`
	DurationMS         int64    `json:"duration_ms"`
	RequestThroughput  float64  `json:"request_throughput_rps"`
	AcceptedThroughput float64  `json:"accepted_throughput_rps"`
	P50MS201           int64    `json:"p50_ms_201"`
	P95MS201           int64    `json:"p95_ms_201"`
	P99MS201           int64    `json:"p99_ms_201"`
	P50MS429           int64    `json:"p50_ms_429"`
	P95MS429           int64    `json:"p95_ms_429"`
	P99MS429           int64    `json:"p99_ms_429"`
	ReplayIdentical    bool     `json:"replay_identical,omitempty"`
}

// createDocumentResponse mirrors the sealed_locally success body (typed compare for replay).
type createDocumentResponse struct {
	ID           string `json:"id"`
	ExternalID   string `json:"external_id"`
	Status       string `json:"status"`
	SubmissionID string `json:"submission_id"`
	CreatedAt    string `json:"created_at"`
}

type class int

const (
	class201 class = iota
	class409
	class429
	class5xx
	classOther
	classTransport
)

type sample struct {
	class class
	us    int64
	body  []byte
	resp  *createDocumentResponse // replay only
}

// Run executes a closed profile. On threshold/transport failure it still returns a filled Report
// with Passed=false and sanitised FailureCodes; the error is ErrThresholds or ErrTransport.
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

	token, err := readTokenFile(cfg.TokenPath, !cfg.AllowNonFixedPaths, cfg.SkipTokenOwnerChecks)
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
	var runErr error

	switch cfg.Profile {
	case ProfileSustained:
		samples, runErr = runSustained(runCtx, clk, client, cfg.BaseURL, token, fixture, spec, &seq)
	case ProfileBurst:
		samples, runErr = runBurst(runCtx, clk, client, cfg.BaseURL, token, fixture, spec, &seq)
	case ProfileReplay:
		samples, runErr = runReplay(runCtx, clk, client, cfg.BaseURL, token, fixture, &seq)
	}
	dur := clk.Since(start)
	rep := buildReport(cfg.Profile, samples, dur)

	if runErr != nil {
		rep.Passed = false
		if !containsCode(rep.FailureCodes, "transport_error") {
			rep.FailureCodes = append(rep.FailureCodes, "transport_error")
		}
		return rep, ErrTransport
	}

	if cfg.Profile == ProfileReplay {
		if err := validateReplaySamples(samples); err != nil {
			rep.Passed = false
			rep.FailureCodes = appendUnique(rep.FailureCodes, "replay_invalid")
			if cfg.EnforceAcceptance {
				return rep, ErrThresholds
			}
			return rep, nil
		}
		rep.ReplayIdentical = true
	}

	if cfg.EnforceAcceptance {
		codes := evaluateThresholds(cfg.Profile, rep, dur, spec)
		if len(codes) > 0 {
			rep.Passed = false
			rep.FailureCodes = appendUniqueAll(rep.FailureCodes, codes)
			return rep, ErrThresholds
		}
	}
	rep.Passed = len(rep.FailureCodes) == 0
	return rep, nil
}

func evaluateThresholds(profile Profile, rep Report, dur time.Duration, spec ProfileSpec) []string {
	var codes []string
	switch profile {
	case ProfileSustained:
		if rep.Attempted != 300 {
			codes = append(codes, "attempted_ne_300")
		}
		if rep.Status409 != 0 {
			codes = append(codes, "status_409")
		}
		if rep.Status5xx != 0 {
			codes = append(codes, "status_5xx")
		}
		if rep.StatusOther != 0 {
			codes = append(codes, "status_other")
		}
		if rep.Status429 > 3 {
			codes = append(codes, "status_429_gt_3")
		}
		if rep.P95MS201 > 250 {
			codes = append(codes, "p95_201_gt_250ms")
		}
		if rep.P99MS201 > 500 {
			codes = append(codes, "p99_201_gt_500ms")
		}
		if dur < spec.WallMin || dur > spec.WallMax {
			codes = append(codes, "duration_out_of_range")
		}
		if rep.RequestThroughput < 9.0 || rep.RequestThroughput > 11.0 {
			codes = append(codes, "request_throughput_out_of_range")
		}
		if rep.TransportErrors != 0 {
			codes = append(codes, "transport_error")
		}
	case ProfileBurst:
		if rep.Status201 < 20 || rep.Status201 > 25 {
			codes = append(codes, "burst_201_out_of_range")
		}
		rest := rep.HTTPResponses - rep.Status201
		if rest < 0 || rep.Status429 != rest {
			codes = append(codes, "burst_remainder_not_429")
		}
		if rep.Status409 != 0 {
			codes = append(codes, "status_409")
		}
		if rep.Status5xx != 0 {
			codes = append(codes, "status_5xx")
		}
		if rep.StatusOther != 0 {
			codes = append(codes, "status_other")
		}
		if rep.TransportErrors != 0 {
			codes = append(codes, "transport_error")
		}
	case ProfileReplay:
		if rep.Status201 != 2 || !rep.ReplayIdentical {
			codes = append(codes, "replay_invalid")
		}
		if rep.TransportErrors != 0 {
			codes = append(codes, "transport_error")
		}
	}
	return codes
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

// runSustained paces with nextAt := now+interval after each send (no catch-up of missed slots).
func runSustained(ctx context.Context, clk Clock, client *http.Client, base string, token, fixture []byte, spec ProfileSpec, seq *atomic.Uint64) ([]sample, error) {
	out := make([]sample, 0, spec.Total)
	interval := time.Duration(float64(time.Second) / spec.RatePerSec)
	nextAt := clk.Now()
	for i := 0; i < spec.Total; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		now := clk.Now()
		if wait := nextAt.Sub(now); wait > 0 {
			tm := clk.NewTimer(wait)
			select {
			case <-ctx.Done():
				tm.Stop()
				return out, ctx.Err()
			case <-tm.C():
			}
		}
		s, err := doOne(ctx, clk, client, base, token, fixture, seq.Add(1), false)
		if err != nil {
			out = append(out, s)
			return out, err
		}
		out = append(out, s)
		// Recalculate from current time — never compress overdue slots into a burst.
		nextAt = clk.Now().Add(interval)
	}
	return out, nil
}

func runBurst(ctx context.Context, clk Clock, client *http.Client, base string, token, fixture []byte, spec ProfileSpec, seq *atomic.Uint64) ([]sample, error) {
	type result struct {
		s   sample
		err error
	}
	ch := make(chan result, spec.Total)
	sem := make(chan struct{}, spec.Concurrency)
	ctxRun, cancel := context.WithCancel(ctx)
	defer cancel()

	var started atomic.Int64
	for i := 0; i < spec.Total; i++ {
		if ctxRun.Err() != nil {
			break
		}
		sem <- struct{}{}
		n := seq.Add(1)
		started.Add(1)
		go func(n uint64) {
			defer func() { <-sem }()
			s, err := doOne(ctxRun, clk, client, base, token, fixture, n, false)
			ch <- result{s: s, err: err}
			if err != nil {
				cancel()
			}
		}(n)
	}
	nStarted := int(started.Load())
	out := make([]sample, 0, nStarted)
	var firstErr error
	for i := 0; i < nStarted; i++ {
		r := <-ch
		out = append(out, r.s)
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
	}
	if firstErr == nil && nStarted < spec.Total {
		if err := ctx.Err(); err != nil {
			return out, err
		}
	}
	return out, firstErr
}

func runReplay(ctx context.Context, clk Clock, client *http.Client, base string, token, fixture []byte, seq *atomic.Uint64) ([]sample, error) {
	n := seq.Add(1)
	ext, idem, err := uniqueKeys(n)
	if err != nil {
		return nil, err
	}
	body, err := mutateFixture(fixture, ext)
	if err != nil {
		return nil, err
	}
	s1, err := doOnePrepared(ctx, clk, client, base, token, body, idem, true)
	if err != nil {
		return []sample{s1}, err
	}
	s2, err := doOnePrepared(ctx, clk, client, base, token, body, idem, true)
	if err != nil {
		return []sample{s1, s2}, err
	}
	return []sample{s1, s2}, nil
}

func doOne(ctx context.Context, clk Clock, client *http.Client, base string, token, fixture []byte, n uint64, keepBody bool) (sample, error) {
	ext, idem, err := uniqueKeys(n)
	if err != nil {
		return sample{class: classTransport}, err
	}
	body, err := mutateFixture(fixture, ext)
	if err != nil {
		return sample{class: classTransport}, err
	}
	return doOnePrepared(ctx, clk, client, base, token, body, idem, keepBody)
}

func doOnePrepared(ctx context.Context, clk Clock, client *http.Client, base string, token, body []byte, idem string, keepBody bool) (sample, error) {
	url := strings.TrimRight(base, "/") + "/v1/documents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return sample{class: classTransport}, ErrTransport
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+string(token))
	req.Header.Set("Idempotency-Key", idem)

	start := clk.Now()
	resp, err := client.Do(req)
	latency := clk.Since(start).Microseconds()
	if err != nil {
		return sample{class: classTransport, us: latency}, ErrTransport
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBody+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return sample{class: classTransport, us: latency}, ErrTransport
	}
	if len(raw) > maxResponseBody {
		return sample{class: classTransport, us: latency}, ErrTransport
	}
	cl := classify(resp.StatusCode)
	s := sample{class: cl, us: latency}
	if keepBody {
		s.body = append([]byte(nil), raw...)
		// Parse failures are not transport errors: validateReplaySamples reports them.
		if cl == class201 {
			if parsed, perr := parseReplayBody(raw); perr == nil {
				s.resp = &parsed
			}
		}
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
	var c201, c409, c429, c5xx, cOther, cTransport int
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
		case classTransport:
			cTransport++
		default:
			cOther++
		}
	}
	httpN := c201 + c409 + c429 + c5xx + cOther
	sec := dur.Seconds()
	if sec <= 0 {
		sec = 1e-9
	}
	return Report{
		Profile:            string(profile),
		FailureCodes:       []string{},
		Attempted:          len(samples),
		HTTPResponses:      httpN,
		TransportErrors:    cTransport,
		Status201:          c201,
		Status409:          c409,
		Status429:          c429,
		Status5xx:          c5xx,
		StatusOther:        cOther,
		DurationMS:         dur.Milliseconds(),
		RequestThroughput:  float64(httpN) / sec,
		AcceptedThroughput: float64(c201) / sec,
		P50MS201:           usToMS(NearestRank(lat201, 50)),
		P95MS201:           usToMS(NearestRank(lat201, 95)),
		P99MS201:           usToMS(NearestRank(lat201, 99)),
		P50MS429:           usToMS(NearestRank(lat429, 50)),
		P95MS429:           usToMS(NearestRank(lat429, 95)),
		P99MS429:           usToMS(NearestRank(lat429, 99)),
	}
}

func usToMS(us int64) int64 {
	if us <= 0 {
		return 0
	}
	return (us + 500) / 1000
}

// NearestRank computes percentile with nearest-rank method on a copy of values (µs).
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
	return json.Marshal(m)
}

func parseReplayBody(raw []byte) (createDocumentResponse, error) {
	var empty createDocumentResponse
	if len(bytes.TrimSpace(raw)) == 0 {
		return empty, fmt.Errorf("sandboxmeasure: replay body empty")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return empty, fmt.Errorf("sandboxmeasure: replay body invalid")
	}
	for _, banned := range []string{"fiscal_number", "authority_request_id"} {
		if _, ok := probe[banned]; ok {
			return empty, fmt.Errorf("sandboxmeasure: replay body has prohibited field")
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var out createDocumentResponse
	if err := dec.Decode(&out); err != nil {
		return empty, fmt.Errorf("sandboxmeasure: replay body invalid")
	}
	if dec.More() {
		return empty, fmt.Errorf("sandboxmeasure: replay body trailing data")
	}
	if strings.TrimSpace(out.ID) == "" || strings.TrimSpace(out.ExternalID) == "" ||
		strings.TrimSpace(out.SubmissionID) == "" || strings.TrimSpace(out.CreatedAt) == "" {
		return empty, fmt.Errorf("sandboxmeasure: replay body missing required fields")
	}
	if out.Status != "sealed_locally" {
		return empty, fmt.Errorf("sandboxmeasure: replay status invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, out.CreatedAt); err != nil {
		if _, err2 := time.Parse(time.RFC3339, out.CreatedAt); err2 != nil {
			return empty, fmt.Errorf("sandboxmeasure: replay created_at invalid")
		}
	}
	return out, nil
}

func validateReplaySamples(samples []sample) error {
	if len(samples) != 2 {
		return fmt.Errorf("sandboxmeasure: replay expected two samples")
	}
	if samples[0].class != class201 || samples[1].class != class201 {
		return fmt.Errorf("sandboxmeasure: replay expected two 201")
	}
	if samples[0].resp == nil || samples[1].resp == nil {
		return fmt.Errorf("sandboxmeasure: replay body invalid")
	}
	a, b := *samples[0].resp, *samples[1].resp
	if a != b {
		return fmt.Errorf("sandboxmeasure: replay payloads differ")
	}
	return nil
}

func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func containsCode(codes []string, c string) bool {
	for _, x := range codes {
		if x == c {
			return true
		}
	}
	return false
}

func appendUnique(codes []string, c string) []string {
	if containsCode(codes, c) {
		return codes
	}
	return append(codes, c)
}

func appendUniqueAll(dst, src []string) []string {
	for _, c := range src {
		dst = appendUnique(dst, c)
	}
	return dst
}

// WriteReport writes a single-line machine-readable report to w.
func WriteReport(w io.Writer, r Report) error {
	if r.FailureCodes == nil {
		r.FailureCodes = []string{}
	}
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
	return filepath.Join(filepath.Dir(abs), "fixtures", "sandbox", FixedFixtureName), nil
}
