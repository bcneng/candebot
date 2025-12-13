// Handler Simulator - Interactive web-based tester for JS message handlers
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

//go:embed static/*
var staticFiles embed.FS

var (
	handlersDir string
	port        int
	cache       = make(map[string]map[string]interface{})
	store       = make(map[string]map[string]interface{})
	cacheMu     sync.RWMutex
	storeMu     sync.RWMutex
)

func main() {
	flag.StringVar(&handlersDir, "handlers", "handlers/js", "Directory containing JS handlers")
	flag.IntVar(&port, "port", 8088, "Port to run the simulator on")
	flag.Parse()

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.FS(staticFS)))
	http.HandleFunc("/api/handlers", handleListHandlers)
	http.HandleFunc("/api/handler/", handleGetHandler)
	http.HandleFunc("/api/execute", handleExecute)
	http.HandleFunc("/api/validate", handleValidate)
	http.HandleFunc("/api/state/cache", handleCache)
	http.HandleFunc("/api/state/store", handleStore)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("🚀 Handler Simulator running at http://localhost%s", addr)
	log.Printf("📁 Loading handlers from: %s", handlersDir)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// Handler represents a loaded JS handler's metadata
type Handler struct {
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Channels    string `json:"channels"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// ExecuteRequest represents a request to execute handler code
type ExecuteRequest struct {
	Code    string                 `json:"code"`
	Message map[string]interface{} `json:"message"`
}

// ExecuteResponse represents the result of handler execution
type ExecuteResponse struct {
	Success  bool                   `json:"success"`
	Result   map[string]interface{} `json:"result"`
	Logs     []LogEntry             `json:"logs"`
	Error    string                 `json:"error,omitempty"`
	Duration string                 `json:"duration"`
}

// LogEntry represents a console log entry
type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

// ValidationResponse represents code validation result
type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Metadata Handler  `json:"metadata"`
}

func handleListHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handlers := []Handler{}

	// Read all JS files from handlers directory
	files, err := os.ReadDir(handlersDir)
	if err != nil {
		// Return empty list if directory doesn't exist
		json.NewEncoder(w).Encode(handlers)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// Include both .js and .js.disabled files
		if !strings.HasSuffix(name, ".js") && !strings.HasSuffix(name, ".js.disabled") {
			continue
		}
		// Skip example files
		if strings.HasPrefix(name, "_") {
			continue
		}

		// Read and parse handler to get metadata
		content, err := os.ReadFile(filepath.Join(handlersDir, name))
		if err != nil {
			continue
		}

		h := parseHandlerMetadata(name, string(content))
		handlers = append(handlers, h)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers)
}

func handleGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/handler/")
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	// Security: prevent directory traversal
	filename = filepath.Base(filename)

	content, err := os.ReadFile(filepath.Join(handlersDir, filename))
	if err != nil {
		http.Error(w, "Handler not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	start := time.Now()
	result, logs, err := executeHandler(req.Code, req.Message)
	duration := time.Since(start)

	resp := ExecuteResponse{
		Success:  err == nil,
		Result:   result,
		Logs:     logs,
		Duration: duration.String(),
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp := validateHandler(req.Code)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCache(w http.ResponseWriter, r *http.Request) {
	handleState(w, r, &cache, &cacheMu)
}

func handleStore(w http.ResponseWriter, r *http.Request) {
	handleState(w, r, &store, &storeMu)
}

func handleState(w http.ResponseWriter, r *http.Request, state *map[string]map[string]interface{}, mu *sync.RWMutex) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(*state)

	case http.MethodPut:
		var newState map[string]map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&newState); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		mu.Lock()
		*state = newState
		mu.Unlock()
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		mu.Lock()
		*state = make(map[string]map[string]interface{})
		mu.Unlock()
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseHandlerMetadata(filename, code string) Handler {
	h := Handler{
		Filename: filename,
		Name:     strings.TrimSuffix(strings.TrimSuffix(filename, ".disabled"), ".js"),
		Enabled:  !strings.HasSuffix(filename, ".disabled"),
		Priority: 100,
	}

	// Create a VM to extract metadata
	vm := goja.New()
	_, err := vm.RunString(code)
	if err != nil {
		return h
	}

	if name := vm.Get("name"); name != nil && !goja.IsUndefined(name) {
		h.Name = name.String()
	}
	if desc := vm.Get("description"); desc != nil && !goja.IsUndefined(desc) {
		h.Description = desc.String()
	}
	if channels := vm.Get("channels"); channels != nil && !goja.IsUndefined(channels) {
		if arr, ok := channels.Export().([]interface{}); ok {
			strs := make([]string, len(arr))
			for i, v := range arr {
				strs[i] = fmt.Sprintf("%v", v)
			}
			h.Channels = strings.Join(strs, ", ")
		}
	}
	if priority := vm.Get("priority"); priority != nil && !goja.IsUndefined(priority) {
		if p, ok := priority.Export().(int64); ok {
			h.Priority = int(p)
		}
	}
	if enabled := vm.Get("enabled"); enabled != nil && !goja.IsUndefined(enabled) {
		if e, ok := enabled.Export().(bool); ok {
			h.Enabled = e && h.Enabled
		}
	}

	return h
}

func validateHandler(code string) ValidationResponse {
	resp := ValidationResponse{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check syntax by trying to compile
	vm := goja.New()
	_, err := vm.RunString(code)
	if err != nil {
		resp.Valid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("Syntax error: %v", err))
		return resp
	}

	// Extract metadata
	resp.Metadata = parseHandlerMetadata("untitled.js", code)

	// Check for required elements
	handle := vm.Get("handle")
	if handle == nil || goja.IsUndefined(handle) {
		resp.Valid = false
		resp.Errors = append(resp.Errors, "Missing required 'handle' function")
	} else if _, ok := goja.AssertFunction(handle); !ok {
		resp.Valid = false
		resp.Errors = append(resp.Errors, "'handle' must be a function")
	}

	// Warnings for best practices
	if vm.Get("name") == nil || goja.IsUndefined(vm.Get("name")) {
		resp.Warnings = append(resp.Warnings, "Handler should define a 'name' variable")
	}
	if vm.Get("channels") == nil || goja.IsUndefined(vm.Get("channels")) {
		resp.Warnings = append(resp.Warnings, "Handler should define a 'channels' array (defaults to no channels)")
	}
	if vm.Get("description") == nil || goja.IsUndefined(vm.Get("description")) {
		resp.Warnings = append(resp.Warnings, "Handler should define a 'description' variable")
	}

	return resp
}

func executeHandler(code string, message map[string]interface{}) (map[string]interface{}, []LogEntry, error) {
	logs := []LogEntry{}
	var logsMu sync.Mutex

	addLog := func(level, msg string) {
		logsMu.Lock()
		logs = append(logs, LogEntry{
			Level:   level,
			Message: msg,
			Time:    time.Now().Format("15:04:05.000"),
		})
		logsMu.Unlock()
	}

	vm := goja.New()

	// Set up console API
	console := map[string]interface{}{
		"log": func(args ...interface{}) {
			addLog("log", formatArgs(args))
		},
		"info": func(args ...interface{}) {
			addLog("info", formatArgs(args))
		},
		"warn": func(args ...interface{}) {
			addLog("warn", formatArgs(args))
		},
		"error": func(args ...interface{}) {
			addLog("error", formatArgs(args))
		},
		"debug": func(args ...interface{}) {
			addLog("debug", formatArgs(args))
		},
	}
	vm.Set("console", console)

	// Set up mock Slack API
	slack := map[string]interface{}{
		"sendMessage": func(channel, text string, opts map[string]interface{}) (map[string]interface{}, error) {
			addLog("slack", fmt.Sprintf("📤 sendMessage to %s: %s", channel, truncate(text, 100)))
			return map[string]interface{}{
				"channel":   channel,
				"timestamp": fmt.Sprintf("%d.%06d", time.Now().Unix(), time.Now().Nanosecond()/1000),
			}, nil
		},
		"sendEphemeral": func(channel, user, text string, opts map[string]interface{}) error {
			addLog("slack", fmt.Sprintf("👻 sendEphemeral to %s in %s: %s", user, channel, truncate(text, 100)))
			return nil
		},
		"addReaction": func(channel, timestamp, emoji string) error {
			addLog("slack", fmt.Sprintf("👍 addReaction :%s: to %s", emoji, timestamp))
			return nil
		},
		"removeReaction": func(channel, timestamp, emoji string) error {
			addLog("slack", fmt.Sprintf("👎 removeReaction :%s: from %s", emoji, timestamp))
			return nil
		},
		"deleteMessage": func(channel, timestamp string) error {
			addLog("slack", fmt.Sprintf("🗑️ deleteMessage %s in %s", timestamp, channel))
			return nil
		},
		"updateMessage": func(channel, timestamp, text string) error {
			addLog("slack", fmt.Sprintf("✏️ updateMessage %s: %s", timestamp, truncate(text, 100)))
			return nil
		},
		"getUserInfo": func(userID string) (map[string]interface{}, error) {
			addLog("slack", fmt.Sprintf("👤 getUserInfo %s", userID))
			return map[string]interface{}{
				"id":          userID,
				"name":        "testuser",
				"realName":    "Test User",
				"displayName": "Test User",
				"isBot":       false,
				"isAdmin":     false,
			}, nil
		},
		"getChannelInfo": func(channelID string) (map[string]interface{}, error) {
			addLog("slack", fmt.Sprintf("📢 getChannelInfo %s", channelID))
			return map[string]interface{}{
				"id":          channelID,
				"name":        "test-channel",
				"isPrivate":   false,
				"isArchived":  false,
				"memberCount": 100,
			}, nil
		},
	}
	vm.Set("slack", slack)

	// Set up mock HTTP API
	httpAPI := map[string]interface{}{
		"get": func(url string, opts map[string]interface{}) (map[string]interface{}, error) {
			addLog("http", fmt.Sprintf("🌐 GET %s", url))
			return map[string]interface{}{
				"status":     200,
				"statusText": "OK (simulated)",
				"body":       "{}",
			}, nil
		},
		"post": func(url string, body string, opts map[string]interface{}) (map[string]interface{}, error) {
			addLog("http", fmt.Sprintf("🌐 POST %s", url))
			return map[string]interface{}{
				"status":     200,
				"statusText": "OK (simulated)",
				"body":       "{}",
			}, nil
		},
	}
	vm.Set("http", httpAPI)

	// Set up state API with actual persistence
	handlerName := "simulator"
	state := map[string]interface{}{
		"cache": createStateAPI(&cache, &cacheMu, handlerName, addLog),
		"store": createStateAPI(&store, &storeMu, handlerName, addLog),
	}
	vm.Set("state", state)

	// Run the handler code
	_, err := vm.RunString(code)
	if err != nil {
		return nil, logs, fmt.Errorf("code error: %v", err)
	}

	// Get the handle function
	handleFn := vm.Get("handle")
	if handleFn == nil || goja.IsUndefined(handleFn) {
		return nil, logs, fmt.Errorf("handler must define a 'handle' function")
	}

	fn, ok := goja.AssertFunction(handleFn)
	if !ok {
		return nil, logs, fmt.Errorf("'handle' must be a function")
	}

	// Call the handler
	result, err := fn(goja.Undefined(), vm.ToValue(message))
	if err != nil {
		return nil, logs, fmt.Errorf("execution error: %v", err)
	}

	// Convert result to map
	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return map[string]interface{}{"handled": false}, logs, nil
	}

	resultMap, ok := result.Export().(map[string]interface{})
	if !ok {
		return map[string]interface{}{"handled": false}, logs, nil
	}

	return resultMap, logs, nil
}

func createStateAPI(state *map[string]map[string]interface{}, mu *sync.RWMutex, handlerName string, addLog func(string, string)) map[string]interface{} {
	return map[string]interface{}{
		"get": func(key string) interface{} {
			mu.RLock()
			defer mu.RUnlock()
			if (*state)[handlerName] == nil {
				return nil
			}
			val := (*state)[handlerName][key]
			addLog("state", fmt.Sprintf("📖 get(%s) = %v", key, val))
			return val
		},
		"set": func(key string, value interface{}) {
			mu.Lock()
			defer mu.Unlock()
			if (*state)[handlerName] == nil {
				(*state)[handlerName] = make(map[string]interface{})
			}
			(*state)[handlerName][key] = value
			addLog("state", fmt.Sprintf("📝 set(%s, %v)", key, value))
		},
		"delete": func(key string) {
			mu.Lock()
			defer mu.Unlock()
			if (*state)[handlerName] != nil {
				delete((*state)[handlerName], key)
			}
			addLog("state", fmt.Sprintf("🗑️ delete(%s)", key))
		},
		"has": func(key string) bool {
			mu.RLock()
			defer mu.RUnlock()
			if (*state)[handlerName] == nil {
				return false
			}
			_, exists := (*state)[handlerName][key]
			return exists
		},
		"keys": func() []string {
			mu.RLock()
			defer mu.RUnlock()
			if (*state)[handlerName] == nil {
				return []string{}
			}
			keys := make([]string, 0, len((*state)[handlerName]))
			for k := range (*state)[handlerName] {
				keys = append(keys, k)
			}
			return keys
		},
		"clear": func() {
			mu.Lock()
			defer mu.Unlock()
			(*state)[handlerName] = make(map[string]interface{})
			addLog("state", "🧹 clear()")
		},
	}
}

func formatArgs(args []interface{}) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = fmt.Sprintf("%v", arg)
	}
	return strings.Join(parts, " ")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
