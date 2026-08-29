package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/auth"
	"github.com/lastlighthouse/lastlighthouse/internal/match"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
	"github.com/lastlighthouse/lastlighthouse/internal/transport/ws"
	nhooyrws "nhooyr.io/websocket"
)

func setupTestServer() (*Server, *auth.Authenticator, store.Store, *match.Registry) {
	s := store.NewMemoryStore()
	a := auth.NewAuthenticator(nil)
	c := core.DefaultContent()
	reg := match.NewRegistry(s, c, 5*time.Second)
	hub := ws.NewHub(reg, s, a)
	srv := NewServer(a, s, reg, hub)
	return srv, a, s, reg
}

func TestGuestAuthAndLobbyAPI(t *testing.T) {
	srv, _, _, _ := setupTestServer()
	h := srv.Handler()

	// 1. Guest Auth
	body, _ := json.Marshal(map[string]string{"displayName": "Alice"})
	req := httptest.NewRequest("POST", "/api/auth/guest", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var authResp guestAuthResp
	_ = json.Unmarshal(rec.Body.Bytes(), &authResp)
	if authResp.Token == "" || authResp.UserID == "" || authResp.DisplayName != "Alice" {
		t.Fatalf("unexpected auth resp: %+v", authResp)
	}

	// 2. Create Match
	createBody, _ := json.Marshal(createMatchReq{
		MatchID: "m_test1",
		Seed:    100,
		Players: []core.PlayerSetup{
			{ID: "p1", Name: "Alice", Character: "navigator"},
			{ID: "p2", Name: "Bob", Character: "engineer"},
		},
	})
	req = httptest.NewRequest("POST", "/api/lobby", bytes.NewReader(createBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. List Lobby
	req = httptest.NewRequest("GET", "/api/lobby", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var list []store.MatchRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].ID != "m_test1" {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestWebSocketEndToEnd(t *testing.T) {
	srv, a, _, _ := setupTestServer()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	token, _, _ := a.GenerateGuestToken("Alice")

	// Create match first via REST
	body, _ := json.Marshal(createMatchReq{
		MatchID: "m_ws1",
		Seed:    200,
		Players: []core.PlayerSetup{
			{ID: "p1", Name: "Alice", Character: "navigator"},
			{ID: "p2", Name: "Bob", Character: "engineer"},
		},
	})
	resp, err := http.Post(server.URL+"/api/lobby", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create match: %v, status: %d", err, resp.StatusCode)
	}

	// Connect to WS
	wsURL := "ws" + server.URL[4:] + "/ws?token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := nhooyrws.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WS Dial failed: %v", err)
	}
	defer conn.Close(nhooyrws.StatusNormalClosure, "")

	// Send join envelope
	joinEnv := ws.InEnvelope{
		V:       1,
		Type:    "join",
		MatchID: "m_ws1",
		Payload: []byte(`{"player":"p1","lastEventSeq":0}`),
	}
	joinBytes, _ := json.Marshal(joinEnv)
	if err := conn.Write(ctx, nhooyrws.MessageText, joinBytes); err != nil {
		t.Fatalf("write join failed: %v", err)
	}

	// Read initial snapshot/sync
	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read snapshot failed: %v", err)
	}

	var outEnv ws.OutEnvelope
	if err := json.Unmarshal(msg, &outEnv); err != nil {
		t.Fatalf("unmarshal envelope failed: %v", err)
	}

	if outEnv.Type != "snapshot" && outEnv.Type != "events" {
		t.Fatalf("expected snapshot/events, got %s: %s", outEnv.Type, string(msg))
	}
}
