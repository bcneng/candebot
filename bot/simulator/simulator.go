// Package simulator provides an embedded web-based handler testing UI.
package simulator

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bcneng/candebot/internal/jsruntime"
)

//go:embed static/*
var staticFiles embed.FS

// Handler represents a JS handler's metadata and code
type Handler struct {
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Enabled     bool   `json:"enabled"`
}

// Server provides HTTP handlers for the simulator
type Server struct {
	handlersDir string
	executor    *jsruntime.Executor
}

// NewServer creates a new simulator server
func NewServer(handlersDir string, runtime *jsruntime.Runtime) *Server {
	var executor *jsruntime.Executor
	if runtime != nil {
		executor = jsruntime.NewExecutor(runtime)
	}
	return &Server{handlersDir: handlersDir, executor: executor}
}

// RegisterRoutes registers the simulator routes on the given mux
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Serve static files at /_simulator/
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/_simulator/", http.StripPrefix("/_simulator/", http.FileServer(http.FS(staticFS))))

	// API endpoint to get all handlers with their code
	mux.HandleFunc("/_simulator/api/handlers", s.handleGetHandlers)

	// API endpoint to execute handler code
	mux.HandleFunc("/_simulator/api/execute", s.handleExecute)
}

func (s *Server) handleGetHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handlers := []Handler{}

	files, err := os.ReadDir(s.handlersDir)
	if err != nil {
		// Return empty list if directory doesn't exist
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(handlers)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".js") && !strings.HasSuffix(name, ".js.disabled") {
			continue
		}
		if strings.HasPrefix(name, "_") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(s.handlersDir, name))
		if err != nil {
			continue
		}

		h := Handler{
			Filename: name,
			Name:     strings.TrimSuffix(strings.TrimSuffix(name, ".disabled"), ".js"),
			Code:     string(content),
			Enabled:  !strings.HasSuffix(name, ".disabled"),
		}

		handlers = append(handlers, h)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers)
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.executor == nil {
		http.Error(w, "Production runtime not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req jsruntime.ExecuteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Execute the handler
	resp := s.executor.Execute(r.Context(), req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
