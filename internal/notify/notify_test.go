package notify

import (
	"context"
	"testing"
	"time"

	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

func TestMockNotifier(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	n := NewMockNotifier(s)

	if n.VAPIDPublicKey() == "" {
		t.Fatal("expected non-empty VAPID public key")
	}

	deadline := time.Now().Add(24 * time.Hour)
	if err := n.NotifyTurn(ctx, "p1", "room-1", "The Navigator", deadline); err != nil {
		t.Fatalf("NotifyTurn failed: %v", err)
	}

	if err := n.NotifyDeadlineWarning(ctx, "p1", "room-1", 2); err != nil {
		t.Fatalf("NotifyDeadlineWarning failed: %v", err)
	}

	if err := n.NotifyMatchEnded(ctx, "p1", "room-1", true); err != nil {
		t.Fatalf("NotifyMatchEnded failed: %v", err)
	}

	if err := n.NotifyAFKWarning(ctx, "p1", "room-1"); err != nil {
		t.Fatalf("NotifyAFKWarning failed: %v", err)
	}

	sent := n.GetSentNotifications("p1")
	if len(sent) != 4 {
		t.Fatalf("expected 4 notifications for p1, got %d", len(sent))
	}
	if sent[0].Type != "turn" || sent[1].Type != "deadline_warning" || sent[2].Type != "match_ended" || sent[3].Type != "afk_warning" {
		t.Fatalf("unexpected notification types: %+v", sent)
	}

	n.Clear()
	if len(n.GetSentNotifications("p1")) != 0 {
		t.Fatal("expected 0 notifications after clear")
	}
}

func TestWebPushNotifier(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	_ = s.SavePushSubscription(ctx, store.PushSubscription{
		PlayerID:  "p1",
		Endpoint:  "https://updates.push.services.mozilla.com/wpush/v2/abc",
		P256dh:    "dummy_p256dh",
		Auth:      "dummy_auth",
		Platform:  "web",
		CreatedAt: time.Now(),
	})

	wp, err := NewWebPushNotifier(s)
	if err != nil {
		t.Fatalf("NewWebPushNotifier failed: %v", err)
	}
	if wp.VAPIDPublicKey() == "" {
		t.Fatal("expected valid VAPID public key")
	}

	if err := wp.NotifyTurn(ctx, "p1", "room-1", "The Engineer", time.Now().Add(10*time.Minute)); err != nil {
		t.Fatalf("wp.NotifyTurn failed: %v", err)
	}
}

func TestFCMNotifier(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	_ = s.SavePushSubscription(ctx, store.PushSubscription{
		PlayerID:  "p1",
		Endpoint:  "fcm-device-token-12345",
		Platform:  "fcm",
		CreatedAt: time.Now(),
	})

	fcm := NewFCMNotifier(s)
	if err := fcm.NotifyTurn(ctx, "p1", "room-1", "The Hunter", time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("fcm.NotifyTurn failed: %v", err)
	}
}
