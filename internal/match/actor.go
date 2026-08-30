package match

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lastlighthouse/lastlighthouse/bot"
	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
)

var (
	ErrMatchNotActive  = errors.New("match is not active")
	ErrMatchFull       = errors.New("match lobby is full")
	ErrAlreadyJoined   = errors.New("player already joined")
	ErrNotYourTurn     = errors.New("not your turn")
	ErrDuplicateClient = errors.New("duplicate command sequence")
)

type Subscriber interface {
	SendEvents(matchID string, eventSeq int64, events []core.Event) error
	SendSnapshot(matchID string, eventSeq int64, view *core.PlayerView) error
	SendError(matchID string, code, message string) error
	Close() error
}

type inMsgType int

const (
	msgCmd inMsgType = iota
	msgJoin
	msgLeave
	msgResync
	msgStart
	msgTimerTick
	msgShutdown
)

type actorMsg struct {
	kind         inMsgType
	playerID     core.PlayerID
	clientSeq    int64
	cmd          core.Command
	sub          Subscriber
	lastEventSeq int64
	setups       []core.PlayerSetup
	seed         int64
	replyErr     chan error
	replyDone    chan struct{}
}

// Actor mengelola satu goroutine per match (ADR-004).
// Menjamin seluruh mutasi state dan I/O terserialisasi tanpa race condition.
type Actor struct {
	id          string
	state       *core.State
	content     *core.Content
	rng         *core.RNG
	eventSeq    int64
	store       store.Store
	subs        map[core.PlayerID]Subscriber
	lastSeq     map[core.PlayerID]int64
	missedTurns map[core.PlayerID]int
	turnTimeout time.Duration
	turnTimer   *time.Timer
	inbox       chan actorMsg
	aiBot       *bot.Bot

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewActor(
	id string,
	st *core.State,
	c *core.Content,
	s store.Store,
	turnTimeout time.Duration,
) *Actor {
	if turnTimeout <= 0 {
		// Default 90s per ADR-003 & ADR-007 realtime preset.
		// Catatan untuk M5 (Async Play): Preset Relaxed (10m) dan Async (24h)
		// dapat disetel di sini atau via field match.TurnTimeoutSec.
		turnTimeout = 90 * time.Second
	}
	if c == nil {
		c = core.DefaultContent()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a := &Actor{
		id:          id,
		state:       st,
		content:     c,
		store:       s,
		subs:        make(map[core.PlayerID]Subscriber),
		lastSeq:     make(map[core.PlayerID]int64),
		missedTurns: make(map[core.PlayerID]int),
		turnTimeout: turnTimeout,
		inbox:       make(chan actorMsg, 128),
		aiBot:       bot.New(bot.Standard),
		ctx:         ctx,
		cancel:      cancel,
	}

	if st != nil {
		a.rng = core.NewRNG(st.RNGState)
	}

	a.wg.Add(1)
	go a.run()
	return a
}

func (a *Actor) ID() string {
	return a.id
}

func (a *Actor) State() *core.State {
	return a.state
}

func (a *Actor) EventSeq() int64 {
	return a.eventSeq
}

func (a *Actor) Start(seed int64, setups []core.PlayerSetup) error {
	reply := make(chan error, 1)
	a.inbox <- actorMsg{
		kind:     msgStart,
		seed:     seed,
		setups:   setups,
		replyErr: reply,
	}
	return <-reply
}

func (a *Actor) SubmitCommand(pid core.PlayerID, clientSeq int64, cmd core.Command) error {
	reply := make(chan error, 1)
	a.inbox <- actorMsg{
		kind:      msgCmd,
		playerID:  pid,
		clientSeq: clientSeq,
		cmd:       cmd,
		replyErr:  reply,
	}
	return <-reply
}

func (a *Actor) Join(pid core.PlayerID, sub Subscriber, lastEventSeq int64) error {
	reply := make(chan error, 1)
	a.inbox <- actorMsg{
		kind:         msgJoin,
		playerID:     pid,
		sub:          sub,
		lastEventSeq: lastEventSeq,
		replyErr:     reply,
	}
	return <-reply
}

func (a *Actor) Leave(pid core.PlayerID) {
	done := make(chan struct{}, 1)
	a.inbox <- actorMsg{
		kind:      msgLeave,
		playerID:  pid,
		replyDone: done,
	}
	<-done
}

func (a *Actor) Resync(pid core.PlayerID, lastEventSeq int64) error {
	reply := make(chan error, 1)
	a.inbox <- actorMsg{
		kind:         msgResync,
		playerID:     pid,
		lastEventSeq: lastEventSeq,
		replyErr:     reply,
	}
	return <-reply
}

func (a *Actor) Stop() {
	a.cancel()
	a.wg.Wait()
}

func (a *Actor) run() {
	defer a.wg.Done()

	var timerChan <-chan time.Time

	for {
		if a.turnTimer != nil {
			timerChan = a.turnTimer.C
		} else {
			timerChan = nil
		}

		select {
		case <-a.ctx.Done():
			a.handleShutdown()
			return

		case <-timerChan:
			a.handleTurnTimeout()

		case msg := <-a.inbox:
			switch msg.kind {
			case msgStart:
				msg.replyErr <- a.handleStart(msg.seed, msg.setups)
			case msgCmd:
				msg.replyErr <- a.handleCommand(msg.playerID, msg.clientSeq, msg.cmd)
			case msgJoin:
				msg.replyErr <- a.handleJoin(msg.playerID, msg.sub, msg.lastEventSeq)
			case msgLeave:
				a.handleLeave(msg.playerID)
				if msg.replyDone != nil {
					msg.replyDone <- struct{}{}
				}
			case msgResync:
				msg.replyErr <- a.handleResync(msg.playerID, msg.lastEventSeq)
			}
		}
	}
}

func (a *Actor) handleStart(seed int64, setups []core.PlayerSetup) error {
	if a.state != nil && a.state.Status == core.StatusActive {
		return errors.New("game already started")
	}

	st, events, err := core.NewGame(a.id, seed, setups, a.content)
	if err != nil {
		return fmt.Errorf("new game: %w", err)
	}

	a.state = st
	a.rng = core.NewRNG(st.RNGState)

	// Persist initial setup & events synchronously
	var records []store.EventRecord
	now := time.Now()
	for _, e := range events {
		a.eventSeq++
		b, _ := json.Marshal(e)
		records = append(records, store.EventRecord{
			MatchID: a.id,
			Seq:     a.eventSeq,
			Kind:    string(e.Kind),
			Payload: b,
			At:      now,
		})
	}

	if err := a.store.AppendEvents(context.Background(), a.id, records); err != nil {
		log.Printf("store append initial events error: %v", err)
	}
	_ = a.store.UpdateMatchStatus(context.Background(), a.id, "active")

	// Save snapshot
	stateBytes, _ := json.Marshal(a.state)
	_ = a.store.SaveSnapshot(context.Background(), a.id, a.eventSeq, stateBytes)

	// Broadcast projected events to all joined subscribers
	a.broadcastProjected(events)
	a.resetTurnTimer()

	return nil
}

func (a *Actor) handleCommand(pid core.PlayerID, clientSeq int64, cmd core.Command) error {
	if a.state == nil || a.state.Over() {
		return ErrMatchNotActive
	}

	// Idempotency check: duplicate clientSeq ignored
	if clientSeq > 0 && a.lastSeq[pid] >= clientSeq {
		return ErrDuplicateClient
	}
	if clientSeq > 0 {
		a.lastSeq[pid] = clientSeq
	}

	// Make sure command player matches authenticated sender
	cmd.Player = pid

	events, err := core.Decide(a.state, cmd, a.content, a.rng)
	if err != nil {
		return err
	}

	// Apply events and build records
	var records []store.EventRecord
	now := time.Now()
	for _, e := range events {
		a.eventSeq++
		core.Apply(a.state, e)
		b, _ := json.Marshal(e)
		records = append(records, store.EventRecord{
			MatchID: a.id,
			Seq:     a.eventSeq,
			Kind:    string(e.Kind),
			Payload: b,
			At:      now,
		})
	}
	a.state.RNGState = a.rng.Seed()

	// Persist synchronously BEFORE broadcasting (ADR-004)
	if err := a.store.AppendEvents(context.Background(), a.id, records); err != nil {
		log.Printf("match %s: persist events failed: %v", a.id, err)
		return fmt.Errorf("persist events: %w", err)
	}

	// Snapshot every 20 events or when game ends
	if a.eventSeq%20 == 0 || a.state.Over() {
		stateBytes, _ := json.Marshal(a.state)
		_ = a.store.SaveSnapshot(context.Background(), a.id, a.eventSeq, stateBytes)
	}

	if a.state.Over() {
		status := "lost"
		won := false
		if a.state.Status == core.StatusWon {
			status = "won"
			won = true
		}
		_ = a.store.UpdateMatchStatus(context.Background(), a.id, status)
		if a.turnTimer != nil {
			a.turnTimer.Stop()
			a.turnTimer = nil
		}

		// Otomatis catat entri leaderboard untuk semua pemain dalam match ini
		for _, p := range a.state.Players {
			_ = a.store.AddLeaderboardEntry(context.Background(), store.LeaderboardEntry{
				PlayerName:            p.Name,
				Character:             string(p.Character),
				VP:                    p.VP,
				Darkness:              a.state.Darkness,
				Rounds:                a.state.Round,
				Won:                   won,
				MonstersSlain:         p.MonstersSlain,
				ComponentsContributed: p.RepairsJoined,
				MatchID:               a.id,
				CreatedAt:             time.Now(),
			})
		}
	} else {
		// Player made a valid manual action -> reset missed turns counter
		if clientSeq > 0 {
			a.missedTurns[pid] = 0
		}
		a.resetTurnTimer()
	}

	// Broadcast projected events to each subscriber
	a.broadcastProjected(events)
	return nil
}

func (a *Actor) handleJoin(pid core.PlayerID, sub Subscriber, lastEventSeq int64) error {
	a.subs[pid] = sub

	// If match is active, send initial view or delta events
	if a.state != nil {
		if err := a.sendSyncToPlayer(pid, sub, lastEventSeq); err != nil {
			return err
		}
	}

	return nil
}

func (a *Actor) handleLeave(pid core.PlayerID) {
	delete(a.subs, pid)
}

func (a *Actor) handleResync(pid core.PlayerID, lastEventSeq int64) error {
	sub, exists := a.subs[pid]
	if !exists {
		return errors.New("subscriber not found")
	}
	return a.sendSyncToPlayer(pid, sub, lastEventSeq)
}

func (a *Actor) sendSyncToPlayer(pid core.PlayerID, sub Subscriber, lastEventSeq int64) error {
	// Delta sync if gap is small and positive
	if lastEventSeq > 0 && (a.eventSeq-lastEventSeq) <= 50 && lastEventSeq <= a.eventSeq {
		records, err := a.store.LoadEvents(context.Background(), a.id, lastEventSeq)
		if err == nil && len(records) > 0 {
			var events []core.Event
			for _, r := range records {
				var ev core.Event
				if err := json.Unmarshal(r.Payload, &ev); err == nil {
					events = append(events, ev)
				}
			}
			projected := core.ProjectEvents(events, pid)
			return sub.SendEvents(a.id, a.eventSeq, projected)
		}
	}

	// Fallback to full PlayerView snapshot projection (ADR-003 & ADR-006)
	view := core.Project(a.state, pid)
	return sub.SendSnapshot(a.id, a.eventSeq, view)
}

func (a *Actor) broadcastProjected(events []core.Event) {
	for pid, sub := range a.subs {
		projected := core.ProjectEvents(events, pid)
		if len(projected) == 0 {
			continue
		}
		_ = sub.SendEvents(a.id, a.eventSeq, projected)
	}
}

func (a *Actor) resetTurnTimer() {
	if a.turnTimer != nil {
		a.turnTimer.Stop()
	}
	if a.state == nil || a.state.Over() || a.turnTimeout <= 0 {
		a.turnTimer = nil
		return
	}
	a.turnTimer = time.NewTimer(a.turnTimeout)
}

func (a *Actor) handleTurnTimeout() {
	if a.state == nil || a.state.Over() {
		return
	}

	active := a.state.ActivePlayer()
	if active == nil {
		return
	}
	pid := active.ID

	a.missedTurns[pid]++

	// Determine timeout action:
	// If player is AFK (2 consecutive timeouts), AI bot takes over this move (ADR-007)
	var cmd core.Command
	view := core.Project(a.state, pid)
	legal := core.LegalCommands(view, a.content)

	if a.missedTurns[pid] >= 2 {
		if c, ok := a.aiBot.Choose(view, legal, a.rng); ok {
			cmd = c
		} else {
			cmd = core.Command{Kind: core.CmdEndTurn, Player: pid}
		}
	} else {
		// Default action on single timeout: end turn or pick first pending option
		if a.state.Pending != nil && a.state.Pending.Player == pid {
			if len(a.state.Pending.Options) > 0 {
				cmd = core.Command{Kind: core.CmdChoose, Player: pid, Option: a.state.Pending.Options[0]}
			} else if len(a.state.Pending.Cards) > 0 {
				cmd = core.Command{Kind: core.CmdChoose, Player: pid, Card: a.state.Pending.Cards[0]}
			} else {
				cmd = core.Command{Kind: core.CmdEndTurn, Player: pid}
			}
		} else {
			cmd = core.Command{Kind: core.CmdEndTurn, Player: pid}
		}
	}

	// Apply default action
	_ = a.handleCommand(pid, 0, cmd)
}

func (a *Actor) handleShutdown() {
	if a.turnTimer != nil {
		a.turnTimer.Stop()
	}
	if a.state != nil {
		stateBytes, _ := json.Marshal(a.state)
		_ = a.store.SaveSnapshot(context.Background(), a.id, a.eventSeq, stateBytes)
	}
}
