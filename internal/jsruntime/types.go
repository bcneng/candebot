// Package jsruntime provides a JavaScript runtime for extensible message handlers.
// Users can write handlers in JavaScript that are executed in a sandboxed environment.
package jsruntime

import "time"

// MessageData represents the data passed to JS handlers for message events.
type MessageData struct {
	Type            string `json:"type"`
	Channel         string `json:"channel"`
	ChannelName     string `json:"channelName"`
	ChannelType     string `json:"channelType"` // "channel", "im", "mpim", "group"
	User            string `json:"user"`
	Text            string `json:"text"`
	Timestamp       string `json:"timestamp"`
	ThreadTimestamp string `json:"threadTimestamp"`
	IsThread        bool   `json:"isThread"`
	IsDM            bool   `json:"isDM"`
	BotID           string `json:"botId"`
	SubType         string `json:"subType"`
}

// HandlerMetadata contains the metadata extracted from a JS handler file.
type HandlerMetadata struct {
	// Name is the unique identifier for this handler
	Name string `json:"name"`
	// Description describes what this handler does
	Description string `json:"description"`
	// Channels is a list of channel patterns this handler applies to.
	// Supports exact names, glob patterns (e.g., "offtopic-*"), and regex (e.g., "/^hiring/").
	// An empty list means the handler won't match any channel (opt-in required).
	Channels []string `json:"channels"`
	// Priority determines the order of handler execution (lower runs first, default 100)
	Priority int `json:"priority"`
	// Enabled allows disabling a handler without removing it
	Enabled bool `json:"enabled"`
	// Timeout is the maximum execution time in milliseconds (default 5000)
	Timeout int `json:"timeout"`
}

// HandlerResult represents the result returned by a JS handler.
type HandlerResult struct {
	// Handled indicates whether the handler processed the message
	Handled bool `json:"handled"`
	// StopPropagation prevents other handlers from running if true
	StopPropagation bool `json:"stopPropagation"`
	// Error contains any error message from the handler
	Error string `json:"error,omitempty"`
}

// Handler represents a loaded JS handler.
type Handler struct {
	Metadata   HandlerMetadata
	FilePath   string
	SourceCode string
	LoadedAt   time.Time
}

// SlackAPI defines the Slack operations available to JS handlers.
type SlackAPI struct {
	// SendMessage sends a message to a channel
	SendMessage func(channel, text string, opts map[string]interface{}) error
	// SendEphemeral sends an ephemeral message to a user in a channel
	SendEphemeral func(channel, user, text string) error
	// AddReaction adds a reaction to a message
	AddReaction func(channel, timestamp, emoji string) error
	// RemoveReaction removes a reaction from a message
	RemoveReaction func(channel, timestamp, emoji string) error
	// GetUserInfo gets information about a user
	GetUserInfo func(userID string) (map[string]interface{}, error)
	// GetChannelInfo gets information about a channel
	GetChannelInfo func(channelID string) (map[string]interface{}, error)
	// DeleteMessage deletes a message (requires admin token)
	DeleteMessage func(channel, timestamp string) error
	// UpdateMessage updates an existing message
	UpdateMessage func(channel, timestamp, text string) error
}

// HTTPAPI defines the HTTP operations available to JS handlers.
type HTTPAPI struct {
	// Get performs an HTTP GET request
	Get func(url string, opts map[string]interface{}) (map[string]interface{}, error)
	// Post performs an HTTP POST request
	Post func(url string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error)
	// Put performs an HTTP PUT request
	Put func(url string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error)
	// Delete performs an HTTP DELETE request
	Delete func(url string, opts map[string]interface{}) (map[string]interface{}, error)
	// Fetch performs a generic HTTP request (similar to JS fetch)
	Fetch func(url string, opts map[string]interface{}) (map[string]interface{}, error)
}

// LogAPI defines the logging operations available to JS handlers.
type LogAPI struct {
	// Info logs an info message
	Info func(args ...interface{})
	// Warn logs a warning message
	Warn func(args ...interface{})
	// Error logs an error message
	Error func(args ...interface{})
	// Debug logs a debug message
	Debug func(args ...interface{})
}

// RuntimeConfig holds configuration for the JS runtime.
type RuntimeConfig struct {
	// HandlersDir is the directory containing JS handler files
	HandlersDir string
	// DefaultTimeout is the default execution timeout in milliseconds
	DefaultTimeout int
	// MaxMemory is the maximum memory allowed per handler execution (bytes)
	MaxMemory int64
	// AllowedHosts is a list of hosts that handlers can make HTTP requests to (empty = all)
	AllowedHosts []string
	// BlockedHosts is a list of hosts that handlers cannot make HTTP requests to
	BlockedHosts []string
}

// DefaultRuntimeConfig returns the default runtime configuration.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		HandlersDir:    "handlers/scripts",
		DefaultTimeout: 5000,
		MaxMemory:      50 * 1024 * 1024, // 50MB
		AllowedHosts:   nil,              // Allow all
		BlockedHosts:   []string{"localhost", "127.0.0.1", "::1", "0.0.0.0"},
	}
}
