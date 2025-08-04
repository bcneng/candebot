package storage

import (
	"os"
	"testing"
	"time"
)

func TestCloseThread(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize database
	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	channelID := "C1234567890"
	threadTS := "1234567890.123456"
	userID := "U1234567890"

	// Test that thread is not closed initially
	isClosed, err := db.IsThreadClosed(channelID, threadTS)
	if err != nil {
		t.Fatalf("Error checking if thread is closed: %v", err)
	}
	if isClosed {
		t.Error("Thread should not be closed initially")
	}

	// Close the thread
	err = db.CloseThread(channelID, threadTS, userID)
	if err != nil {
		t.Fatalf("Error closing thread: %v", err)
	}

	// Verify thread is now closed
	isClosed, err = db.IsThreadClosed(channelID, threadTS)
	if err != nil {
		t.Fatalf("Error checking if thread is closed: %v", err)
	}
	if !isClosed {
		t.Error("Thread should be closed")
	}

	// Get closed thread details
	closedThread, err := db.GetClosedThread(channelID, threadTS)
	if err != nil {
		t.Fatalf("Error getting closed thread: %v", err)
	}
	if closedThread == nil {
		t.Error("Closed thread should not be nil")
	}
	if closedThread.ChannelID != channelID {
		t.Errorf("Expected channel ID %s, got %s", channelID, closedThread.ChannelID)
	}
	if closedThread.ThreadTS != threadTS {
		t.Errorf("Expected thread TS %s, got %s", threadTS, closedThread.ThreadTS)
	}
	if closedThread.ClosedBy != userID {
		t.Errorf("Expected closed by %s, got %s", userID, closedThread.ClosedBy)
	}
	if time.Since(closedThread.ClosedAt) > time.Minute {
		t.Error("ClosedAt timestamp should be recent")
	}

	// Test reopening thread
	err = db.ReopenThread(channelID, threadTS)
	if err != nil {
		t.Fatalf("Error reopening thread: %v", err)
	}

	// Verify thread is no longer closed
	isClosed, err = db.IsThreadClosed(channelID, threadTS)
	if err != nil {
		t.Fatalf("Error checking if thread is closed: %v", err)
	}
	if isClosed {
		t.Error("Thread should not be closed after reopening")
	}

	// Get closed thread details after reopening
	closedThread, err = db.GetClosedThread(channelID, threadTS)
	if err != nil {
		t.Fatalf("Error getting closed thread: %v", err)
	}
	if closedThread != nil {
		t.Error("Closed thread should be nil after reopening")
	}
}

func TestMultipleThreads(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize database
	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Close multiple threads
	err = db.CloseThread("C1", "T1", "U1")
	if err != nil {
		t.Fatalf("Error closing thread 1: %v", err)
	}

	err = db.CloseThread("C1", "T2", "U1")
	if err != nil {
		t.Fatalf("Error closing thread 2: %v", err)
	}

	err = db.CloseThread("C2", "T1", "U2")
	if err != nil {
		t.Fatalf("Error closing thread 3: %v", err)
	}

	// Check each thread
	tests := []struct {
		channel string
		thread  string
		closed  bool
	}{
		{"C1", "T1", true},
		{"C1", "T2", true},
		{"C2", "T1", true},
		{"C1", "T3", false}, // not closed
		{"C2", "T2", false}, // not closed
	}

	for _, test := range tests {
		isClosed, err := db.IsThreadClosed(test.channel, test.thread)
		if err != nil {
			t.Fatalf("Error checking thread %s/%s: %v", test.channel, test.thread, err)
		}
		if isClosed != test.closed {
			t.Errorf("Thread %s/%s: expected closed=%v, got %v", test.channel, test.thread, test.closed, isClosed)
		}
	}
}