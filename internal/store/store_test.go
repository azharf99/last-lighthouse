package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreMatchLifecycle(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	match := MatchRecord{
		ID:             "match-1",
		Status:         "lobby",
		Seed:           42,
		ContentHash:    "hash123",
		PlayerIDs:      []string{"p1", "p2"},
		TurnTimeoutSec: 90,
		CreatedAt:      time.Now(),
	}

	if err := s.CreateMatch(ctx, match); err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	// Duplicate create should fail
	if err := s.CreateMatch(ctx, match); err != ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// Get match
	got, err := s.GetMatch(ctx, "match-1")
	if err != nil {
		t.Fatalf("GetMatch failed: %v", err)
	}
	if got.ID != "match-1" || got.Status != "lobby" || len(got.PlayerIDs) != 2 {
		t.Fatalf("unexpected match: %+v", got)
	}

	// List matches
	list, err := s.ListMatches(ctx, "lobby")
	if err != nil {
		t.Fatalf("ListMatches failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 match, got %d", len(list))
	}

	// Update status
	if err := s.UpdateMatchStatus(ctx, "match-1", "active"); err != nil {
		t.Fatalf("UpdateMatchStatus failed: %v", err)
	}
	got, _ = s.GetMatch(ctx, "match-1")
	if got.Status != "active" {
		t.Fatalf("expected active, got %s", got.Status)
	}
}

func TestMemoryStoreEventsAndSnapshots(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	events := []EventRecord{
		{MatchID: "m1", Seq: 1, Kind: "turn_started", Payload: []byte(`{"turn":1}`), At: time.Now()},
		{MatchID: "m1", Seq: 2, Kind: "moved", Payload: []byte(`{"to":"forest"}`), At: time.Now()},
	}

	if err := s.AppendEvents(ctx, "m1", events); err != nil {
		t.Fatalf("AppendEvents failed: %v", err)
	}

	// Conflict on duplicate/lower seq
	conflict := []EventRecord{
		{MatchID: "m1", Seq: 2, Kind: "moved", Payload: []byte(`{"to":"cave"}`), At: time.Now()},
	}
	if err := s.AppendEvents(ctx, "m1", conflict); err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	// Load all events
	loaded, err := s.LoadEvents(ctx, "m1", 0)
	if err != nil {
		t.Fatalf("LoadEvents failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	// Load delta events
	delta, err := s.LoadEvents(ctx, "m1", 1)
	if err != nil {
		t.Fatalf("LoadEvents after 1 failed: %v", err)
	}
	if len(delta) != 1 || delta[0].Seq != 2 {
		t.Fatalf("unexpected delta: %+v", delta)
	}

	// Snapshot
	if err := s.SaveSnapshot(ctx, "m1", 2, []byte(`{"round":1}`)); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	snap, err := s.LoadLatestSnapshot(ctx, "m1")
	if err != nil {
		t.Fatalf("LoadLatestSnapshot failed: %v", err)
	}
	if snap == nil || snap.Seq != 2 || string(snap.State) != `{"round":1}` {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestMemoryStoreUserAuth(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	user := UserRecord{
		ID:          "u1",
		DisplayName: "Player One",
		GuestToken:  "token-abc",
		CreatedAt:   time.Now(),
	}

	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	byID, err := s.GetUser(ctx, "u1")
	if err != nil || byID.DisplayName != "Player One" {
		t.Fatalf("GetUser failed: %v, %+v", err, byID)
	}

	byToken, err := s.GetUserByToken(ctx, "token-abc")
	if err != nil || byToken.ID != "u1" {
		t.Fatalf("GetUserByToken failed: %v, %+v", err, byToken)
	}

	// Not found
	if _, err := s.GetUser(ctx, "unknown"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
