package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/auth"
	"github.com/lastlighthouse/lastlighthouse/internal/match"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
	"nhooyr.io/websocket"
)

type Hub struct {
	registry *match.Registry
	store    store.Store
	auth     *auth.Authenticator
	conns    map[*Conn]struct{}
	mu       sync.RWMutex
}

func NewHub(r *match.Registry, s store.Store, a *auth.Authenticator) *Hub {
	return &Hub{
		registry: r,
		store:    s,
		auth:     a,
		conns:    make(map[*Conn]struct{}),
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate from query param ?token=... or header
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	claims, err := h.auth.ValidateToken(tokenStr)
	if err != nil {
		http.Error(w, "unauthorized: valid JWT token required", http.StatusUnauthorized)
		return
	}

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}

	conn := NewConn(wsConn, claims.UserID, h.dispatchMessage)

	h.mu.Lock()
	h.conns[conn] = struct{}{}
	h.mu.Unlock()

	// Wait until connection closes
	<-conn.ctx.Done()

	h.mu.Lock()
	delete(h.conns, conn)
	h.mu.Unlock()

	if conn.MatchID() != "" && conn.PlayerID() != "" {
		if actor, err := h.registry.GetOrLoad(context.Background(), conn.MatchID()); err == nil {
			actor.Leave(core.PlayerID(conn.PlayerID()))
		}
	}
}

type joinPayload struct {
	Player       string `json:"player"`
	LastEventSeq int64  `json:"lastEventSeq"`
}

func (h *Hub) dispatchMessage(conn *Conn, env InEnvelope) {
	ctx := context.Background()

	switch env.Type {
	case "join":
		var p joinPayload
		if len(env.Payload) > 0 {
			_ = json.Unmarshal(env.Payload, &p)
		}
		if p.Player == "" {
			p.Player = conn.UserID()
		}

		actor, err := h.registry.GetOrLoad(ctx, env.MatchID)
		if err != nil {
			_ = conn.SendEnvelope(OutEnvelope{
				V:       ProtocolVersion,
				Type:    "error",
				MatchID: env.MatchID,
				Payload: ErrorPayload{Code: "MATCH_NOT_FOUND", Message: err.Error()},
			})
			return
		}

		pid := core.PlayerID(p.Player)
		conn.SetPlayer(p.Player, env.MatchID)

		if err := actor.Join(pid, conn, p.LastEventSeq); err != nil {
			_ = conn.SendEnvelope(OutEnvelope{
				V:       ProtocolVersion,
				Type:    "error",
				MatchID: env.MatchID,
				Payload: ErrorPayload{Code: "JOIN_FAILED", Message: err.Error()},
			})
		}

	case "cmd":
		actor, err := h.registry.GetOrLoad(ctx, env.MatchID)
		if err != nil {
			_ = conn.SendEnvelope(OutEnvelope{
				V:       ProtocolVersion,
				Type:    "error",
				MatchID: env.MatchID,
				Payload: ErrorPayload{Code: "MATCH_NOT_FOUND", Message: err.Error()},
			})
			return
		}

		var cmd core.Command
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			_ = conn.SendEnvelope(OutEnvelope{
				V:       ProtocolVersion,
				Type:    "error",
				MatchID: env.MatchID,
				Payload: ErrorPayload{Code: "BAD_COMMAND", Message: "invalid command payload"},
			})
			return
		}

		pid := core.PlayerID(conn.PlayerID())
		if pid == "" {
			pid = core.PlayerID(cmd.Player)
		}

		if err := actor.SubmitCommand(pid, env.ClientSeq, cmd); err != nil {
			_ = conn.SendEnvelope(OutEnvelope{
				V:       ProtocolVersion,
				Type:    "error",
				MatchID: env.MatchID,
				Payload: ErrorPayload{Code: "RULE_REJECTION", Message: err.Error()},
			})
		}

	case "resync":
		actor, err := h.registry.GetOrLoad(ctx, env.MatchID)
		if err != nil {
			return
		}
		pid := core.PlayerID(conn.PlayerID())
		_ = actor.Resync(pid, env.ClientSeq)

	case "leave":
		actor, err := h.registry.GetOrLoad(ctx, env.MatchID)
		if err == nil && conn.PlayerID() != "" {
			actor.Leave(core.PlayerID(conn.PlayerID()))
		}
	}
}
