package jsruntime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryStateStore_BasicOperations(t *testing.T) {
	store := NewMemoryStateStore()

	// Test Set and Get
	err := store.Set("handler1", "key1", "value1")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := store.Get("handler1", "key1")
	if !ok {
		t.Fatal("Get returned not found")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Test Has
	if !store.Has("handler1", "key1") {
		t.Error("Has returned false for existing key")
	}
	if store.Has("handler1", "nonexistent") {
		t.Error("Has returned true for nonexistent key")
	}

	// Test Keys
	store.Set("handler1", "key2", "value2")
	keys := store.Keys("handler1")
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Test Delete
	store.Delete("handler1", "key1")
	if store.Has("handler1", "key1") {
		t.Error("Key still exists after delete")
	}

	// Test Clear
	store.Clear("handler1")
	if len(store.Keys("handler1")) != 0 {
		t.Error("Keys still exist after clear")
	}
}

func TestMemoryStateStore_ComplexValues(t *testing.T) {
	store := NewMemoryStateStore()

	// Test storing various types
	store.Set("handler1", "string", "hello")
	store.Set("handler1", "number", 42)
	store.Set("handler1", "float", 3.14)
	store.Set("handler1", "bool", true)
	store.Set("handler1", "array", []interface{}{1, 2, 3})
	store.Set("handler1", "object", map[string]interface{}{"foo": "bar"})

	// Verify retrieval
	if val, _ := store.Get("handler1", "string"); val != "hello" {
		t.Errorf("String mismatch: %v", val)
	}
	if val, _ := store.Get("handler1", "number"); val != 42 {
		t.Errorf("Number mismatch: %v", val)
	}
	if val, _ := store.Get("handler1", "bool"); val != true {
		t.Errorf("Bool mismatch: %v", val)
	}
}

func TestMemoryStateStore_HandlerIsolation(t *testing.T) {
	store := NewMemoryStateStore()

	store.Set("handler1", "key", "value1")
	store.Set("handler2", "key", "value2")

	val1, _ := store.Get("handler1", "key")
	val2, _ := store.Get("handler2", "key")

	if val1 != "value1" {
		t.Errorf("Handler1 value mismatch: %v", val1)
	}
	if val2 != "value2" {
		t.Errorf("Handler2 value mismatch: %v", val2)
	}

	// Clear one handler shouldn't affect the other
	store.Clear("handler1")
	if store.Has("handler1", "key") {
		t.Error("Handler1 key still exists after clear")
	}
	if !store.Has("handler2", "key") {
		t.Error("Handler2 key was incorrectly cleared")
	}
}

func TestFileStateStore_Persistence(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create store and add data
	store1, err := NewFileStateStore(stateFile, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	store1.Set("handler1", "key1", "value1")
	store1.Set("handler1", "key2", map[string]interface{}{"nested": "data"})

	// Close to flush
	if err := store1.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Create new store and verify data loaded
	store2, err := NewFileStateStore(stateFile, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}
	defer store2.Close()

	val, ok := store2.Get("handler1", "key1")
	if !ok {
		t.Fatal("Key not found after reload")
	}
	if val != "value1" {
		t.Errorf("Value mismatch after reload: %v", val)
	}

	val2, ok := store2.Get("handler1", "key2")
	if !ok {
		t.Fatal("Nested key not found after reload")
	}
	if m, ok := val2.(map[string]interface{}); !ok || m["nested"] != "data" {
		t.Errorf("Nested value mismatch after reload: %v", val2)
	}
}

func TestFileStateStore_AutoFlush(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	store, err := NewFileStateStore(stateFile, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	store.Set("handler1", "key", "value")

	// Wait for auto-flush
	time.Sleep(100 * time.Millisecond)

	// Check file was written
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	if len(data) == 0 {
		t.Error("State file is empty after auto-flush")
	}
}

func TestStateStore_SizeLimits(t *testing.T) {
	store := NewMemoryStateStore()

	// Create a large value (over 5MB per handler limit)
	largeValue := make([]byte, 6*1024*1024)

	err := store.Set("handler1", "large", string(largeValue))
	if err == nil {
		t.Error("Expected error for value exceeding size limit")
	}
}

func TestCreateStateAPI(t *testing.T) {
	cache := NewMemoryStateStore()
	fileStore := NewMemoryStateStore() // Use memory for testing

	api := CreateStateAPI(cache, fileStore, "test-handler")

	// Test cache operations
	cacheAPI := api["cache"].(map[string]interface{})

	setFn := cacheAPI["set"].(func(string, interface{}) error)
	getFn := cacheAPI["get"].(func(string) interface{})
	hasFn := cacheAPI["has"].(func(string) bool)

	if err := setFn("mykey", "myvalue"); err != nil {
		t.Fatalf("cache.set failed: %v", err)
	}

	if !hasFn("mykey") {
		t.Error("cache.has returned false")
	}

	if val := getFn("mykey"); val != "myvalue" {
		t.Errorf("cache.get returned wrong value: %v", val)
	}

	// Test store operations
	storeAPI := api["store"].(map[string]interface{})

	setFn2 := storeAPI["set"].(func(string, interface{}) error)
	getFn2 := storeAPI["get"].(func(string) interface{})

	if err := setFn2("persistent", 123); err != nil {
		t.Fatalf("store.set failed: %v", err)
	}

	if val := getFn2("persistent"); val != 123 {
		t.Errorf("store.get returned wrong value: %v", val)
	}

	// Verify isolation between cache and store
	if val := getFn("persistent"); val != nil {
		t.Error("cache should not see store keys")
	}
}

func TestCreateStateAPI_GlobalAccess(t *testing.T) {
	cache := NewMemoryStateStore()
	fileStore := NewMemoryStateStore()

	// Set up data for handler1
	cache.Set("handler1", "shared", "data1")

	// Create API for handler2
	api := CreateStateAPI(cache, fileStore, "handler2")
	cacheAPI := api["cache"].(map[string]interface{})
	globalAPI := cacheAPI["global"].(map[string]interface{})

	getFn := globalAPI["get"].(func(string, string) interface{})

	// Handler2 should be able to read handler1's data via global
	if val := getFn("handler1", "shared"); val != "data1" {
		t.Errorf("global.get failed: %v", val)
	}
}
