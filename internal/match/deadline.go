package match

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/notify"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

// DeadlineScheduler mengelola polling batas waktu giliran di background untuk mode async & realtime (ADR-007).
type DeadlineScheduler struct {
	store    store.Store
	registry *Registry
	notifier notify.Notifier
	interval time.Duration
	afkLimit int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewDeadlineScheduler(
	s store.Store,
	reg *Registry,
	notif notify.Notifier,
	interval time.Duration,
) *DeadlineScheduler {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if notif == nil {
		notif = notify.NewMockNotifier(s)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &DeadlineScheduler{
		store:    s,
		registry: reg,
		notifier: notif,
		interval: interval,
		afkLimit: 2,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start menjalankan background goroutine untuk memeriksa deadline secara periodik.
func (d *DeadlineScheduler) Start() {
	d.wg.Add(1)
	go d.run()
}

// Stop menghentikan worker scheduler.
func (d *DeadlineScheduler) Stop() {
	d.cancel()
	d.wg.Wait()
}

func (d *DeadlineScheduler) run() {
	defer d.wg.Done()
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case now := <-ticker.C:
			d.CheckDeadlines(d.ctx, now)
		}
	}
}

// CheckDeadlines memproses seluruh match yang batas waktu gilirannya telah terlewat.
func (d *DeadlineScheduler) CheckDeadlines(ctx context.Context, now time.Time) {
	expired, err := d.store.GetExpiredDeadlines(ctx, now)
	if err != nil {
		log.Printf("[DeadlineScheduler] error fetching expired deadlines: %v", err)
		return
	}

	for _, item := range expired {
		d.processExpiredDeadline(ctx, item, now)
	}
}

func (d *DeadlineScheduler) processExpiredDeadline(ctx context.Context, dl store.TurnDeadline, now time.Time) {
	actor, err := d.registry.GetOrLoad(ctx, dl.MatchID)
	if err != nil {
		log.Printf("[DeadlineScheduler] cannot load match %s: %v", dl.MatchID, err)
		return
	}

	st := actor.State()
	if st == nil || st.Over() {
		_ = d.store.ClearTurnDeadline(ctx, dl.MatchID)
		return
	}

	active := st.ActivePlayer()
	if active == nil || string(active.ID) != dl.PlayerID {
		// Player already took turn; update deadline to current active player
		if active != nil {
			_ = d.ScheduleNextTurnDeadline(ctx, dl.MatchID, string(active.ID), int(actor.turnTimeout.Seconds()))
		} else {
			_ = d.store.ClearTurnDeadline(ctx, dl.MatchID)
		}
		return
	}

	// Player timed out: increment missed count
	dl.Missed++

	if dl.Missed >= d.afkLimit {
		_ = d.notifier.NotifyAFKWarning(ctx, dl.PlayerID, dl.MatchID)
	}

	// Trigger timeout on match actor
	actor.handleTurnTimeout()

	// Check state after timeout execution
	newSt := actor.State()
	if newSt == nil || newSt.Over() {
		_ = d.store.ClearTurnDeadline(ctx, dl.MatchID)
		if newSt != nil {
			won := newSt.Status == core.StatusWon
			for _, p := range newSt.Players {
				_ = d.notifier.NotifyMatchEnded(ctx, string(p.ID), dl.MatchID, won)
			}
		}
		return
	}

	// Schedule deadline for newly active player
	newActive := newSt.ActivePlayer()
	if newActive != nil {
		missedCount := 0
		if string(newActive.ID) == dl.PlayerID {
			missedCount = dl.Missed
		}
		newDeadline := now.Add(actor.turnTimeout)
		_ = d.store.SetTurnDeadline(ctx, store.TurnDeadline{
			MatchID:    dl.MatchID,
			PlayerID:   string(newActive.ID),
			DeadlineAt: newDeadline,
			Missed:     missedCount,
		})

		roleName := string(newActive.Character)
		_ = d.notifier.NotifyTurn(ctx, string(newActive.ID), dl.MatchID, roleName, newDeadline)
	}
}

// ScheduleNextTurnDeadline mencatat batas waktu giliran baru dan mengirim notifikasi push.
func (d *DeadlineScheduler) ScheduleNextTurnDeadline(
	ctx context.Context,
	matchID string,
	playerID string,
	timeoutSec int,
) error {
	if timeoutSec <= 0 {
		return d.store.ClearTurnDeadline(ctx, matchID)
	}

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	err := d.store.SetTurnDeadline(ctx, store.TurnDeadline{
		MatchID:    matchID,
		PlayerID:   playerID,
		DeadlineAt: deadline,
		Missed:     0,
	})
	if err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	return nil
}
