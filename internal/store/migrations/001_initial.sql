-- 001_initial.sql
-- Skema PostgreSQL untuk The Last Lighthouse (ADR-004)

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    guest_token TEXT UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS matches (
    id           TEXT PRIMARY KEY,
    status       TEXT NOT NULL DEFAULT 'lobby',
    seed         BIGINT NOT NULL,
    content_hash TEXT NOT NULL,
    player_ids   TEXT[] NOT NULL,
    turn_timeout_sec INT NOT NULL DEFAULT 90,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS match_events (
    match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    seq      BIGINT NOT NULL,
    kind     TEXT NOT NULL,
    payload  JSONB NOT NULL,
    at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, seq)
);

CREATE TABLE IF NOT EXISTS match_snapshots (
    match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    seq      BIGINT NOT NULL,
    state    JSONB NOT NULL,
    at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_match_events_seq ON match_events(match_id, seq);
