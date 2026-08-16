package femock

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

type replayEntry struct {
	payloadHash string
	response    []byte
	status      int
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

// lookup returns cached response, conflict, or miss.
func (s *replayStore) lookup(key, hash string) (resp []byte, status int, conflict, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.byKey[key]
	if !found {
		return nil, 0, false, false
	}
	if e.payloadHash != hash {
		return nil, 0, true, true
	}
	out := make([]byte, len(e.response))
	copy(out, e.response)
	return out, e.status, false, true
}

func (s *replayStore) store(key, hash string, status int, resp []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(resp))
	copy(cp, resp)
	s.byKey[key] = replayEntry{payloadHash: hash, response: cp, status: status}
}

func (s *replayStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey = make(map[string]replayEntry)
}
