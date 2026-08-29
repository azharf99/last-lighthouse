package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"nhooyr.io/websocket"
)

const (
	writeWait      = 10 * time.Second
	pingPeriod     = 20 * time.Second
	maxMessageSize = 64 * 1024 // 64 KB
)

type MessageHandler func(conn *Conn, env InEnvelope)

// Conn membungkus websocket.Conn dengan write queue dan heartbeat (ADR-003).
// Mengimplementasikan match.Subscriber.
type Conn struct {
	wsConn    *websocket.Conn
	userID    string
	playerID  string
	matchID   string
	sendQueue chan []byte
	handler   MessageHandler
	closeOnce sync.Once
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewConn(wsConn *websocket.Conn, userID string, handler MessageHandler) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		wsConn:    wsConn,
		userID:    userID,
		sendQueue: make(chan []byte, 64),
		handler:   handler,
		ctx:       ctx,
		cancel:    cancel,
	}

	go c.writePump()
	go c.readPump()
	go c.heartbeat()

	return c
}

func (c *Conn) UserID() string {
	return c.userID
}

func (c *Conn) PlayerID() string {
	return c.playerID
}

func (c *Conn) SetPlayer(playerID, matchID string) {
	c.playerID = playerID
	c.matchID = matchID
}

func (c *Conn) MatchID() string {
	return c.matchID
}

func (c *Conn) SendEvents(matchID string, eventSeq int64, events []core.Event) error {
	return c.SendEnvelope(OutEnvelope{
		V:        ProtocolVersion,
		Type:     "events",
		MatchID:  matchID,
		EventSeq: eventSeq,
		Payload:  events,
	})
}

func (c *Conn) SendSnapshot(matchID string, eventSeq int64, view *core.PlayerView) error {
	return c.SendEnvelope(OutEnvelope{
		V:        ProtocolVersion,
		Type:     "snapshot",
		MatchID:  matchID,
		EventSeq: eventSeq,
		Payload:  view,
	})
}

func (c *Conn) SendError(matchID string, code, message string) error {
	return c.SendEnvelope(OutEnvelope{
		V:        ProtocolVersion,
		Type:     "error",
		MatchID:  matchID,
		Payload:  ErrorPayload{Code: code, Message: message},
	})
}

func (c *Conn) SendEnvelope(env OutEnvelope) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}

	select {
	case c.sendQueue <- b:
		return nil
	case <-c.ctx.Done():
		return errors.New("connection closed")
	default:
		log.Printf("ws send queue full for user %s, dropping message", c.userID)
		return errors.New("send queue full")
	}
}

func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		_ = c.wsConn.Close(websocket.StatusNormalClosure, "closed")
	})
	return nil
}

func (c *Conn) writePump() {
	defer c.Close()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.sendQueue:
			if !ok {
				return
			}
			writeCtx, writeCancel := context.WithTimeout(c.ctx, writeWait)
			err := c.wsConn.Write(writeCtx, websocket.MessageText, msg)
			writeCancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Conn) readPump() {
	defer c.Close()

	c.wsConn.SetReadLimit(maxMessageSize)

	for {
		typ, data, err := c.wsConn.Read(c.ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			continue
		}

		var env InEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			_ = c.SendEnvelope(OutEnvelope{
				V:    ProtocolVersion,
				Type: "error",
				Payload: ErrorPayload{
					Code:    "BAD_REQUEST",
					Message: "invalid JSON envelope",
				},
			})
			continue
		}

		if env.Type == "ping" {
			_ = c.SendEnvelope(OutEnvelope{
				V:    ProtocolVersion,
				Type: "pong",
			})
			continue
		}

		if c.handler != nil {
			c.handler(c, env)
		}
	}
}

func (c *Conn) heartbeat() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(c.ctx, 5*time.Second)
			err := c.wsConn.Ping(pingCtx)
			pingCancel()
			if err != nil {
				c.Close()
				return
			}
		}
	}
}
