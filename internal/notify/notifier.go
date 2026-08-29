package notify

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

// NotificationPayload berisi data yang dikirimkan ke perangkat pengguna (Web Push / FCM / APNs).
type NotificationPayload struct {
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	MatchID   string    `json:"matchId,omitempty"`
	Type      string    `json:"type"` // "turn", "deadline_warning", "match_ended", "afk_warning"
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

// Notifier adalah antarmuka pengiriman notifikasi dorong ke berbagai platform (ADR-007).
type Notifier interface {
	NotifyTurn(ctx context.Context, playerID, matchID, roleName string, deadline time.Time) error
	NotifyDeadlineWarning(ctx context.Context, playerID, matchID string, hoursLeft int) error
	NotifyMatchEnded(ctx context.Context, playerID, matchID string, won bool) error
	NotifyAFKWarning(ctx context.Context, playerID, matchID string) error
	VAPIDPublicKey() string
}

// MockNotifier menyimpan notifikasi terkirim dalam memori untuk pengujian & verifikasi.
type MockNotifier struct {
	mu            sync.RWMutex
	store         store.Store
	notifications []struct {
		PlayerID string
		Payload  NotificationPayload
	}
}

func NewMockNotifier(s store.Store) *MockNotifier {
	return &MockNotifier{
		store: s,
	}
}

func (m *MockNotifier) VAPIDPublicKey() string {
	return "BNxM9Q8_mock_vapid_public_key_for_testing_last_lighthouse_webpush"
}

func (m *MockNotifier) send(playerID string, payload NotificationPayload) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, struct {
		PlayerID string
		Payload  NotificationPayload
	}{PlayerID: playerID, Payload: payload})
}

func (m *MockNotifier) GetSentNotifications(playerID string) []NotificationPayload {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []NotificationPayload
	for _, n := range m.notifications {
		if playerID == "" || n.PlayerID == playerID {
			out = append(out, n.Payload)
		}
	}
	return out
}

func (m *MockNotifier) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = nil
}

func (m *MockNotifier) NotifyTurn(ctx context.Context, playerID, matchID, roleName string, deadline time.Time) error {
	body := fmt.Sprintf("Giliranmu telah tiba sebagai %s di Room %s!", roleName, matchID)
	if !deadline.IsZero() {
		dur := time.Until(deadline).Round(time.Minute)
		if dur > 0 {
			body += fmt.Sprintf(" Sisa waktu: %v.", dur)
		}
	}
	m.send(playerID, NotificationPayload{
		Title:     "The Last Lighthouse — Giliranmu!",
		Body:      body,
		MatchID:   matchID,
		Type:      "turn",
		Timestamp: time.Now(),
	})
	return nil
}

func (m *MockNotifier) NotifyDeadlineWarning(ctx context.Context, playerID, matchID string, hoursLeft int) error {
	m.send(playerID, NotificationPayload{
		Title:     "Peringatan Batas Waktu Giliran",
		Body:      fmt.Sprintf("Tersisa %d jam untuk menyelesaikan giliranmu di Room %s sebelum bot mengambil alih.", hoursLeft, matchID),
		MatchID:   matchID,
		Type:      "deadline_warning",
		Timestamp: time.Now(),
	})
	return nil
}

func (m *MockNotifier) NotifyMatchEnded(ctx context.Context, playerID, matchID string, won bool) error {
	status := "Kalah menghadapi kegelapan"
	if won {
		status = "Menang! Mercusuar terakhir berhasil dinyalakan!"
	}
	m.send(playerID, NotificationPayload{
		Title:     "Match Telah Selesai",
		Body:      fmt.Sprintf("Permainan di Room %s telah selesai: %s.", matchID, status),
		MatchID:   matchID,
		Type:      "match_ended",
		Timestamp: time.Now(),
	})
	return nil
}

func (m *MockNotifier) NotifyAFKWarning(ctx context.Context, playerID, matchID string) error {
	m.send(playerID, NotificationPayload{
		Title:     "Pemain AFK — Bot Mengambil Alih",
		Body:      fmt.Sprintf("Kamu melewatkan giliran berturut-turut di Room %s. Bot mengambil alih giliranmu. Kamu bisa masuk kembali kapan saja.", matchID),
		MatchID:   matchID,
		Type:      "afk_warning",
		Timestamp: time.Now(),
	})
	return nil
}
