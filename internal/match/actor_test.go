package match

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

type mockSubscriber struct {
	mu       sync.Mutex
	events   [][]core.Event
	snapshot *core.PlayerView
}

func (m *mockSubscriber) SendEvents(matchID string, eventSeq int64, events []core.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, events)
	return nil
}

func (m *mockSubscriber) SendSnapshot(matchID string, eventSeq int64, view *core.PlayerView) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot = view
	return nil
}

func (m *mockSubscriber) SendError(matchID string, code, message string) error {
	return nil
}

func (m *mockSubscriber) Close() error {
	return nil
}

func (m *mockSubscriber) EventBatches() [][]core.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([][]core.Event, len(m.events))
	copy(out, m.events)
	return out
}

func TestActorLifecycleAndCommands(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	c := core.DefaultContent()

	reg := NewRegistry(s, c, 5*time.Second)
	defer reg.Shutdown()

	setups := []core.PlayerSetup{
		{ID: "p1", Name: "Player 1", Character: "navigator"},
		{ID: "p2", Name: "Player 2", Character: "engineer"},
		{ID: "p3", Name: "Player 3", Character: "hunter"},
	}

	actor, err := reg.CreateMatch(ctx, "m1", 42, setups)
	if err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	subP1 := &mockSubscriber{}
	if err := actor.Join("p1", subP1, 0); err != nil {
		t.Fatalf("Join p1 failed: %v", err)
	}

	// Active player sends a legal action
	state := actor.State()
	activePlayer := state.ActivePlayer()
	if activePlayer == nil {
		t.Fatalf("expected active player")
	}

	// Move active player to an adjacent location
	loc := state.Board.Location(activePlayer.At)
	if loc == nil || len(loc.Adjacent) == 0 {
		t.Fatalf("no adjacent location")
	}
	targetLoc := loc.Adjacent[0]

	cmd := core.Command{
		Kind:   core.CmdMove,
		Player: activePlayer.ID,
		To:     targetLoc,
	}

	if err := actor.SubmitCommand(activePlayer.ID, 1, cmd); err != nil {
		t.Fatalf("SubmitCommand failed: %v", err)
	}

	// Duplicate clientSeq should be rejected (idempotency)
	if err := actor.SubmitCommand(activePlayer.ID, 1, cmd); err != ErrDuplicateClient {
		t.Fatalf("expected ErrDuplicateClient, got %v", err)
	}

	// Subscriber p1 should have received events
	batches := subP1.EventBatches()
	if len(batches) == 0 {
		t.Fatalf("expected events sent to p1")
	}
}

func TestActorRecoveryFromStore(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	c := core.DefaultContent()

	reg := NewRegistry(s, c, 5*time.Second)

	setups := []core.PlayerSetup{
		{ID: "p1", Name: "Player 1", Character: "navigator"},
		{ID: "p2", Name: "Player 2", Character: "engineer"},
	}

	actor, err := reg.CreateMatch(ctx, "m2", 123, setups)
	if err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	active := actor.State().ActivePlayer().ID
	adj := actor.State().Board.Location(actor.State().Player(active).At).Adjacent[0]

	_ = actor.SubmitCommand(active, 1, core.Command{Kind: core.CmdMove, Player: active, To: adj})
	firstEventSeq := actor.EventSeq()

	// Shutdown actor in memory
	reg.RemoveActor("m2")

	// Restore actor from store
	recovered, err := reg.GetOrLoad(ctx, "m2")
	if err != nil {
		t.Fatalf("GetOrLoad failed: %v", err)
	}
	if recovered.EventSeq() != firstEventSeq {
		t.Fatalf("expected eventSeq %d, got %d", firstEventSeq, recovered.EventSeq())
	}
	if recovered.State().Player(active).At != adj {
		t.Fatalf("expected player at %s, got %s", adj, recovered.State().Player(active).At)
	}
}

func TestActorTurnTimeoutAutoAction(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	c := core.DefaultContent()

	// 100ms turn timeout for testing
	reg := NewRegistry(s, c, 100*time.Millisecond)
	defer reg.Shutdown()

	setups := []core.PlayerSetup{
		{ID: "p1", Name: "Player 1", Character: "navigator"},
		{ID: "p2", Name: "Player 2", Character: "engineer"},
	}

	actor, err := reg.CreateMatch(ctx, "m3", 456, setups)
	if err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	sub := &mockSubscriber{}
	_ = actor.Join("p1", sub, 0)

	firstActive := actor.State().ActivePlayer().ID

	// Wait 130ms for exactly 1 turn timeout to trigger
	time.Sleep(130 * time.Millisecond)

	secondActive := actor.State().ActivePlayer().ID
	if secondActive == firstActive {
		t.Fatalf("turn timer should have advanced turn to other player (was %s, now %s)", firstActive, secondActive)
	}
}
