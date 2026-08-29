package match

import (
	"context"
	"testing"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/notify"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

func TestDeadlineScheduler(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	notif := notify.NewMockNotifier(s)
	content := core.DefaultContent()
	reg := NewRegistry(s, content, 90*time.Second)

	setups := []core.PlayerSetup{
		{ID: "p1", Name: "Alice", Character: "navigator"},
		{ID: "p2", Name: "Bob", Character: "engineer"},
	}

	actor, err := reg.CreateMatch(ctx, "match-async-1", 42, setups)
	if err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	sched := NewDeadlineScheduler(s, reg, notif, 50*time.Millisecond)

	// Check initial deadline set
	dl, err := s.GetTurnDeadline(ctx, "match-async-1")
	if err != nil {
		t.Fatalf("GetTurnDeadline failed: %v", err)
	}
	if dl.PlayerID != "p1" {
		t.Fatalf("expected p1 active deadline, got %s", dl.PlayerID)
	}

	// 1. Manually expire the deadline
	pastTime := time.Now().Add(-10 * time.Minute)
	dl.DeadlineAt = pastTime
	_ = s.SetTurnDeadline(ctx, *dl)

	// Run CheckDeadlines
	sched.CheckDeadlines(ctx, time.Now())

	// p1 missed 1 turn -> turn should advance to p2
	st := actor.State()
	if st.ActivePlayer().ID != "p2" {
		t.Fatalf("expected active player to advance to p2, got %s", st.ActivePlayer().ID)
	}

	// Check that p2 now has a new deadline
	newDl, err := s.GetTurnDeadline(ctx, "match-async-1")
	if err != nil {
		t.Fatalf("GetTurnDeadline for p2 failed: %v", err)
	}
	if newDl.PlayerID != "p2" {
		t.Fatalf("expected p2 deadline, got %s", newDl.PlayerID)
	}

	// 2. Test AFK Bot Takeover (p2 misses 2 deadlines)
	// First timeout for p2
	newDl.DeadlineAt = pastTime
	_ = s.SetTurnDeadline(ctx, *newDl)
	sched.CheckDeadlines(ctx, time.Now())

	// Second timeout for p2 -> AFK limit reached
	p2Dl, _ := s.GetTurnDeadline(ctx, "match-async-1")
	if p2Dl != nil {
		p2Dl.DeadlineAt = pastTime
		_ = s.SetTurnDeadline(ctx, *p2Dl)
		sched.CheckDeadlines(ctx, time.Now())
	}

	// Verify notification sent
	notifications := notif.GetSentNotifications("p2")
	if len(notifications) == 0 {
		t.Fatal("expected notifications sent to p2")
	}

	// Verify scheduler start/stop clean
	sched.Start()
	time.Sleep(100 * time.Millisecond)
	sched.Stop()
	reg.Shutdown()
}
