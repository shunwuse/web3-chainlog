// Package store persists indexed guestbook events in SQLite.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Event is a decoded MessagePosted log.
type Event struct {
	BlockNumber uint64
	TxHash      string
	LogIndex    uint
	Sender      string
	Content     string
	Timestamp   uint64 // block timestamp, in seconds
}

// Message is a guestbook entry as served to the API.
type Message struct {
	Sender      string
	Content     string
	BlockNumber uint64
	Timestamp   uint64 // block timestamp, in seconds
}

// Store is the SQLite persistence layer.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database at path and applies the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite allows a single writer; one connection avoids SQLITE_BUSY between
	// the indexer and the API.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// InsertEvent stores a decoded log together with its read-model message in one
// transaction. It is idempotent on (tx_hash, log_index): a replayed log is
// ignored and does not duplicate the message.
func (s *Store) InsertEvent(ctx context.Context, event Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO events (block_number, tx_hash, log_index, sender, content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (tx_hash, log_index) DO NOTHING`,
		event.BlockNumber, event.TxHash, event.LogIndex, event.Sender, event.Content, event.Timestamp)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return nil // already indexed; skip the message too
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO messages (sender, content, block_number, created_at) VALUES (?, ?, ?, ?)`,
		event.Sender, event.Content, event.BlockNumber, event.Timestamp); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListMessages returns messages in posting order.
func (s *Store) ListMessages(ctx context.Context, limit, offset int) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sender, content, block_number, created_at
		 FROM messages ORDER BY id LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.Sender, &message.Content, &message.BlockNumber, &message.Timestamp); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return messages, nil
}

// GetLastBlock returns the indexer cursor, or 0 when nothing was indexed yet.
func (s *Store) GetLastBlock(ctx context.Context) (uint64, error) {
	var lastBlock uint64
	err := s.db.QueryRowContext(ctx, `SELECT last_block FROM sync_state WHERE id = 1`).Scan(&lastBlock)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get last block: %w", err)
	}
	return lastBlock, nil
}

// SetLastBlock stores the indexer cursor.
func (s *Store) SetLastBlock(ctx context.Context, block uint64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sync_state (id, last_block) VALUES (1, ?)
		 ON CONFLICT (id) DO UPDATE SET last_block = excluded.last_block`,
		block)
	if err != nil {
		return fmt.Errorf("set last block: %w", err)
	}
	return nil
}
