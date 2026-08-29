package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/auth"
	"github.com/lastlighthouse/lastlighthouse/internal/match"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
	"github.com/lastlighthouse/lastlighthouse/internal/transport/ws"
)

type Server struct {
	mux      *http.ServeMux
	auth     *auth.Authenticator
	store    store.Store
	registry *match.Registry
	hub      *ws.Hub
}

func NewServer(
	a *auth.Authenticator,
	s store.Store,
	r *match.Registry,
	hub *ws.Hub,
) *Server {
	srv := &Server{
		mux:      http.NewServeMux(),
		auth:     a,
		store:    s,
		registry: r,
		hub:      hub,
	}
	srv.routes()
	return srv
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/auth/guest", s.handleGuestAuth)
	s.mux.HandleFunc("GET /api/lobby", s.handleListLobbies)
	s.mux.HandleFunc("POST /api/lobby", s.handleCreateMatch)
	s.mux.HandleFunc("GET /api/match/{id}", s.handleGetMatch)
	s.mux.Handle("/ws", s.hub)
}

type guestAuthReq struct {
	DisplayName string `json:"displayName"`
}

type guestAuthResp struct {
	Token       string `json:"token"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

func (s *Server) handleGuestAuth(w http.ResponseWriter, r *http.Request) {
	var req guestAuthReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.DisplayName) == "" {
		http.Error(w, `{"error":"invalid display name"}`, http.StatusBadRequest)
		return
	}

	token, uid, err := s.auth.GenerateGuestToken(req.DisplayName)
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	_ = s.store.CreateUser(r.Context(), store.UserRecord{
		ID:          uid,
		DisplayName: req.DisplayName,
		GuestToken:  token,
		CreatedAt:   time.Now(),
	})

	writeJSON(w, http.StatusOK, guestAuthResp{
		Token:       token,
		UserID:      uid,
		DisplayName: req.DisplayName,
	})
}

func (s *Server) handleListLobbies(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	list, err := s.store.ListMatches(r.Context(), status)
	if err != nil {
		http.Error(w, `{"error":"failed to list matches"}`, http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []store.MatchRecord{}
	}
	writeJSON(w, http.StatusOK, list)
}

type createMatchReq struct {
	MatchID string             `json:"matchId"`
	Seed    int64              `json:"seed"`
	Players []core.PlayerSetup `json:"players"`
}

func (s *Server) handleCreateMatch(w http.ResponseWriter, r *http.Request) {
	var req createMatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.MatchID == "" {
		req.MatchID = "m_" + uuid.New().String()[:8]
	}
	if req.Seed == 0 {
		req.Seed = time.Now().UnixNano()
	}
	if len(req.Players) < 2 {
		http.Error(w, `{"error":"at least 2 players required"}`, http.StatusBadRequest)
		return
	}

	actor, err := s.registry.CreateMatch(r.Context(), req.MatchID, req.Seed, req.Players)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"matchId":     actor.ID(),
		"status":      "active",
		"contentHash": actor.State().ContentHash,
	})
}

func (s *Server) handleGetMatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error":"match id required"}`, http.StatusBadRequest)
		return
	}

	rec, err := s.store.GetMatch(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
