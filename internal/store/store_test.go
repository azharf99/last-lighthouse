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

func TestMemoryStoreAsyncAndPush(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	// 1. Player matches list
	m1 := MatchRecord{ID: "m1", Status: "active", PlayerIDs: []string{"p1", "p2"}, TurnTimeoutSec: 86400, CreatedAt: time.Now()}
	m2 := MatchRecord{ID: "m2", Status: "active", PlayerIDs: []string{"p2", "p3"}, TurnTimeoutSec: 86400, CreatedAt: time.Now()}
	_ = s.CreateMatch(ctx, m1)
	_ = s.CreateMatch(ctx, m2)

	p1Matches, err := s.ListPlayerMatches(ctx, "p1")
	if err != nil || len(p1Matches) != 1 {
		t.Fatalf("expected 1 match for p1, got %d (err: %v)", len(p1Matches), err)
	}

	p2Matches, err := s.ListPlayerMatches(ctx, "p2")
	if err != nil || len(p2Matches) != 2 {
		t.Fatalf("expected 2 matches for p2, got %d (err: %v)", len(p2Matches), err)
	}

	// 2. Turn deadlines
	now := time.Now()
	expiredDeadline := TurnDeadline{MatchID: "m1", PlayerID: "p1", DeadlineAt: now.Add(-5 * time.Minute), Missed: 0}
	futureDeadline := TurnDeadline{MatchID: "m2", PlayerID: "p2", DeadlineAt: now.Add(24 * time.Hour), Missed: 0}

	if err := s.SetTurnDeadline(ctx, expiredDeadline); err != nil {
		t.Fatalf("SetTurnDeadline failed: %v", err)
	}
	if err := s.SetTurnDeadline(ctx, futureDeadline); err != nil {
		t.Fatalf("SetTurnDeadline failed: %v", err)
	}

	gotD, err := s.GetTurnDeadline(ctx, "m1")
	if err != nil || gotD.PlayerID != "p1" {
		t.Fatalf("GetTurnDeadline failed: %v, %+v", err, gotD)
	}

	expiredList, err := s.GetExpiredDeadlines(ctx, now)
	if err != nil || len(expiredList) != 1 || expiredList[0].MatchID != "m1" {
		t.Fatalf("expected 1 expired deadline for m1, got %d (err: %v)", len(expiredList), err)
	}

	if err := s.ClearTurnDeadline(ctx, "m1"); err != nil {
		t.Fatalf("ClearTurnDeadline failed: %v", err)
	}
	if _, err := s.GetTurnDeadline(ctx, "m1"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after clear, got %v", err)
	}

	// 3. Push subscriptions
	sub1 := PushSubscription{PlayerID: "p1", Endpoint: "https://push.example.com/sub1", P256dh: "key1", Auth: "auth1", Platform: "web", CreatedAt: now}
	sub2 := PushSubscription{PlayerID: "p1", Endpoint: "https://fcm.example.com/sub2", P256dh: "key2", Auth: "auth2", Platform: "fcm", CreatedAt: now}

	if err := s.SavePushSubscription(ctx, sub1); err != nil {
		t.Fatalf("SavePushSubscription failed: %v", err)
	}
	if err := s.SavePushSubscription(ctx, sub2); err != nil {
		t.Fatalf("SavePushSubscription failed: %v", err)
	}

	subs, err := s.GetPushSubscriptions(ctx, "p1")
	if err != nil || len(subs) != 2 {
		t.Fatalf("expected 2 subscriptions for p1, got %d (err: %v)", len(subs), err)
	}
}
