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
	"github.com/lastlighthouse/lastlighthouse/internal/notify"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
	"github.com/lastlighthouse/lastlighthouse/internal/telemetry"
	"github.com/lastlighthouse/lastlighthouse/internal/transport/ws"
)

type Server struct {
	mux       *http.ServeMux
	auth      *auth.Authenticator
	store     store.Store
	registry  *match.Registry
	hub       *ws.Hub
	notifier  notify.Notifier
	scheduler *match.DeadlineScheduler
	telemetry *telemetry.Collector
}

func NewServer(
	a *auth.Authenticator,
	s store.Store,
	r *match.Registry,
	hub *ws.Hub,
) *Server {
	notif := notify.NewMockNotifier(s)
	sched := match.NewDeadlineScheduler(s, r, notif, 10*time.Second)
	telem := telemetry.NewCollector()

	srv := &Server{
		mux:       http.NewServeMux(),
		auth:      a,
		store:     s,
		registry:  r,
		hub:       hub,
		notifier:  notif,
		scheduler: sched,
		telemetry: telem,
	}
	srv.routes()
	return srv
}

func (s *Server) SetNotifier(n notify.Notifier) {
	s.notifier = n
	if s.scheduler != nil {
		s.scheduler = match.NewDeadlineScheduler(s.store, s.registry, n, 10*time.Second)
	}
}

func (s *Server) Scheduler() *match.DeadlineScheduler {
	return s.scheduler
}

func (s *Server) Telemetry() *telemetry.Collector {
	return s.telemetry
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/auth/guest", s.handleGuestAuth)
	s.mux.HandleFunc("GET /api/lobby", s.handleListLobbies)
	s.mux.HandleFunc("POST /api/lobby", s.handleCreateMatch)
	s.mux.HandleFunc("GET /api/match/{id}", s.handleGetMatch)
	s.mux.HandleFunc("GET /api/match/{id}/replay", s.handleGetReplay)
	s.mux.HandleFunc("GET /api/matches/my", s.handleListMyMatches)
	s.mux.HandleFunc("GET /api/push/vapid-key", s.handleGetVAPIDKey)
	s.mux.HandleFunc("POST /api/push/subscribe", s.handlePushSubscribe)
	s.mux.HandleFunc("POST /api/telemetry/report", s.handleReportTelemetry)
	s.mux.HandleFunc("GET /api/telemetry/stats", s.handleGetTelemetryStats)
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
	MatchID        string             `json:"matchId"`
	Seed           int64              `json:"seed"`
	TurnTimeoutSec int                `json:"turnTimeoutSec"`
	Players        []core.PlayerSetup `json:"players"`
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

type replayResponse struct {
	MatchID     string       `json:"matchId"`
	Status      string       `json:"status"`
	Seed        int64        `json:"seed"`
	PlayerIDs   []string     `json:"playerIds"`
	Events      []core.Event `json:"events"`
	TotalEvents int          `json:"totalEvents"`
}

func (s *Server) handleGetReplay(w http.ResponseWriter, r *http.Request) {
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

	records, err := s.store.LoadEvents(r.Context(), id, 0)
	if err != nil {
		http.Error(w, `{"error":"failed to load events"}`, http.StatusInternalServerError)
		return
	}

	var events []core.Event
	for _, recItem := range records {
		var ev core.Event
		if err := json.Unmarshal(recItem.Payload, &ev); err == nil {
			events = append(events, ev)
		}
	}

	writeJSON(w, http.StatusOK, replayResponse{
		MatchID:     rec.ID,
		Status:      rec.Status,
		Seed:        rec.Seed,
		PlayerIDs:   rec.PlayerIDs,
		Events:      events,
		TotalEvents: len(events),
	})
}

type playerMatchItem struct {
	MatchID      string     `json:"matchId"`
	Status       string     `json:"status"`
	PlayerIDs    []string   `json:"playerIds"`
	ActivePlayer string     `json:"activePlayer,omitempty"`
	IsMyTurn     bool       `json:"isMyTurn"`
	Round        int        `json:"round"`
	Darkness     int        `json:"darkness"`
	DeadlineAt   *time.Time `json:"deadlineAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
}

func (s *Server) handleListMyMatches(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	authHeader := r.Header.Get("Authorization")
	if playerID == "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.auth.ValidateToken(token)
		if err == nil && claims != nil {
			playerID = claims.UserID
		}
	}

	if playerID == "" {
		http.Error(w, `{"error":"playerId or authorization token required"}`, http.StatusBadRequest)
		return
	}

	records, err := s.store.ListPlayerMatches(r.Context(), playerID)
	if err != nil {
		http.Error(w, `{"error":"failed to list player matches"}`, http.StatusInternalServerError)
		return
	}

	var items []playerMatchItem
	for _, rec := range records {
		item := playerMatchItem{
			MatchID:    rec.ID,
			Status:     rec.Status,
			PlayerIDs:  rec.PlayerIDs,
			CreatedAt:  rec.CreatedAt,
			FinishedAt: rec.FinishedAt,
		}

		if actor, err := s.registry.GetOrLoad(r.Context(), rec.ID); err == nil && actor != nil {
			st := actor.State()
			if st != nil {
				item.Round = st.Round
				item.Darkness = st.Darkness
				active := st.ActivePlayer()
				if active != nil {
					item.ActivePlayer = string(active.ID)
					item.IsMyTurn = (string(active.ID) == playerID)
				}
			}
		}

		if dl, err := s.store.GetTurnDeadline(r.Context(), rec.ID); err == nil && dl != nil {
			item.DeadlineAt = &dl.DeadlineAt
		}

		items = append(items, item)
	}

	if items == nil {
		items = []playerMatchItem{}
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetVAPIDKey(w http.ResponseWriter, _ *http.Request) {
	key := s.notifier.VAPIDPublicKey()
	writeJSON(w, http.StatusOK, map[string]string{
		"publicKey": key,
	})
}

type pushSubscribeReq struct {
	PlayerID string `json:"playerId"`
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
	Platform string `json:"platform"`
}

func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	var req pushSubscribeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		http.Error(w, `{"error":"invalid push subscription body"}`, http.StatusBadRequest)
		return
	}

	if req.PlayerID == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := s.auth.ValidateToken(token)
			if err == nil && claims != nil {
				req.PlayerID = claims.UserID
			}
		}
	}

	if req.PlayerID == "" {
		http.Error(w, `{"error":"playerId required"}`, http.StatusBadRequest)
		return
	}

	if req.Platform == "" {
		req.Platform = "web"
	}

	sub := store.PushSubscription{
		PlayerID:  req.PlayerID,
		Endpoint:  req.Endpoint,
		P256dh:    req.P256dh,
		Auth:      req.Auth,
		Platform:  req.Platform,
		CreatedAt: time.Now(),
	}

	if err := s.store.SavePushSubscription(r.Context(), sub); err != nil {
		http.Error(w, `{"error":"failed to save push subscription"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Telemetri Handlers (M6)

func (s *Server) handleReportTelemetry(w http.ResponseWriter, r *http.Request) {
	var report telemetry.MatchReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, `{"error":"invalid telemetry report body"}`, http.StatusBadRequest)
		return
	}

	s.telemetry.RecordMatch(r.Context(), report)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetTelemetryStats(w http.ResponseWriter, r *http.Request) {
	stats := s.telemetry.GetStats(r.Context())
	writeJSON(w, http.StatusOK, stats)
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
