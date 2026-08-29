package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrConflict      = errors.New("sequence conflict")
)

type MatchRecord struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"` // lobby, active, won, lost, paused
	Seed           int64      `json:"seed"`
	ContentHash    string     `json:"contentHash"`
	PlayerIDs      []string   `json:"playerIds"`
	TurnTimeoutSec int        `json:"turnTimeoutSec"`
	CreatedAt      time.Time  `json:"createdAt"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
}

type EventRecord struct {
	MatchID string    `json:"matchId"`
	Seq     int64     `json:"seq"`
	Kind    string    `json:"kind"`
	Payload []byte    `json:"payload"` // JSON encoded core.Event
	At      time.Time `json:"at"`
}

type SnapshotRecord struct {
	MatchID string    `json:"matchId"`
	Seq     int64     `json:"seq"`
	State   []byte    `json:"state"` // JSON encoded core.State
	At      time.Time `json:"at"`
}

type UserRecord struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	GuestToken  string    `json:"guestToken"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Store adalah antarmuka penyimpanan persisten untuk server Last Lighthouse (ADR-004).
// Menangani metadata match, event log append-only, snapshot, dan identitas user.
type Store interface {
	// Match metadata
	CreateMatch(ctx context.Context, m MatchRecord) error
	GetMatch(ctx context.Context, id string) (*MatchRecord, error)
	ListMatches(ctx context.Context, status string) ([]MatchRecord, error)
	UpdateMatchStatus(ctx context.Context, id string, status string) error

	// Event log — sumber kebenaran (append-only)
	AppendEvents(ctx context.Context, matchID string, events []EventRecord) error
	LoadEvents(ctx context.Context, matchID string, afterSeq int64) ([]EventRecord, error)

	// Snapshot — recovery cepat (setiap 20 event)
	SaveSnapshot(ctx context.Context, matchID string, seq int64, state []byte) error
	LoadLatestSnapshot(ctx context.Context, matchID string) (*SnapshotRecord, error)

	// User & auth
	CreateUser(ctx context.Context, u UserRecord) error
	GetUser(ctx context.Context, id string) (*UserRecord, error)
	GetUserByToken(ctx context.Context, token string) (*UserRecord, error)

	Close() error
}
