package match

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

type Registry struct {
	mu          sync.RWMutex
	actors      map[string]*Actor
	store       store.Store
	content     *core.Content
	turnTimeout time.Duration
}

func NewRegistry(s store.Store, c *core.Content, turnTimeout time.Duration) *Registry {
	if c == nil {
		c = core.DefaultContent()
	}
	return &Registry{
		actors:      make(map[string]*Actor),
		store:       s,
		content:     c,
		turnTimeout: turnTimeout,
	}
}

// CreateMatch initializes a new match, creates a Match Actor, and starts the game.
func (r *Registry) CreateMatch(ctx context.Context, matchID string, seed int64, setups []core.PlayerSetup) (*Actor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.actors[matchID]; exists {
		return nil, errors.New("match actor already active")
	}

	playerIDs := make([]string, len(setups))
	for i, s := range setups {
		playerIDs[i] = string(s.ID)
	}

	rec := store.MatchRecord{
		ID:             matchID,
		Status:         "active",
		Seed:           seed,
		ContentHash:    r.content.Hash,
		PlayerIDs:      playerIDs,
		TurnTimeoutSec: int(r.turnTimeout.Seconds()),
		CreatedAt:      time.Now(),
	}

	if err := r.store.CreateMatch(ctx, rec); err != nil {
		return nil, fmt.Errorf("store create match: %w", err)
	}

	actor := NewActor(matchID, nil, r.content, r.store, r.turnTimeout)
	if err := actor.Start(seed, setups); err != nil {
		actor.Stop()
		return nil, fmt.Errorf("start game: %w", err)
	}

	// Schedule turn deadline for first active player
	if actor.state != nil && len(actor.state.TurnOrder) > 0 {
		firstPlayer := actor.state.TurnOrder[0]
		_ = r.store.SetTurnDeadline(ctx, store.TurnDeadline{
			MatchID:    matchID,
			PlayerID:   string(firstPlayer),
			DeadlineAt: time.Now().Add(r.turnTimeout),
			Missed:     0,
		})
	}

	r.actors[matchID] = actor
	return actor, nil
}

// GetOrLoad fetches an active Match Actor from memory or restores it from the database (ADR-004).
func (r *Registry) GetOrLoad(ctx context.Context, matchID string) (*Actor, error) {
	r.mu.RLock()
	actor, exists := r.actors[matchID]
	r.mu.RUnlock()
	if exists {
		return actor, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check after lock
	if actor, exists := r.actors[matchID]; exists {
		return actor, nil
	}

	rec, err := r.store.GetMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// Restore state from snapshot + replayed events
	snap, err := r.store.LoadLatestSnapshot(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	var st core.State
	var afterSeq int64 = 0

	if snap != nil {
		if err := json.Unmarshal(snap.State, &st); err != nil {
			return nil, fmt.Errorf("unmarshal state snapshot: %w", err)
		}
		afterSeq = snap.Seq
	}

	// Load trailing events
	records, err := r.store.LoadEvents(ctx, matchID, afterSeq)
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}

	var events []core.Event
	for _, r := range records {
		var ev core.Event
		if err := json.Unmarshal(r.Payload, &ev); err != nil {
			return nil, fmt.Errorf("unmarshal event: %w", err)
		}
		events = append(events, ev)
	}

	if len(events) > 0 {
		core.ApplyAll(&st, events)
	}

	lastSeq := afterSeq
	if len(records) > 0 {
		lastSeq = records[len(records)-1].Seq
	}

	timeout := time.Duration(rec.TurnTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = r.turnTimeout
	}

	actor = NewActor(matchID, &st, r.content, r.store, timeout)
	actor.eventSeq = lastSeq

	r.actors[matchID] = actor
	return actor, nil
}

// RemoveActor removes an actor from the active registry.
func (r *Registry) RemoveActor(matchID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if actor, exists := r.actors[matchID]; exists {
		actor.Stop()
		delete(r.actors, matchID)
	}
}

// Shutdown stops all active match actors.
func (r *Registry) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, actor := range r.actors {
		actor.Stop()
		delete(r.actors, id)
	}
}
