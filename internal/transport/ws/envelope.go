package ws

import (
	"encoding/json"
)

const ProtocolVersion = 1

// InEnvelope adalah format pesan dari client ke server melalui WebSocket (ADR-003).
type InEnvelope struct {
	V         int             `json:"v"`
	Type      string          `json:"type"` // cmd, resync, ping, join, leave
	MatchID   string          `json:"matchId"`
	ClientSeq int64           `json:"clientSeq,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// OutEnvelope adalah format pesan dari server ke client melalui WebSocket (ADR-003).
type OutEnvelope struct {
	V        int         `json:"v"`
	Type     string      `json:"type"` // events, snapshot, error, pong, lobby, joined, sync
	MatchID  string      `json:"matchId,omitempty"`
	EventSeq int64       `json:"eventSeq,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

// ErrorPayload adalah payload untuk OutEnvelope dengan Type "error".
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
