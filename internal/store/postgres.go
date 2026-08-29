package store

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_initial.sql
var initialMigrationSQL string

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run initial migrations: %w", err)
	}

	return s, nil
}

func (p *PostgresStore) Migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, initialMigrationSQL)
	return err
}

func (p *PostgresStore) CreateMatch(ctx context.Context, m MatchRecord) error {
	query := `
		INSERT INTO matches (id, status, seed, content_hash, player_ids, turn_timeout_sec, created_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	timeoutSec := m.TurnTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 90
	}
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := p.pool.Exec(ctx, query, m.ID, m.Status, m.Seed, m.ContentHash, m.PlayerIDs, timeoutSec, createdAt, m.FinishedAt)
	if err != nil {
		return fmt.Errorf("create match: %w", err)
	}
	return nil
}

func (p *PostgresStore) GetMatch(ctx context.Context, id string) (*MatchRecord, error) {
	query := `
		SELECT id, status, seed, content_hash, player_ids, turn_timeout_sec, created_at, finished_at
		FROM matches
		WHERE id = $1
	`
	var m MatchRecord
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Status, &m.Seed, &m.ContentHash, &m.PlayerIDs, &m.TurnTimeoutSec, &m.CreatedAt, &m.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get match %s: %w", id, err)
	}
	return &m, nil
}

func (p *PostgresStore) ListMatches(ctx context.Context, status string) ([]MatchRecord, error) {
	var query string
	var rows pgx.Rows
	var err error

	if status != "" {
		query = `
			SELECT id, status, seed, content_hash, player_ids, turn_timeout_sec, created_at, finished_at
			FROM matches
			WHERE status = $1
			ORDER BY created_at DESC
		`
		rows, err = p.pool.Query(ctx, query, status)
	} else {
		query = `
			SELECT id, status, seed, content_hash, player_ids, turn_timeout_sec, created_at, finished_at
			FROM matches
			ORDER BY created_at DESC
		`
		rows, err = p.pool.Query(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("list matches: %w", err)
	}
	defer rows.Close()

	var out []MatchRecord
	for rows.Next() {
		var m MatchRecord
		if err := rows.Scan(&m.ID, &m.Status, &m.Seed, &m.ContentHash, &m.PlayerIDs, &m.TurnTimeoutSec, &m.CreatedAt, &m.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan match: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (p *PostgresStore) UpdateMatchStatus(ctx context.Context, id string, status string) error {
	var query string
	var err error
	if status == "won" || status == "lost" {
		now := time.Now()
		query = `UPDATE matches SET status = $1, finished_at = $2 WHERE id = $3`
		_, err = p.pool.Exec(ctx, query, status, now, id)
	} else {
		query = `UPDATE matches SET status = $1 WHERE id = $2`
		_, err = p.pool.Exec(ctx, query, status, id)
	}
	if err != nil {
		return fmt.Errorf("update match status %s to %s: %w", id, status, err)
	}
	return nil
}

func (p *PostgresStore) AppendEvents(ctx context.Context, matchID string, events []EventRecord) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO match_events (match_id, seq, kind, payload, at)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, e := range events {
		at := e.At
		if at.IsZero() {
			at = time.Now()
		}
		_, err := tx.Exec(ctx, query, matchID, e.Seq, e.Kind, e.Payload, at)
		if err != nil {
			return fmt.Errorf("insert event seq %d for match %s: %w", e.Seq, matchID, err)
		}
	}

	return tx.Commit(ctx)
}

func (p *PostgresStore) LoadEvents(ctx context.Context, matchID string, afterSeq int64) ([]EventRecord, error) {
	query := `
		SELECT match_id, seq, kind, payload, at
		FROM match_events
		WHERE match_id = $1 AND seq > $2
		ORDER BY seq ASC
	`
	rows, err := p.pool.Query(ctx, query, matchID, afterSeq)
	if err != nil {
		return nil, fmt.Errorf("load events for match %s after %d: %w", matchID, afterSeq, err)
	}
	defer rows.Close()

	var out []EventRecord
	for rows.Next() {
		var e EventRecord
		if err := rows.Scan(&e.MatchID, &e.Seq, &e.Kind, &e.Payload, &e.At); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (p *PostgresStore) SaveSnapshot(ctx context.Context, matchID string, seq int64, state []byte) error {
	query := `
		INSERT INTO match_snapshots (match_id, seq, state, at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (match_id, seq) DO UPDATE SET state = EXCLUDED.state, at = EXCLUDED.at
	`
	_, err := p.pool.Exec(ctx, query, matchID, seq, state, time.Now())
	if err != nil {
		return fmt.Errorf("save snapshot seq %d for match %s: %w", seq, matchID, err)
	}
	return nil
}

func (p *PostgresStore) LoadLatestSnapshot(ctx context.Context, matchID string) (*SnapshotRecord, error) {
	query := `
		SELECT match_id, seq, state, at
		FROM match_snapshots
		WHERE match_id = $1
		ORDER BY seq DESC
		LIMIT 1
	`
	var s SnapshotRecord
	err := p.pool.QueryRow(ctx, query, matchID).Scan(&s.MatchID, &s.Seq, &s.State, &s.At)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load latest snapshot for match %s: %w", matchID, err)
	}
	return &s, nil
}

func (p *PostgresStore) CreateUser(ctx context.Context, u UserRecord) error {
	query := `
		INSERT INTO users (id, display_name, guest_token, created_at)
		VALUES ($1, $2, $3, $4)
	`
	createdAt := u.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := p.pool.Exec(ctx, query, u.ID, u.DisplayName, u.GuestToken, createdAt)
	if err != nil {
		return fmt.Errorf("create user %s: %w", u.ID, err)
	}
	return nil
}

func (p *PostgresStore) GetUser(ctx context.Context, id string) (*UserRecord, error) {
	query := `SELECT id, display_name, guest_token, created_at FROM users WHERE id = $1`
	var u UserRecord
	err := p.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.DisplayName, &u.GuestToken, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user %s: %w", id, err)
	}
	return &u, nil
}

func (p *PostgresStore) GetUserByToken(ctx context.Context, token string) (*UserRecord, error) {
	query := `SELECT id, display_name, guest_token, created_at FROM users WHERE guest_token = $1`
	var u UserRecord
	err := p.pool.QueryRow(ctx, query, token).Scan(&u.ID, &u.DisplayName, &u.GuestToken, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by token: %w", err)
	}
	return &u, nil
}

func (p *PostgresStore) Close() error {
	p.pool.Close()
	return nil
}
