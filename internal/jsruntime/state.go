package jsruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Size limits for state storage
const (
	MaxStatePerHandler = 5 * 1024 * 1024  // 5MB per handler
	MaxTotalState      = 50 * 1024 * 1024 // 50MB total
)

// StateStore defines the interface for handler state storage.
type StateStore interface {
	// Get retrieves a value for the given handler and key.
	// Returns the value and true if found, nil and false otherwise.
	Get(handler, key string) (interface{}, bool)

	// Set stores a value for the given handler and key.
	// Returns an error if size limits are exceeded.
	Set(handler, key string, value interface{}) error

	// Delete removes a key from the handler's state.
	Delete(handler, key string) error

	// Has checks if a key exists in the handler's state.
	Has(handler, key string) bool

	// Keys returns all keys for the given handler.
	Keys(handler string) []string

	// Clear removes all state for the given handler.
	Clear(handler string) error

	// Close performs cleanup (e.g., flushing to disk).
	Close() error
}

// MemoryStateStore implements StateStore with in-memory storage.
type MemoryStateStore struct {
	data map[string]map[string]interface{}
	mu   sync.RWMutex
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		data: make(map[string]map[string]interface{}),
	}
}

func (s *MemoryStateStore) Get(handler, key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if handlerData, ok := s.data[handler]; ok {
		if value, ok := handlerData[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func (s *MemoryStateStore) Set(handler, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[handler] == nil {
		s.data[handler] = make(map[string]interface{})
	}

	// Check size limits
	if err := s.checkSizeLimitsLocked(handler, key, value); err != nil {
		return err
	}

	s.data[handler][key] = value
	return nil
}

func (s *MemoryStateStore) Delete(handler, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if handlerData, ok := s.data[handler]; ok {
		delete(handlerData, key)
	}
	return nil
}

func (s *MemoryStateStore) Has(handler, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if handlerData, ok := s.data[handler]; ok {
		_, exists := handlerData[key]
		return exists
	}
	return false
}

func (s *MemoryStateStore) Keys(handler string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if handlerData, ok := s.data[handler]; ok {
		keys := make([]string, 0, len(handlerData))
		for k := range handlerData {
			keys = append(keys, k)
		}
		return keys
	}
	return []string{}
}

func (s *MemoryStateStore) Clear(handler string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, handler)
	return nil
}

func (s *MemoryStateStore) Close() error {
	return nil
}

// checkSizeLimitsLocked checks if adding the value would exceed limits.
// Must be called with lock held.
func (s *MemoryStateStore) checkSizeLimitsLocked(handler, key string, value interface{}) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}
	valueSize := len(valueBytes)

	// Calculate current handler size (excluding the key being set)
	handlerSize := 0
	if handlerData, ok := s.data[handler]; ok {
		for k, v := range handlerData {
			if k == key {
				continue // Skip the key being replaced
			}
			if b, err := json.Marshal(v); err == nil {
				handlerSize += len(b)
			}
		}
	}

	if handlerSize+valueSize > MaxStatePerHandler {
		return fmt.Errorf("state size limit exceeded for handler %q (max %d bytes)", handler, MaxStatePerHandler)
	}

	// Calculate total size
	totalSize := 0
	for h, handlerData := range s.data {
		for k, v := range handlerData {
			if h == handler && k == key {
				continue // Skip the key being replaced
			}
			if b, err := json.Marshal(v); err == nil {
				totalSize += len(b)
			}
		}
	}

	if totalSize+valueSize > MaxTotalState {
		return fmt.Errorf("total state size limit exceeded (max %d bytes)", MaxTotalState)
	}

	return nil
}

// GetData returns a copy of all data (for serialization).
func (s *MemoryStateStore) GetData() map[string]map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy
	result := make(map[string]map[string]interface{})
	for handler, handlerData := range s.data {
		result[handler] = make(map[string]interface{})
		for k, v := range handlerData {
			result[handler][k] = v
		}
	}
	return result
}

// SetData replaces all data (for deserialization).
func (s *MemoryStateStore) SetData(data map[string]map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = data
}

// FileStateStore implements StateStore with JSON file persistence.
type FileStateStore struct {
	*MemoryStateStore
	filePath      string
	flushInterval time.Duration
	dirty         bool
	dirtyMu       sync.Mutex
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// NewFileStateStore creates a new file-backed state store.
func NewFileStateStore(filePath string, flushInterval time.Duration) (*FileStateStore, error) {
	s := &FileStateStore{
		MemoryStateStore: NewMemoryStateStore(),
		filePath:         filePath,
		flushInterval:    flushInterval,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}

	// Load existing state from file
	if err := s.load(); err != nil {
		// If file doesn't exist, that's okay - start fresh
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load state file: %w", err)
		}
	}

	// Start background flush goroutine
	go s.flushLoop()

	return s, nil
}

func (s *FileStateStore) Set(handler, key string, value interface{}) error {
	if err := s.MemoryStateStore.Set(handler, key, value); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStateStore) Delete(handler, key string) error {
	if err := s.MemoryStateStore.Delete(handler, key); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStateStore) Clear(handler string) error {
	if err := s.MemoryStateStore.Clear(handler); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStateStore) Close() error {
	// Signal flush loop to stop
	close(s.stopCh)
	// Wait for flush loop to finish
	<-s.doneCh
	// Final flush
	return s.flush()
}

func (s *FileStateStore) markDirty() {
	s.dirtyMu.Lock()
	s.dirty = true
	s.dirtyMu.Unlock()
}

func (s *FileStateStore) isDirty() bool {
	s.dirtyMu.Lock()
	defer s.dirtyMu.Unlock()
	return s.dirty
}

func (s *FileStateStore) clearDirty() {
	s.dirtyMu.Lock()
	s.dirty = false
	s.dirtyMu.Unlock()
}

func (s *FileStateStore) flushLoop() {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.isDirty() {
				if err := s.flush(); err != nil {
					// Log error but continue
					fmt.Printf("[state] flush error: %v\n", err)
				}
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *FileStateStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var state map[string]map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	s.SetData(state)
	return nil
}

func (s *FileStateStore) flush() error {
	data := s.GetData()

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write atomically using temp file
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tempFile, s.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	s.clearDirty()
	return nil
}

// CreateStateAPI creates the state API object to be exposed to JS handlers.
func CreateStateAPI(store StateStore, handlerName string) map[string]interface{} {
	return map[string]interface{}{
		"get": func(key string) interface{} {
			if value, ok := store.Get(handlerName, key); ok {
				return value
			}
			return nil
		},
		"set": func(key string, value interface{}) error {
			return store.Set(handlerName, key, value)
		},
		"delete": func(key string) error {
			return store.Delete(handlerName, key)
		},
		"has": func(key string) bool {
			return store.Has(handlerName, key)
		},
		"keys": func() []string {
			return store.Keys(handlerName)
		},
		"clear": func() error {
			return store.Clear(handlerName)
		},
		// Global access to other handlers' state
		"global": map[string]interface{}{
			"get": func(handler, key string) interface{} {
				if value, ok := store.Get(handler, key); ok {
					return value
				}
				return nil
			},
			"set": func(handler, key string, value interface{}) error {
				return store.Set(handler, key, value)
			},
			"delete": func(handler, key string) error {
				return store.Delete(handler, key)
			},
			"has": func(handler, key string) bool {
				return store.Has(handler, key)
			},
			"keys": func(handler string) []string {
				return store.Keys(handler)
			},
		},
	}
}
