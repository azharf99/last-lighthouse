package store

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu            sync.RWMutex
	matches       map[string]MatchRecord
	events        map[string][]EventRecord
	snapshots     map[string][]SnapshotRecord
	users         map[string]UserRecord
	tokens        map[string]string // token -> userID
	deadlines     map[string]TurnDeadline
	subscriptions map[string][]PushSubscription // playerID -> subscriptions
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		matches:       make(map[string]MatchRecord),
		events:        make(map[string][]EventRecord),
		snapshots:     make(map[string][]SnapshotRecord),
		users:         make(map[string]UserRecord),
		tokens:        make(map[string]string),
		deadlines:     make(map[string]TurnDeadline),
		subscriptions: make(map[string][]PushSubscription),
	}
}

func (m *MemoryStore) CreateMatch(_ context.Context, rec MatchRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.matches[rec.ID]; exists {
		return ErrAlreadyExists
	}
	m.matches[rec.ID] = rec
	return nil
}

func (m *MemoryStore) GetMatch(_ context.Context, id string) (*MatchRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, exists := m.matches[id]
	if !exists {
		return nil, ErrNotFound
	}
	pids := make([]string, len(rec.PlayerIDs))
	copy(pids, rec.PlayerIDs)
	rec.PlayerIDs = pids
	return &rec, nil
}

func (m *MemoryStore) ListMatches(_ context.Context, status string) ([]MatchRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []MatchRecord
	for _, rec := range m.matches {
		if status == "" || rec.Status == status {
			pids := make([]string, len(rec.PlayerIDs))
			copy(pids, rec.PlayerIDs)
			rec.PlayerIDs = pids
			out = append(out, rec)
		}
	}
	return out, nil
}

func (m *MemoryStore) ListPlayerMatches(_ context.Context, playerID string) ([]MatchRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []MatchRecord
	for _, rec := range m.matches {
		isPlayer := false
		for _, pid := range rec.PlayerIDs {
			if pid == playerID {
				isPlayer = true
				break
			}
		}
		if isPlayer {
			pids := make([]string, len(rec.PlayerIDs))
			copy(pids, rec.PlayerIDs)
			rec.PlayerIDs = pids
			out = append(out, rec)
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateMatchStatus(_ context.Context, id string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.matches[id]
	if !exists {
		return ErrNotFound
	}
	rec.Status = status
	if status == "won" || status == "lost" {
		now := time.Now()
		rec.FinishedAt = &now
	}
	m.matches[id] = rec
	return nil
}

func (m *MemoryStore) AppendEvents(_ context.Context, matchID string, events []EventRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.events[matchID]
	for _, e := range events {
		// Verify monotonic sequence
		if len(list) > 0 && e.Seq <= list[len(list)-1].Seq {
			return ErrConflict
		}
		list = append(list, e)
	}
	m.events[matchID] = list
	return nil
}

func (m *MemoryStore) LoadEvents(_ context.Context, matchID string, afterSeq int64) ([]EventRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list, exists := m.events[matchID]
	if !exists {
		return nil, nil
	}

	var out []EventRecord
	for _, e := range list {
		if e.Seq > afterSeq {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *MemoryStore) SaveSnapshot(_ context.Context, matchID string, seq int64, state []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := make([]byte, len(state))
	copy(copied, state)

	m.snapshots[matchID] = append(m.snapshots[matchID], SnapshotRecord{
		MatchID: matchID,
		Seq:     seq,
		State:   copied,
		At:      time.Now(),
	})
	return nil
}

func (m *MemoryStore) LoadLatestSnapshot(_ context.Context, matchID string) (*SnapshotRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list, exists := m.snapshots[matchID]
	if !exists || len(list) == 0 {
		return nil, nil
	}

	last := list[len(list)-1]
	copied := make([]byte, len(last.State))
	copy(copied, last.State)
	last.State = copied
	return &last, nil
}

func (m *MemoryStore) CreateUser(_ context.Context, u UserRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[u.ID]; exists {
		return ErrAlreadyExists
	}
	m.users[u.ID] = u
	if u.GuestToken != "" {
		m.tokens[u.GuestToken] = u.ID
	}
	return nil
}

func (m *MemoryStore) GetUser(_ context.Context, id string) (*UserRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.users[id]
	if !exists {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (m *MemoryStore) GetUserByToken(_ context.Context, token string) (*UserRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uid, exists := m.tokens[token]
	if !exists {
		return nil, ErrNotFound
	}
	u := m.users[uid]
	return &u, nil
}

// Turn Deadline Scheduler (M5 - ADR-007)

func (m *MemoryStore) SetTurnDeadline(_ context.Context, d TurnDeadline) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deadlines[d.MatchID] = d
	return nil
}

func (m *MemoryStore) GetTurnDeadline(_ context.Context, matchID string) (*TurnDeadline, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	d, exists := m.deadlines[matchID]
	if !exists {
		return nil, ErrNotFound
	}
	return &d, nil
}

func (m *MemoryStore) GetExpiredDeadlines(_ context.Context, now time.Time) ([]TurnDeadline, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []TurnDeadline
	for _, d := range m.deadlines {
		if !d.DeadlineAt.IsZero() && d.DeadlineAt.Before(now) {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *MemoryStore) ClearTurnDeadline(_ context.Context, matchID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.deadlines, matchID)
	return nil
}

// Push Notifications (M5 - ADR-007)

func (m *MemoryStore) SavePushSubscription(_ context.Context, sub PushSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscriptions[sub.PlayerID]
	// Replace if endpoint already registered
	found := false
	for i, existing := range subs {
		if existing.Endpoint == sub.Endpoint {
			subs[i] = sub
			found = true
			break
		}
	}
	if !found {
		subs = append(subs, sub)
	}
	m.subscriptions[sub.PlayerID] = subs
	return nil
}

func (m *MemoryStore) GetPushSubscriptions(_ context.Context, playerID string) ([]PushSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs := m.subscriptions[playerID]
	out := make([]PushSubscription, len(subs))
	copy(out, subs)
	return out, nil
}

func (m *MemoryStore) Close() error {
	return nil
}
