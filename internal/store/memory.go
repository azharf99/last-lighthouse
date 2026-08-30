package store

import (
	"context"
	"sort"
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
	leaderboard   []LeaderboardEntry
}

func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{
		matches:       make(map[string]MatchRecord),
		events:        make(map[string][]EventRecord),
		snapshots:     make(map[string][]SnapshotRecord),
		users:         make(map[string]UserRecord),
		tokens:        make(map[string]string),
		deadlines:     make(map[string]TurnDeadline),
		subscriptions: make(map[string][]PushSubscription),
		leaderboard:   make([]LeaderboardEntry, 0),
	}
	ms.seedBaselineLeaderboard()
	return ms
}

func (m *MemoryStore) seedBaselineLeaderboard() {
	now := time.Now()
	sample := []LeaderboardEntry{
		{
			ID:                    "lb_001",
			PlayerName:            "Kapten Ana",
			Character:             "navigator",
			VP:                    24,
			Darkness:              4,
			Rounds:                6,
			Won:                   true,
			MonstersSlain:         2,
			ComponentsContributed: 3,
			MatchID:               "m_solo_pro",
			CreatedAt:             now.Add(-3 * 24 * time.Hour),
		},
		{
			ID:                    "lb_002",
			PlayerName:            "Mekanik Budi",
			Character:             "engineer",
			VP:                    22,
			Darkness:              5,
			Rounds:                7,
			Won:                   true,
			MonstersSlain:         1,
			ComponentsContributed: 4,
			MatchID:               "m_coop_duo",
			CreatedAt:             now.Add(-2 * 24 * time.Hour),
		},
		{
			ID:                    "lb_003",
			PlayerName:            "Ranger Citra",
			Character:             "hunter",
			VP:                    19,
			Darkness:              6,
			Rounds:                8,
			Won:                   true,
			MonstersSlain:         5,
			ComponentsContributed: 2,
			MatchID:               "m_hunt_01",
			CreatedAt:             now.Add(-1 * 24 * time.Hour),
		},
		{
			ID:                    "lb_004",
			PlayerName:            "Arsiparis Dewi",
			Character:             "scholar",
			VP:                    18,
			Darkness:              7,
			Rounds:                8,
			Won:                   true,
			MonstersSlain:         0,
			ComponentsContributed: 3,
			MatchID:               "m_lore_99",
			CreatedAt:             now.Add(-12 * time.Hour),
		},
		{
			ID:                    "lb_005",
			PlayerName:            "Petualang Eko",
			Character:             "navigator",
			VP:                    15,
			Darkness:              8,
			Rounds:                5,
			Won:                   false,
			MonstersSlain:         2,
			ComponentsContributed: 2,
			MatchID:               "m_dread_01",
			CreatedAt:             now.Add(-6 * time.Hour),
		},
	}
	m.leaderboard = append(m.leaderboard, sample...)
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

// Leaderboard & Pencapaian Skor

func (m *MemoryStore) AddLeaderboardEntry(_ context.Context, entry LeaderboardEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	m.leaderboard = append(m.leaderboard, entry)
	return nil
}

func (m *MemoryStore) GetLeaderboard(_ context.Context, category string, limit int) ([]LeaderboardEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}

	copied := make([]LeaderboardEntry, len(m.leaderboard))
	copy(copied, m.leaderboard)

	switch category {
	case "speed", "rounds":
		// Kemenangan tercepat (hanya yang menang, ronde lebih sedikit lebih baik)
		var wonOnly []LeaderboardEntry
		for _, e := range copied {
			if e.Won {
				wonOnly = append(wonOnly, e)
			}
		}
		sort.SliceStable(wonOnly, func(i, j int) bool {
			if wonOnly[i].Rounds != wonOnly[j].Rounds {
				return wonOnly[i].Rounds < wonOnly[j].Rounds
			}
			return wonOnly[i].VP > wonOnly[j].VP
		})
		if len(wonOnly) > limit {
			wonOnly = wonOnly[:limit]
		}
		return wonOnly, nil

	case "monsters":
		// Pemburu monster terbanyak
		sort.SliceStable(copied, func(i, j int) bool {
			if copied[i].MonstersSlain != copied[j].MonstersSlain {
				return copied[i].MonstersSlain > copied[j].MonstersSlain
			}
			return copied[i].VP > copied[j].VP
		})

	case "vp":
		fallthrough
	default:
		// Skor Victory Points tertinggi
		sort.SliceStable(copied, func(i, j int) bool {
			if copied[i].VP != copied[j].VP {
				return copied[i].VP > copied[j].VP
			}
			if copied[i].Won != copied[j].Won {
				return copied[i].Won
			}
			return copied[i].Rounds < copied[j].Rounds
		})
	}

	if len(copied) > limit {
		copied = copied[:limit]
	}
	return copied, nil
}

func (m *MemoryStore) Close() error {
	return nil
}

