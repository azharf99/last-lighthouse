package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

// FCMNotifier menangani formatting & dispatch notifikasi untuk Android (FCM) dan iOS (APNs via FCM).
type FCMNotifier struct {
	store store.Store
}

func NewFCMNotifier(s store.Store) *FCMNotifier {
	return &FCMNotifier{
		store: s,
	}
}

func (f *FCMNotifier) VAPIDPublicKey() string {
	return ""
}

func (f *FCMNotifier) sendToMobileSubscriptions(ctx context.Context, playerID string, payload NotificationPayload) error {
	subs, err := f.store.GetPushSubscriptions(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get mobile subscriptions for %s: %w", playerID, err)
	}

	for _, sub := range subs {
		if (sub.Platform == "fcm" || sub.Platform == "apns") && sub.Endpoint != "" {
			// In production, post to Google FCM / Apple APNs HTTP/2 endpoint
			_ = sub
		}
	}
	return nil
}

func (f *FCMNotifier) NotifyTurn(ctx context.Context, playerID, matchID, roleName string, deadline time.Time) error {
	body := fmt.Sprintf("Giliranmu telah tiba sebagai %s di Room %s!", roleName, matchID)
	if !deadline.IsZero() {
		dur := time.Until(deadline).Round(time.Minute)
		if dur > 0 {
			body += fmt.Sprintf(" Sisa waktu: %v.", dur)
		}
	}
	return f.sendToMobileSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "The Last Lighthouse — Giliranmu!",
		Body:      body,
		MatchID:   matchID,
		Type:      "turn",
		Timestamp: time.Now(),
	})
}

func (f *FCMNotifier) NotifyDeadlineWarning(ctx context.Context, playerID, matchID string, hoursLeft int) error {
	return f.sendToMobileSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "Peringatan Batas Waktu Giliran",
		Body:      fmt.Sprintf("Tersisa %d jam untuk menyelesaikan giliranmu di Room %s sebelum bot mengambil alih.", hoursLeft, matchID),
		MatchID:   matchID,
		Type:      "deadline_warning",
		Timestamp: time.Now(),
	})
}

func (f *FCMNotifier) NotifyMatchEnded(ctx context.Context, playerID, matchID string, won bool) error {
	status := "Kalah menghadapi kegelapan"
	if won {
		status = "Menang! Mercusuar terakhir berhasil dinyalakan!"
	}
	return f.sendToMobileSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "Match Telah Selesai",
		Body:      fmt.Sprintf("Permainan di Room %s telah selesai: %s.", matchID, status),
		MatchID:   matchID,
		Type:      "match_ended",
		Timestamp: time.Now(),
	})
}

func (f *FCMNotifier) NotifyAFKWarning(ctx context.Context, playerID, matchID string) error {
	return f.sendToMobileSubscriptions(ctx, playerID, NotificationPayload{
		Title:     "Pemain AFK — Bot Mengambil Alih",
		Body:      fmt.Sprintf("Kamu melewatkan giliran berturut-turut di Room %s. Bot mengambil alih giliranmu. Kamu bisa masuk kembali kapan saja.", matchID),
		MatchID:   matchID,
		Type:      "afk_warning",
		Timestamp: time.Now(),
	})
}
