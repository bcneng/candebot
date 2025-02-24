package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseManager manages the database connection and operations
type DatabaseManager struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(dbPath string) (*DatabaseManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &DatabaseManager{
		db:   db,
		path: dbPath,
	}, nil
}

// initDB creates the necessary tables if they don't exist
func initDB(db *sql.DB) error {
	// Create a table for closed threads
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS closed_threads (
			channel_id TEXT NOT NULL,
			thread_ts TEXT NOT NULL,
			closed_by TEXT NOT NULL,
			closed_at TIMESTAMP NOT NULL,
			PRIMARY KEY (channel_id, thread_ts)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create closed_threads table: %w", err)
	}

	return nil
}

// CloseThread marks a thread as closed
func (m *DatabaseManager) CloseThread(channelID, threadTS, closedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec(
		"INSERT OR REPLACE INTO closed_threads (channel_id, thread_ts, closed_by, closed_at) VALUES (?, ?, ?, ?)",
		channelID, threadTS, closedBy, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to close thread: %w", err)
	}

	return nil
}

// IsThreadClosed checks if a thread is closed
func (m *DatabaseManager) IsThreadClosed(channelID, threadTS string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int
	err := m.db.QueryRow(
		"SELECT COUNT(*) FROM closed_threads WHERE channel_id = ? AND thread_ts = ?",
		channelID, threadTS,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if thread is closed: %w", err)
	}

	return count > 0, nil
}

// ReopenThread removes a thread from the closed threads list
func (m *DatabaseManager) ReopenThread(channelID, threadTS string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec(
		"DELETE FROM closed_threads WHERE channel_id = ? AND thread_ts = ?",
		channelID, threadTS,
	)
	if err != nil {
		return fmt.Errorf("failed to reopen thread: %w", err)
	}

	return nil
}

// Close closes the database connection
func (m *DatabaseManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}

// GetClosedThreadInfo returns information about a closed thread
func (m *DatabaseManager) GetClosedThreadInfo(channelID, threadTS string) (closedBy string, closedAt time.Time, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	err = m.db.QueryRow(
		"SELECT closed_by, closed_at FROM closed_threads WHERE channel_id = ? AND thread_ts = ?",
		channelID, threadTS,
	).Scan(&closedBy, &closedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, nil
		}
		return "", time.Time{}, fmt.Errorf("failed to get closed thread info: %w", err)
	}

	return closedBy, closedAt, nil
}

// ListClosedThreads returns a list of all closed threads
func (m *DatabaseManager) ListClosedThreads() ([]struct {
	ChannelID string
	ThreadTS  string
	ClosedBy  string
	ClosedAt  time.Time
}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows, err := m.db.Query("SELECT channel_id, thread_ts, closed_by, closed_at FROM closed_threads ORDER BY closed_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to list closed threads: %w", err)
	}
	defer rows.Close()

	var result []struct {
		ChannelID string
		ThreadTS  string
		ClosedBy  string
		ClosedAt  time.Time
	}

	for rows.Next() {
		var thread struct {
			ChannelID string
			ThreadTS  string
			ClosedBy  string
			ClosedAt  time.Time
		}
		if err := rows.Scan(&thread.ChannelID, &thread.ThreadTS, &thread.ClosedBy, &thread.ClosedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result = append(result, thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}