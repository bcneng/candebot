package db

import (
	"os"
	"testing"
	"time"
)

func TestNewDatabaseManager(t *testing.T) {
	// Use in-memory database for testing
	dbManager, err := NewDatabaseManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	if dbManager.db == nil {
		t.Error("Expected db to be initialized, got nil")
	}
}

func TestNewDatabaseManagerWithInvalidPath(t *testing.T) {
	// Test with an invalid path
	_, err := NewDatabaseManager("/invalid/path/that/does/not/exist/db.sqlite")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestCloseThreadAndIsThreadClosed(t *testing.T) {
	dbManager, err := NewDatabaseManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	// Test closing a thread
	channelID := "C123456"
	threadTS := "1612345678.123456"
	closedBy := "U123456"

	// Initially, thread should not be closed
	isClosed, err := dbManager.IsThreadClosed(channelID, threadTS)
	if err != nil {
		t.Fatalf("Failed to check if thread is closed: %v", err)
	}
	if isClosed {
		t.Error("Expected thread to not be closed initially")
	}

	// Close the thread
	err = dbManager.CloseThread(channelID, threadTS, closedBy)
	if err != nil {
		t.Fatalf("Failed to close thread: %v", err)
	}

	// Now thread should be closed
	isClosed, err = dbManager.IsThreadClosed(channelID, threadTS)
	if err != nil {
		t.Fatalf("Failed to check if thread is closed: %v", err)
	}
	if !isClosed {
		t.Error("Expected thread to be closed after CloseThread")
	}
}

func TestReopenThread(t *testing.T) {
	dbManager, err := NewDatabaseManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	// Set up a closed thread
	channelID := "C123456"
	threadTS := "1612345678.123456"
	closedBy := "U123456"

	err = dbManager.CloseThread(channelID, threadTS, closedBy)
	if err != nil {
		t.Fatalf("Failed to close thread: %v", err)
	}

	// Reopen the thread
	err = dbManager.ReopenThread(channelID, threadTS)
	if err != nil {
		t.Fatalf("Failed to reopen thread: %v", err)
	}

	// Thread should no longer be closed
	isClosed, err := dbManager.IsThreadClosed(channelID, threadTS)
	if err != nil {
		t.Fatalf("Failed to check if thread is closed: %v", err)
	}
	if isClosed {
		t.Error("Expected thread to not be closed after reopening")
	}
}

func TestGetClosedThreadInfo(t *testing.T) {
	dbManager, err := NewDatabaseManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	// Set up a closed thread
	channelID := "C123456"
	threadTS := "1612345678.123456"
	closedBy := "U123456"

	// Close the thread
	err = dbManager.CloseThread(channelID, threadTS, closedBy)
	if err != nil {
		t.Fatalf("Failed to close thread: %v", err)
	}

	// Get info about the closed thread
	gotClosedBy, closedAt, err := dbManager.GetClosedThreadInfo(channelID, threadTS)
	if err != nil {
		t.Fatalf("Failed to get closed thread info: %v", err)
	}

	if gotClosedBy != closedBy {
		t.Errorf("Expected closedBy to be %s, got %s", closedBy, gotClosedBy)
	}

	// Check that closedAt is relatively recent (within last minute)
	if time.Since(closedAt) > time.Minute {
		t.Errorf("Expected closedAt to be recent, got %v", closedAt)
	}

	// Test for non-existent thread
	gotClosedBy, closedAt, err = dbManager.GetClosedThreadInfo("nonexistent", "nonexistent")
	if err != nil {
		t.Fatalf("Expected nil error for non-existent thread, got %v", err)
	}
	if gotClosedBy != "" || !closedAt.IsZero() {
		t.Errorf("Expected empty results for non-existent thread, got closedBy=%s, closedAt=%v", gotClosedBy, closedAt)
	}
}

func TestListClosedThreads(t *testing.T) {
	dbManager, err := NewDatabaseManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	// Initially, there should be no closed threads
	threads, err := dbManager.ListClosedThreads()
	if err != nil {
		t.Fatalf("Failed to list closed threads: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("Expected no closed threads initially, got %d", len(threads))
	}

	// Close some threads
	testData := []struct {
		channelID string
		threadTS  string
		closedBy  string
	}{
		{"C111", "1111.1111", "U111"},
		{"C222", "2222.2222", "U222"},
		{"C333", "3333.3333", "U333"},
	}

	for _, td := range testData {
		err = dbManager.CloseThread(td.channelID, td.threadTS, td.closedBy)
		if err != nil {
			t.Fatalf("Failed to close thread: %v", err)
		}
		// Add a small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Check list of closed threads
	threads, err = dbManager.ListClosedThreads()
	if err != nil {
		t.Fatalf("Failed to list closed threads: %v", err)
	}

	if len(threads) != len(testData) {
		t.Errorf("Expected %d closed threads, got %d", len(testData), len(threads))
	}

	// The order should be reversed (newest first) due to ORDER BY closed_at DESC
	for i := 0; i < len(testData); i++ {
		idx := len(testData) - 1 - i
		if threads[i].ChannelID != testData[idx].channelID {
			t.Errorf("Expected channelID %s at position %d, got %s", testData[idx].channelID, i, threads[i].ChannelID)
		}
		if threads[i].ThreadTS != testData[idx].threadTS {
			t.Errorf("Expected threadTS %s at position %d, got %s", testData[idx].threadTS, i, threads[i].ThreadTS)
		}
		if threads[i].ClosedBy != testData[idx].closedBy {
			t.Errorf("Expected closedBy %s at position %d, got %s", testData[idx].closedBy, i, threads[i].ClosedBy)
		}
	}
}

func TestDatabaseManagerClose(t *testing.T) {
	// Use a temporary file for testing Close()
	tmpFile, err := os.CreateTemp("", "test_db_*.sqlite")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	dbManager, err := NewDatabaseManager(tmpPath)
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}

	// Close should work without errors
	err = dbManager.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	// Attempting to use the manager after close should fail
	_, err = dbManager.IsThreadClosed("channel", "thread")
	if err == nil {
		t.Error("Expected error when using manager after Close(), got nil")
	}
}

func TestConcurrentAccess(t *testing.T) {
	dbManager, err := NewDatabaseManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	channelID := "C123456"
	threadTS := "1612345678.123456"
	closedBy := "U123456"

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			// Alternate between closing and checking if closed
			if i%2 == 0 {
				err := dbManager.CloseThread(channelID, threadTS, closedBy)
				if err != nil {
					t.Errorf("Goroutine %d: Failed to close thread: %v", i, err)
				}
			} else {
				_, err := dbManager.IsThreadClosed(channelID, threadTS)
				if err != nil {
					t.Errorf("Goroutine %d: Failed to check if thread is closed: %v", i, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}
}