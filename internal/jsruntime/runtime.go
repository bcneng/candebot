package jsruntime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// Runtime manages the execution of JS handlers.
type Runtime struct {
	config      RuntimeConfig
	httpClient  *HTTPClient
	slackClient *SlackClient
	aiClient    *AIClient
	cacheStore  StateStore // in-memory state (volatile)
	fileStore   StateStore // file-backed state (persistent)
	handlers    []*Handler
	mu          sync.RWMutex
}

// NewRuntime creates a new JS runtime.
func NewRuntime(config RuntimeConfig, slackClient *SlackClient) *Runtime {
	return &Runtime{
		config:      config,
		httpClient:  NewHTTPClient(config),
		slackClient: slackClient,
		aiClient:    NewAIClient(config.GeminiAPIKey, time.Duration(config.DefaultTimeout)*time.Millisecond),
		handlers:    make([]*Handler, 0),
	}
}

// SetStateStores configures the state stores for the runtime.
func (r *Runtime) SetStateStores(cache, file StateStore) {
	r.cacheStore = cache
	r.fileStore = file
}

// Close cleans up runtime resources.
func (r *Runtime) Close() error {
	var errs []error
	if r.cacheStore != nil {
		if err := r.cacheStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.fileStore != nil {
		if err := r.fileStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// RegisterHandler adds a handler to the runtime.
func (r *Runtime) RegisterHandler(handler *Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, handler)
}

// GetHandlers returns all registered handlers.
func (r *Runtime) GetHandlers() []*Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Handler, len(r.handlers))
	copy(result, r.handlers)
	return result
}

// ClearHandlers removes all handlers.
func (r *Runtime) ClearHandlers() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = make([]*Handler, 0)
}

// ExecuteHandlers runs all matching handlers for a message.
func (r *Runtime) ExecuteHandlers(ctx context.Context, channelName string, message MessageData) []HandlerResult {
	r.mu.RLock()
	handlers := MatchChannelToHandlers(channelName, r.handlers)
	r.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	// Sort by priority (lower first)
	sortHandlersByPriority(handlers)

	results := make([]HandlerResult, 0, len(handlers))

	for _, handler := range handlers {
		timeout := time.Duration(handler.Metadata.Timeout) * time.Millisecond
		if timeout <= 0 {
			timeout = time.Duration(r.config.DefaultTimeout) * time.Millisecond
		}

		handlerCtx, cancel := context.WithTimeout(ctx, timeout)
		result := r.executeHandler(handlerCtx, handler, message)
		cancel()

		results = append(results, result)

		if result.StopPropagation {
			break
		}
	}

	return results
}

// executeHandler runs a single handler.
func (r *Runtime) executeHandler(ctx context.Context, handler *Handler, message MessageData) HandlerResult {
	result := HandlerResult{}

	vm := goja.New()

	// Set up interrupt for timeout
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			vm.Interrupt("handler timeout exceeded")
		case <-done:
		}
	}()
	defer close(done)

	// Set up console API
	console := createConsoleAPI(handler.Metadata.Name)
	if err := vm.Set("console", console); err != nil {
		result.Error = fmt.Sprintf("failed to set console: %v", err)
		return result
	}

	// Set up log API (alias for console)
	if err := vm.Set("log", console); err != nil {
		result.Error = fmt.Sprintf("failed to set log: %v", err)
		return result
	}

	// Set up Slack API
	if r.slackClient != nil {
		slackAPI := CreateSlackAPI(r.slackClient)
		if err := vm.Set("slack", slackAPI); err != nil {
			result.Error = fmt.Sprintf("failed to set slack: %v", err)
			return result
		}
	}

	// Set up HTTP API
	httpAPI := CreateHTTPAPI(r.httpClient, ctx)
	if err := vm.Set("http", httpAPI); err != nil {
		result.Error = fmt.Sprintf("failed to set http: %v", err)
		return result
	}

	// Set up AI API
	if r.aiClient != nil && r.aiClient.IsConfigured() {
		aiAPI := CreateAIAPI(r.aiClient, ctx)
		if err := vm.Set("ai", aiAPI); err != nil {
			result.Error = fmt.Sprintf("failed to set ai: %v", err)
			return result
		}
	}

	// Set up State API (cache + store)
	if r.cacheStore != nil && r.fileStore != nil {
		stateAPI := CreateStateAPI(r.cacheStore, r.fileStore, handler.Metadata.Name)
		if err := vm.Set("state", stateAPI); err != nil {
			result.Error = fmt.Sprintf("failed to set state: %v", err)
			return result
		}
	}

	// Run the handler source code to define the handler
	_, err := vm.RunString(handler.SourceCode)
	if err != nil {
		result.Error = fmt.Sprintf("failed to execute handler source: %v", err)
		return result
	}

	// Get the handle function
	handleFn := vm.Get("handle")
	if handleFn == nil || goja.IsUndefined(handleFn) || goja.IsNull(handleFn) {
		result.Error = "handler does not export a 'handle' function"
		return result
	}

	fn, ok := goja.AssertFunction(handleFn)
	if !ok {
		result.Error = "'handle' is not a function"
		return result
	}

	// Call the handle function with the message (including helper methods)
	msgMap := messageToMap(message)
	// Add isBot convenience property
	msgMap["isBot"] = message.BotID != ""
	// Always reply in thread: use existing thread or start new one from message
	threadTs := message.ThreadTimestamp
	if threadTs == "" {
		threadTs = message.Timestamp
	}
	// Add helper methods to message object
	if r.slackClient != nil {
		msgMap["reply"] = func(text string, opts map[string]interface{}) (map[string]interface{}, error) {
			if opts == nil {
				opts = make(map[string]interface{})
			}
			opts["threadTimestamp"] = threadTs
			return r.slackClient.SendMessage(message.Channel, text, opts)
		}
		msgMap["replyEphemeral"] = func(text string, opts map[string]interface{}) error {
			if opts == nil {
				opts = make(map[string]interface{})
			}
			opts["threadTimestamp"] = threadTs
			return r.slackClient.SendEphemeral(message.Channel, message.User, text, opts)
		}
		msgMap["react"] = func(emoji string) error {
			return r.slackClient.AddReaction(message.Channel, message.Timestamp, emoji)
		}
		msgMap["delete"] = func() error {
			return r.slackClient.DeleteMessage(message.Channel, message.Timestamp)
		}
	}
	messageVal := vm.ToValue(msgMap)
	retVal, err := fn(goja.Undefined(), messageVal)
	if err != nil {
		if interruptErr, ok := err.(*goja.InterruptedError); ok {
			result.Error = fmt.Sprintf("handler interrupted: %v", interruptErr.Value())
		} else {
			result.Error = fmt.Sprintf("handler error: %v", err)
		}
		return result
	}

	// Parse the return value
	if retVal != nil && !goja.IsUndefined(retVal) && !goja.IsNull(retVal) {
		if obj := retVal.ToObject(vm); obj != nil {
			if handled := obj.Get("handled"); handled != nil {
				result.Handled = handled.ToBoolean()
			}
			if stopProp := obj.Get("stopPropagation"); stopProp != nil {
				result.StopPropagation = stopProp.ToBoolean()
			}
		}
	}

	return result
}

func messageToMap(m MessageData) map[string]interface{} {
	return map[string]interface{}{
		"type":            m.Type,
		"channel":         m.Channel,
		"channelName":     m.ChannelName,
		"channelType":     m.ChannelType,
		"user":            m.User,
		"text":            m.Text,
		"timestamp":       m.Timestamp,
		"threadTimestamp": m.ThreadTimestamp,
		"isThread":        m.IsThread,
		"isDM":            m.IsDM,
		"botId":           m.BotID,
		"subType":         m.SubType,
	}
}

func createConsoleAPI(handlerName string) map[string]interface{} {
	prefix := fmt.Sprintf("[handler:%s]", handlerName)
	return map[string]interface{}{
		"log": func(args ...interface{}) {
			log.Println(append([]interface{}{prefix}, args...)...)
		},
		"info": func(args ...interface{}) {
			log.Println(append([]interface{}{prefix, "[INFO]"}, args...)...)
		},
		"warn": func(args ...interface{}) {
			log.Println(append([]interface{}{prefix, "[WARN]"}, args...)...)
		},
		"error": func(args ...interface{}) {
			log.Println(append([]interface{}{prefix, "[ERROR]"}, args...)...)
		},
		"debug": func(args ...interface{}) {
			log.Println(append([]interface{}{prefix, "[DEBUG]"}, args...)...)
		},
	}
}

func sortHandlersByPriority(handlers []*Handler) {
	// Simple bubble sort - handlers list is typically small
	n := len(handlers)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if handlers[j].Metadata.Priority > handlers[j+1].Metadata.Priority {
				handlers[j], handlers[j+1] = handlers[j+1], handlers[j]
			}
		}
	}
}

// ValidateHandler validates a handler's source code without executing it.
func (r *Runtime) ValidateHandler(source string) (*HandlerMetadata, error) {
	vm := goja.New()

	// Run the source to define exports
	_, err := vm.RunString(source)
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	// Check for required exports
	handleFn := vm.Get("handle")
	if handleFn == nil || goja.IsUndefined(handleFn) || goja.IsNull(handleFn) {
		return nil, fmt.Errorf("handler must export a 'handle' function")
	}

	if _, ok := goja.AssertFunction(handleFn); !ok {
		return nil, fmt.Errorf("'handle' must be a function")
	}

	// Extract metadata
	metadata := &HandlerMetadata{
		Name:     "unnamed",
		Enabled:  true,
		Priority: 100,
		Timeout:  r.config.DefaultTimeout,
	}

	if name := vm.Get("name"); name != nil && !goja.IsUndefined(name) {
		metadata.Name = name.String()
	}

	if desc := vm.Get("description"); desc != nil && !goja.IsUndefined(desc) {
		metadata.Description = desc.String()
	}

	if channels := vm.Get("channels"); channels != nil && !goja.IsUndefined(channels) {
		obj := channels.ToObject(vm)
		if obj != nil {
			// Try to iterate as array
			length := obj.Get("length")
			if length != nil && !goja.IsUndefined(length) {
				l := int(length.ToInteger())
				metadata.Channels = make([]string, 0, l)
				for i := 0; i < l; i++ {
					v := obj.Get(fmt.Sprintf("%d", i))
					if v != nil && !goja.IsUndefined(v) {
						metadata.Channels = append(metadata.Channels, v.String())
					}
				}
			}
		}
	}

	if priority := vm.Get("priority"); priority != nil && !goja.IsUndefined(priority) {
		metadata.Priority = int(priority.ToInteger())
	}

	if enabled := vm.Get("enabled"); enabled != nil && !goja.IsUndefined(enabled) {
		metadata.Enabled = enabled.ToBoolean()
	}

	if timeout := vm.Get("timeout"); timeout != nil && !goja.IsUndefined(timeout) {
		metadata.Timeout = int(timeout.ToInteger())
	}

	return metadata, nil
}
