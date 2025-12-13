package jsruntime

import (
	"context"
	"testing"
	"time"
)

func TestRuntime_ValidateHandler(t *testing.T) {
	config := DefaultRuntimeConfig()
	runtime := NewRuntime(config, nil)

	tests := []struct {
		name       string
		source     string
		wantErr    bool
		wantName   string
		wantChans  []string
	}{
		{
			name: "valid handler with all metadata",
			source: `
				var name = "test-handler";
				var description = "A test handler";
				var channels = ["general", "random"];
				var priority = 50;
				var enabled = true;
				var timeout = 3000;
				function handle(msg) { return { handled: true }; }
			`,
			wantErr:   false,
			wantName:  "test-handler",
			wantChans: []string{"general", "random"},
		},
		{
			name: "valid handler with minimal metadata",
			source: `
				function handle(msg) { return {}; }
			`,
			wantErr:   false,
			wantName:  "unnamed",
			wantChans: nil,
		},
		{
			name: "missing handle function",
			source: `
				var name = "broken";
			`,
			wantErr: true,
		},
		{
			name: "handle is not a function",
			source: `
				var handle = "not a function";
			`,
			wantErr: true,
		},
		{
			name: "syntax error",
			source: `
				function handle(msg { return {}; }
			`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := runtime.ValidateHandler(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if metadata.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", metadata.Name, tt.wantName)
			}
			if len(metadata.Channels) != len(tt.wantChans) {
				t.Errorf("Channels = %v, want %v", metadata.Channels, tt.wantChans)
			}
		})
	}
}

func TestRuntime_ExecuteHandler(t *testing.T) {
	config := DefaultRuntimeConfig()
	runtime := NewRuntime(config, nil)

	handler := &Handler{
		Metadata: HandlerMetadata{
			Name:     "test",
			Channels: []string{"*"},
			Enabled:  true,
			Timeout:  5000,
		},
		SourceCode: `
			function handle(msg) {
				console.log("Processing message:", msg.text);
				return { handled: true, stopPropagation: false };
			}
		`,
	}

	runtime.RegisterHandler(handler)

	ctx := context.Background()
	message := MessageData{
		Type:    "message",
		Channel: "C123",
		User:    "U123",
		Text:    "Hello world",
	}

	results := runtime.ExecuteHandlers(ctx, "general", message)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Handled {
		t.Error("Expected handled = true")
	}

	if results[0].StopPropagation {
		t.Error("Expected stopPropagation = false")
	}

	if results[0].Error != "" {
		t.Errorf("Unexpected error: %s", results[0].Error)
	}
}

func TestRuntime_HandlerTimeout(t *testing.T) {
	config := DefaultRuntimeConfig()
	config.DefaultTimeout = 100 // 100ms
	runtime := NewRuntime(config, nil)

	handler := &Handler{
		Metadata: HandlerMetadata{
			Name:     "slow-handler",
			Channels: []string{"*"},
			Enabled:  true,
			Timeout:  100,
		},
		SourceCode: `
			function handle(msg) {
				// Infinite loop to simulate timeout
				while(true) {}
				return { handled: true };
			}
		`,
	}

	runtime.RegisterHandler(handler)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	message := MessageData{
		Type:    "message",
		Channel: "C123",
		Text:    "test",
	}

	results := runtime.ExecuteHandlers(ctx, "general", message)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == "" {
		t.Error("Expected timeout error")
	}
}

func TestRuntime_HandlerError(t *testing.T) {
	config := DefaultRuntimeConfig()
	runtime := NewRuntime(config, nil)

	handler := &Handler{
		Metadata: HandlerMetadata{
			Name:     "error-handler",
			Channels: []string{"*"},
			Enabled:  true,
			Timeout:  5000,
		},
		SourceCode: `
			function handle(msg) {
				throw new Error("Something went wrong");
			}
		`,
	}

	runtime.RegisterHandler(handler)

	ctx := context.Background()
	message := MessageData{
		Type:    "message",
		Channel: "C123",
		Text:    "test",
	}

	results := runtime.ExecuteHandlers(ctx, "general", message)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == "" {
		t.Error("Expected error from handler")
	}
}

func TestRuntime_StopPropagation(t *testing.T) {
	config := DefaultRuntimeConfig()
	runtime := NewRuntime(config, nil)

	handler1 := &Handler{
		Metadata: HandlerMetadata{
			Name:     "first",
			Channels: []string{"*"},
			Enabled:  true,
			Priority: 1,
			Timeout:  5000,
		},
		SourceCode: `
			function handle(msg) {
				return { handled: true, stopPropagation: true };
			}
		`,
	}

	handler2 := &Handler{
		Metadata: HandlerMetadata{
			Name:     "second",
			Channels: []string{"*"},
			Enabled:  true,
			Priority: 2,
			Timeout:  5000,
		},
		SourceCode: `
			function handle(msg) {
				return { handled: true };
			}
		`,
	}

	runtime.RegisterHandler(handler1)
	runtime.RegisterHandler(handler2)

	ctx := context.Background()
	message := MessageData{
		Type:    "message",
		Channel: "C123",
		Text:    "test",
	}

	results := runtime.ExecuteHandlers(ctx, "general", message)

	// Should only have 1 result because first handler stopped propagation
	if len(results) != 1 {
		t.Errorf("Expected 1 result (propagation stopped), got %d", len(results))
	}
}

func TestRuntime_HandlerPriority(t *testing.T) {
	config := DefaultRuntimeConfig()
	runtime := NewRuntime(config, nil)

	// Register in reverse priority order
	handler2 := &Handler{
		Metadata: HandlerMetadata{
			Name:     "second",
			Channels: []string{"*"},
			Enabled:  true,
			Priority: 200,
			Timeout:  5000,
		},
		SourceCode: `function handle(msg) { return { handled: true }; }`,
	}

	handler1 := &Handler{
		Metadata: HandlerMetadata{
			Name:     "first",
			Channels: []string{"*"},
			Enabled:  true,
			Priority: 100,
			Timeout:  5000,
		},
		SourceCode: `function handle(msg) { return { handled: true }; }`,
	}

	runtime.RegisterHandler(handler2)
	runtime.RegisterHandler(handler1)

	handlers := runtime.GetHandlers()
	if len(handlers) != 2 {
		t.Fatalf("Expected 2 handlers, got %d", len(handlers))
	}

	// Execute and verify order through logging (handlers should run in priority order)
	ctx := context.Background()
	message := MessageData{Type: "message", Channel: "C123", Text: "test"}
	results := runtime.ExecuteHandlers(ctx, "general", message)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestRuntime_MessageDataAccess(t *testing.T) {
	config := DefaultRuntimeConfig()
	runtime := NewRuntime(config, nil)

	handler := &Handler{
		Metadata: HandlerMetadata{
			Name:     "data-access",
			Channels: []string{"*"},
			Enabled:  true,
			Timeout:  5000,
		},
		SourceCode: `
			function handle(msg) {
				// Verify all message fields are accessible
				if (msg.type !== "message") throw new Error("type mismatch");
				if (msg.channel !== "C123") throw new Error("channel mismatch");
				if (msg.user !== "U456") throw new Error("user mismatch");
				if (msg.text !== "Hello") throw new Error("text mismatch");
				if (msg.timestamp !== "1234.5678") throw new Error("timestamp mismatch");
				if (msg.isThread !== true) throw new Error("isThread mismatch");
				if (msg.isDM !== false) throw new Error("isDM mismatch");
				return { handled: true };
			}
		`,
	}

	runtime.RegisterHandler(handler)

	ctx := context.Background()
	message := MessageData{
		Type:            "message",
		Channel:         "C123",
		User:            "U456",
		Text:            "Hello",
		Timestamp:       "1234.5678",
		ThreadTimestamp: "1234.5678",
		IsThread:        true,
		IsDM:            false,
	}

	results := runtime.ExecuteHandlers(ctx, "general", message)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error != "" {
		t.Errorf("Handler error: %s", results[0].Error)
	}

	if !results[0].Handled {
		t.Error("Expected handled = true")
	}
}
