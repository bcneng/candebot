package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// ClosedThread represents a closed thread record
type ClosedThread struct {
	ChannelID   string
	ThreadTS    string
	ClosedAt    time.Time
	ClosedBy    string
}

// NewDB creates a new database connection and initializes the schema
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates the necessary tables
func (db *DB) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS closed_threads (
		channel_id TEXT NOT NULL,
		thread_ts TEXT NOT NULL,
		closed_at DATETIME NOT NULL,
		closed_by TEXT NOT NULL,
		PRIMARY KEY (channel_id, thread_ts)
	);
	`

	_, err := db.conn.Exec(query)
	return err
}

// CloseThread marks a thread as closed
func (db *DB) CloseThread(channelID, threadTS, closedBy string) error {
	query := `
	INSERT OR REPLACE INTO closed_threads (channel_id, thread_ts, closed_at, closed_by)
	VALUES (?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query, channelID, threadTS, time.Now(), closedBy)
	return err
}

// IsThreadClosed checks if a thread is closed
func (db *DB) IsThreadClosed(channelID, threadTS string) (bool, error) {
	query := `
	SELECT 1 FROM closed_threads 
	WHERE channel_id = ? AND thread_ts = ?
	`

	var exists int
	err := db.conn.QueryRow(query, channelID, threadTS).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReopenThread removes a thread from the closed threads list
func (db *DB) ReopenThread(channelID, threadTS string) error {
	query := `
	DELETE FROM closed_threads 
	WHERE channel_id = ? AND thread_ts = ?
	`

	_, err := db.conn.Exec(query, channelID, threadTS)
	return err
}

// GetClosedThread retrieves details about a closed thread
func (db *DB) GetClosedThread(channelID, threadTS string) (*ClosedThread, error) {
	query := `
	SELECT channel_id, thread_ts, closed_at, closed_by 
	FROM closed_threads 
	WHERE channel_id = ? AND thread_ts = ?
	`

	var ct ClosedThread
	err := db.conn.QueryRow(query, channelID, threadTS).Scan(
		&ct.ChannelID, &ct.ThreadTS, &ct.ClosedAt, &ct.ClosedBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ct, nil
}