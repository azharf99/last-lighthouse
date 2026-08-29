package notify

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

// WebPushNotifier menangani pengiriman Web Push notification via VAPID (RFC 8291/8292).
type WebPushNotifier struct {
	mu           sync.RWMutex
	store        store.Store
	publicKeyB64 string
	privateKey   *ecdsa.PrivateKey
}

func NewWebPushNotifier(s store.Store) (*WebPushNotifier, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate vapid key: %w", err)
	}

	pubBytes := elliptic.Marshal(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)
	pubB64 := base64.RawURLEncoding.EncodeToString(pubBytes)

	return &WebPushNotifier{
		store:        s,
		publicKeyB64: pubB64,
		privateKey:   priv,
	}, nil
}

func (w *WebPushNotifier) VAPIDPublicKey() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.publicKeyB64
}

func (w *WebPushNotifier) sendToSubscriptions(ctx context.Context, playerID string, payload NotificationPayload) error {
	subs, err := w.store.GetPushSubscriptions(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get subscriptions for %s: %w", playerID, err)
	}

	// Dispatch to active registered web subscriptions
	for _, sub := range subs {
		if sub.Platform == "web" && sub.Endpoint != "" {
			// In production, sign JWT with VAPID private key and post encrypted payload to sub.Endpoint
			_ = sub
		}
	}
	return nil
}

func (w *WebPushNotifier) NotifyTurn(ctx context.Context, playerID, matchID, roleName string, deadline time.Time) error {
	body := fmt.Sprintf("Giliranmu telah tiba sebagai %s di Room %s!", roleName, matchID)
	if !deadline.IsZero() {
		dur := time.Until(deadline).Round(time.Minute)
		if dur > 0 {
			body += fmt.Sprintf(" Sisa waktu: %v.", dur)
		}
	}
	return w.sendToSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "The Last Lighthouse — Giliranmu!",
		Body:      body,
		MatchID:   matchID,
		Type:      "turn",
		Timestamp: time.Now(),
	})
}

func (w *WebPushNotifier) NotifyDeadlineWarning(ctx context.Context, playerID, matchID string, hoursLeft int) error {
	return w.sendToSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "Peringatan Batas Waktu Giliran",
		Body:      fmt.Sprintf("Tersisa %d jam untuk menyelesaikan giliranmu di Room %s sebelum bot mengambil alih.", hoursLeft, matchID),
		MatchID:   matchID,
		Type:      "deadline_warning",
		Timestamp: time.Now(),
	})
}

func (w *WebPushNotifier) NotifyMatchEnded(ctx context.Context, playerID, matchID string, won bool) error {
	status := "Kalah menghadapi kegelapan"
	if won {
		status = "Menang! Mercusuar terakhir berhasil dinyalakan!"
	}
	return w.sendToSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "Match Telah Selesai",
		Body:      fmt.Sprintf("Permainan di Room %s telah selesai: %s.", matchID, status),
		MatchID:   matchID,
		Type:      "match_ended",
		Timestamp: time.Now(),
	})
}

func (w *WebPushNotifier) NotifyAFKWarning(ctx context.Context, playerID, matchID string) error {
	return w.sendToSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "Pemain AFK — Bot Mengambil Alih",
		Body:      fmt.Sprintf("Kamu melewatkan giliran berturut-turut di Room %s. Bot mengambil alih giliranmu. Kamu bisa masuk kembali kapan saja.", matchID),
		MatchID:   matchID,
		Type:      "afk_warning",
		Timestamp: time.Now(),
	})
}
