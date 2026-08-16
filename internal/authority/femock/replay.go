package femock

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

type replayEntry struct {
	payloadHash string
	status      int
	// functional is the idempotent body without requestID.
	functional map[string]any
}

type replayStore struct {
	mu    sync.Mutex
	byKey map[string]replayEntry
}

func newReplayStore() *replayStore {
	return &replayStore{byKey: make(map[string]replayEntry)}
}

func payloadHash(op, jws string) string {
	sum := sha256.Sum256([]byte(op + "\x00" + jws))
	return hex.EncodeToString(sum[:])
}

func (s *replayStore) lookup(key, hash string) (functional map[string]any, status int, conflict, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.byKey[key]
	if !found {
		return nil, 0, false, false
	}
	if e.payloadHash != hash {
		return nil, 0, true, true
	}
	return cloneMap(e.functional), e.status, false, true
}

func (s *replayStore) store(key, hash string, status int, functional map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[key] = replayEntry{payloadHash: hash, status: status, functional: cloneMap(functional)}
}

func (s *replayStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey = make(map[string]replayEntry)
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	// Round-trip via JSON to deep-copy JSON-safe values.
	b, err := json.Marshal(in)
	if err != nil {
		out := make(map[string]any, len(in))
		for k, v := range in {
			out[k] = v
		}
		return out
	}
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func stripRequestID(m map[string]any) map[string]any {
	out := cloneMap(m)
	delete(out, "requestID")
	return out
}

func withRequestID(functional map[string]any, reqID string) map[string]any {
	out := cloneMap(functional)
	out["requestID"] = reqID
	return out
}
