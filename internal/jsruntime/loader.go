package jsruntime

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Loader loads and manages JS handlers from a directory.
type Loader struct {
	runtime *Runtime
	dir     string
}

// NewLoader creates a new handler loader.
func NewLoader(runtime *Runtime, dir string) *Loader {
	return &Loader{
		runtime: runtime,
		dir:     dir,
	}
}

// LoadAll loads all JS handlers from the directory.
func (l *Loader) LoadAll() error {
	// Ensure directory exists
	if _, err := os.Stat(l.dir); os.IsNotExist(err) {
		log.Printf("Handlers directory %q does not exist, skipping handler loading", l.dir)
		return nil
	}

	// Find all .js files
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("failed to read handlers directory: %w", err)
	}

	// Clear existing handlers
	l.runtime.ClearHandlers()

	var loaded, failed int
	var errors []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".js") {
			continue
		}

		filePath := filepath.Join(l.dir, name)
		handler, err := l.LoadFile(filePath)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			log.Printf("Failed to load handler %s: %v", name, err)
			continue
		}

		l.runtime.RegisterHandler(handler)
		loaded++
		log.Printf("Loaded handler: %s (channels: %v, priority: %d, enabled: %v)",
			handler.Metadata.Name,
			handler.Metadata.Channels,
			handler.Metadata.Priority,
			handler.Metadata.Enabled)
	}

	log.Printf("Handler loading complete: %d loaded, %d failed", loaded, failed)

	if len(errors) > 0 {
		return fmt.Errorf("some handlers failed to load: %s", strings.Join(errors, "; "))
	}

	return nil
}

// LoadFile loads a single handler from a file.
func (l *Loader) LoadFile(filePath string) (*Handler, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	source := string(content)

	// Validate and extract metadata
	metadata, err := l.runtime.ValidateHandler(source)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Use filename as default name if not specified
	if metadata.Name == "unnamed" {
		base := filepath.Base(filePath)
		metadata.Name = strings.TrimSuffix(base, ".js")
	}

	handler := &Handler{
		Metadata:   *metadata,
		FilePath:   filePath,
		SourceCode: source,
		LoadedAt:   time.Now(),
	}

	return handler, nil
}

// Reload reloads all handlers from the directory.
func (l *Loader) Reload() error {
	log.Println("Reloading handlers...")
	return l.LoadAll()
}

// GetLoadedHandlers returns information about loaded handlers.
func (l *Loader) GetLoadedHandlers() []HandlerInfo {
	handlers := l.runtime.GetHandlers()
	infos := make([]HandlerInfo, len(handlers))

	for i, h := range handlers {
		infos[i] = HandlerInfo{
			Name:        h.Metadata.Name,
			Description: h.Metadata.Description,
			Channels:    h.Metadata.Channels,
			Priority:    h.Metadata.Priority,
			Enabled:     h.Metadata.Enabled,
			FilePath:    h.FilePath,
			LoadedAt:    h.LoadedAt,
		}
	}

	// Sort by priority
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Priority < infos[j].Priority
	})

	return infos
}

// HandlerInfo provides information about a loaded handler.
type HandlerInfo struct {
	Name        string
	Description string
	Channels    []string
	Priority    int
	Enabled     bool
	FilePath    string
	LoadedAt    time.Time
}

// String returns a string representation of the handler info.
func (h HandlerInfo) String() string {
	status := "enabled"
	if !h.Enabled {
		status = "disabled"
	}
	return fmt.Sprintf("%s (%s) - channels: %v, priority: %d",
		h.Name, status, h.Channels, h.Priority)
}
