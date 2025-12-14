package jsruntime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// ExecuteRequest represents a request to execute handler code.
type ExecuteRequest struct {
	Code    string      `json:"code"`
	Message MessageData `json:"message"`
	Options struct {
		MockAPIs bool `json:"mockAPIs"`
		DryRun   bool `json:"dryRun"`
	} `json:"options"`
}

// ExecuteResponse represents the result of executing handler code.
type ExecuteResponse struct {
	Result      HandlerResult `json:"result"`
	APICalls    []APICall     `json:"apiCalls"`
	ConsoleLogs []ConsoleLog  `json:"consoleLogs"`
	Duration    int64         `json:"duration"`
	Error       string        `json:"error,omitempty"`
}

// APICall represents a recorded API call.
type APICall struct {
	Type      string      `json:"type"`
	Method    string      `json:"method"`
	Args      interface{} `json:"args"`
	Response  interface{} `json:"response,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ConsoleLog represents a captured console log entry.
type ConsoleLog struct {
	Level string    `json:"level"`
	Args  []string  `json:"args"`
	Time  time.Time `json:"time"`
}

// Executor handles execution of handler code for testing.
type Executor struct {
	runtime *Runtime
}

// NewExecutor creates a new executor.
func NewExecutor(runtime *Runtime) *Executor {
	return &Executor{runtime: runtime}
}

// Execute runs handler code and returns captured results.
func (e *Executor) Execute(ctx context.Context, req ExecuteRequest) ExecuteResponse {
	start := time.Now()
	resp := ExecuteResponse{
		APICalls:    make([]APICall, 0),
		ConsoleLogs: make([]ConsoleLog, 0),
	}

	// Create recording clients
	recorder := &APIRecorder{
		calls:    &resp.APICalls,
		mockMode: req.Options.MockAPIs,
		dryRun:   req.Options.DryRun,
	}

	consoleCapture := &ConsoleCapture{
		logs: &resp.ConsoleLogs,
	}

	// Set up timeout
	timeout := time.Duration(e.runtime.config.DefaultTimeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute
	result, err := e.executeCode(ctx, req.Code, req.Message, recorder, consoleCapture)
	if err != nil {
		resp.Error = err.Error()
	}
	resp.Result = result
	resp.Duration = time.Since(start).Milliseconds()

	return resp
}

func (e *Executor) executeCode(ctx context.Context, code string, message MessageData, recorder *APIRecorder, console *ConsoleCapture) (HandlerResult, error) {
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

	// Set up console API with capture
	if err := vm.Set("console", console.CreateAPI()); err != nil {
		return result, fmt.Errorf("failed to set console: %w", err)
	}

	// Set up Slack API with recording
	slackAPI := recorder.CreateSlackAPI(e.runtime.slackClient)
	if err := vm.Set("slack", slackAPI); err != nil {
		return result, fmt.Errorf("failed to set slack: %w", err)
	}

	// Set up HTTP API with recording
	httpAPI := recorder.CreateHTTPAPI(e.runtime.httpClient, ctx)
	if err := vm.Set("http", httpAPI); err != nil {
		return result, fmt.Errorf("failed to set http: %w", err)
	}

	// Set up AI API with recording
	if e.runtime.aiClient != nil {
		aiAPI := recorder.CreateAIAPI(e.runtime.aiClient, ctx)
		if err := vm.Set("ai", aiAPI); err != nil {
			return result, fmt.Errorf("failed to set ai: %w", err)
		}
	}

	// Set up State API
	if e.runtime.cacheStore != nil && e.runtime.fileStore != nil {
		stateAPI := CreateStateAPI(e.runtime.cacheStore, e.runtime.fileStore, "test-handler")
		if err := vm.Set("state", stateAPI); err != nil {
			return result, fmt.Errorf("failed to set state: %w", err)
		}
	}

	// Set up return value constants
	if err := vm.Set("SKIP", map[string]interface{}{"handled": false}); err != nil {
		return result, fmt.Errorf("failed to set SKIP: %w", err)
	}
	if err := vm.Set("HANDLED", map[string]interface{}{"handled": true}); err != nil {
		return result, fmt.Errorf("failed to set HANDLED: %w", err)
	}
	if err := vm.Set("STOP", map[string]interface{}{"handled": true, "stopPropagation": true}); err != nil {
		return result, fmt.Errorf("failed to set STOP: %w", err)
	}

	// Run the handler source code
	_, err := vm.RunString(code)
	if err != nil {
		return result, fmt.Errorf("syntax error: %w", err)
	}

	// Get the handler object
	handlerVal := vm.Get("handler")
	if handlerVal == nil || goja.IsUndefined(handlerVal) || goja.IsNull(handlerVal) {
		return result, fmt.Errorf("handler object not defined")
	}

	handlerObj := handlerVal.ToObject(vm)
	handleFn := handlerObj.Get("handle")
	if handleFn == nil || goja.IsUndefined(handleFn) || goja.IsNull(handleFn) {
		return result, fmt.Errorf("handler.handle function not defined")
	}

	fn, ok := goja.AssertFunction(handleFn)
	if !ok {
		return result, fmt.Errorf("handler.handle is not a function")
	}

	// Build message with helper methods
	msgMap := messageToMap(message)
	msgMap["isBot"] = message.BotID != ""

	// Always reply in thread
	threadTs := message.ThreadTimestamp
	if threadTs == "" {
		threadTs = message.Timestamp
	}

	// Add helper methods using recording slack API
	msgMap["reply"] = func(text string, opts map[string]interface{}) (map[string]interface{}, error) {
		if opts == nil {
			opts = make(map[string]interface{})
		}
		opts["threadTimestamp"] = threadTs
		args := map[string]interface{}{"channel": message.Channel, "text": text, "opts": opts}
		if recorder.mockMode || recorder.dryRun {
			resp := map[string]interface{}{"channel": message.Channel, "timestamp": fmt.Sprintf("mock.%d", time.Now().UnixNano())}
			recorder.record("slack", "sendMessage", args, resp, nil)
			return resp, nil
		}
		resp, err := e.runtime.slackClient.SendMessage(message.Channel, text, opts)
		recorder.record("slack", "sendMessage", args, resp, err)
		return resp, err
	}
	msgMap["replyEphemeral"] = func(text string, opts map[string]interface{}) error {
		if opts == nil {
			opts = make(map[string]interface{})
		}
		opts["threadTimestamp"] = threadTs
		args := map[string]interface{}{"channel": message.Channel, "user": message.User, "text": text}
		if recorder.mockMode || recorder.dryRun {
			recorder.record("slack", "sendEphemeral", args, nil, nil)
			return nil
		}
		err := e.runtime.slackClient.SendEphemeral(message.Channel, message.User, text, opts)
		recorder.record("slack", "sendEphemeral", args, nil, err)
		return err
	}
	msgMap["react"] = func(emoji string) error {
		args := map[string]interface{}{"channel": message.Channel, "timestamp": message.Timestamp, "emoji": emoji}
		if recorder.mockMode || recorder.dryRun {
			recorder.record("slack", "addReaction", args, nil, nil)
			return nil
		}
		err := e.runtime.slackClient.AddReaction(message.Channel, message.Timestamp, emoji)
		recorder.record("slack", "addReaction", args, nil, err)
		return err
	}
	msgMap["delete"] = func() error {
		args := map[string]interface{}{"channel": message.Channel, "timestamp": message.Timestamp}
		if recorder.mockMode || recorder.dryRun {
			recorder.record("slack", "deleteMessage", args, nil, nil)
			return nil
		}
		err := e.runtime.slackClient.DeleteMessage(message.Channel, message.Timestamp)
		recorder.record("slack", "deleteMessage", args, nil, err)
		return err
	}

	// Call handler
	messageVal := vm.ToValue(msgMap)
	retVal, err := fn(goja.Undefined(), messageVal)
	if err != nil {
		if interruptErr, ok := err.(*goja.InterruptedError); ok {
			return result, fmt.Errorf("timeout: %v", interruptErr.Value())
		}
		return result, fmt.Errorf("handler error: %w", err)
	}

	// Parse return value
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

	return result, nil
}

// APIRecorder records API calls for testing.
type APIRecorder struct {
	calls    *[]APICall
	mu       sync.Mutex
	mockMode bool
	dryRun   bool
}

func (r *APIRecorder) record(apiType, method string, args interface{}, response interface{}, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call := APICall{
		Type:      apiType,
		Method:    method,
		Args:      args,
		Response:  response,
		Timestamp: time.Now(),
	}
	if err != nil {
		call.Error = err.Error()
	}
	*r.calls = append(*r.calls, call)
}

func (r *APIRecorder) RecordSlackCall(method string, client *SlackClient, fn func() (interface{}, error), args interface{}) (interface{}, error) {
	resp, err := fn()
	r.record("slack", method, args, resp, err)
	return resp, err
}

func (r *APIRecorder) CreateSlackAPI(client *SlackClient) map[string]interface{} {
	return map[string]interface{}{
		"sendMessage": func(channel, text string, opts map[string]interface{}) (map[string]interface{}, error) {
			args := map[string]interface{}{"channel": channel, "text": text, "opts": opts}
			if r.mockMode || r.dryRun {
				resp := map[string]interface{}{"channel": channel, "timestamp": fmt.Sprintf("mock.%d", time.Now().UnixNano())}
				r.record("slack", "sendMessage", args, resp, nil)
				return resp, nil
			}
			resp, err := client.SendMessage(channel, text, opts)
			r.record("slack", "sendMessage", args, resp, err)
			return resp, err
		},
		"sendEphemeral": func(channel, user, text string, opts map[string]interface{}) error {
			args := map[string]interface{}{"channel": channel, "user": user, "text": text}
			if r.mockMode || r.dryRun {
				r.record("slack", "sendEphemeral", args, nil, nil)
				return nil
			}
			err := client.SendEphemeral(channel, user, text, opts)
			r.record("slack", "sendEphemeral", args, nil, err)
			return err
		},
		"addReaction": func(channel, timestamp, emoji string) error {
			args := map[string]interface{}{"channel": channel, "timestamp": timestamp, "emoji": emoji}
			if r.mockMode || r.dryRun {
				r.record("slack", "addReaction", args, nil, nil)
				return nil
			}
			err := client.AddReaction(channel, timestamp, emoji)
			r.record("slack", "addReaction", args, nil, err)
			return err
		},
		"removeReaction": func(channel, timestamp, emoji string) error {
			args := map[string]interface{}{"channel": channel, "timestamp": timestamp, "emoji": emoji}
			if r.mockMode || r.dryRun {
				r.record("slack", "removeReaction", args, nil, nil)
				return nil
			}
			err := client.RemoveReaction(channel, timestamp, emoji)
			r.record("slack", "removeReaction", args, nil, err)
			return err
		},
		"deleteMessage": func(channel, timestamp string) error {
			args := map[string]interface{}{"channel": channel, "timestamp": timestamp}
			if r.mockMode || r.dryRun {
				r.record("slack", "deleteMessage", args, nil, nil)
				return nil
			}
			err := client.DeleteMessage(channel, timestamp)
			r.record("slack", "deleteMessage", args, nil, err)
			return err
		},
		"getUserInfo": func(userID string) (map[string]interface{}, error) {
			args := map[string]interface{}{"userId": userID}
			if r.mockMode {
				resp := map[string]interface{}{"id": userID, "name": "mockuser", "realName": "Mock User", "isBot": false}
				r.record("slack", "getUserInfo", args, resp, nil)
				return resp, nil
			}
			resp, err := client.GetUserInfo(userID)
			r.record("slack", "getUserInfo", args, resp, err)
			return resp, err
		},
		"getChannelInfo": func(channelID string) (map[string]interface{}, error) {
			args := map[string]interface{}{"channelId": channelID}
			if r.mockMode {
				resp := map[string]interface{}{"id": channelID, "name": "mock-channel", "isPrivate": false}
				r.record("slack", "getChannelInfo", args, resp, nil)
				return resp, nil
			}
			resp, err := client.GetChannelInfo(channelID)
			r.record("slack", "getChannelInfo", args, resp, err)
			return resp, err
		},
	}
}

func (r *APIRecorder) CreateHTTPAPI(client *HTTPClient, ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"get": func(url string, opts map[string]interface{}) (map[string]interface{}, error) {
			args := map[string]interface{}{"url": url, "opts": opts}
			if r.mockMode {
				resp := map[string]interface{}{"status": 200, "body": "Mock response body"}
				r.record("http", "get", args, resp, nil)
				return resp, nil
			}
			resp, err := client.Get(ctx, url, opts)
			r.record("http", "get", args, resp, err)
			return resp, err
		},
		"post": func(url string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error) {
			args := map[string]interface{}{"url": url, "body": body, "opts": opts}
			if r.mockMode {
				resp := map[string]interface{}{"status": 200, "body": "{}"}
				r.record("http", "post", args, resp, nil)
				return resp, nil
			}
			resp, err := client.Post(ctx, url, body, opts)
			r.record("http", "post", args, resp, err)
			return resp, err
		},
	}
}

func (r *APIRecorder) CreateAIAPI(client *AIClient, ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"generate": func(prompt string) (string, error) {
			args := map[string]interface{}{"prompt": prompt[:min(100, len(prompt))] + "..."}
			if r.mockMode {
				resp := "This is a mock AI response for testing purposes."
				r.record("ai", "generate", args, resp, nil)
				return resp, nil
			}
			resp, err := client.Generate(ctx, prompt)
			r.record("ai", "generate", args, resp, err)
			return resp, err
		},
		"summarize": func(url string) (string, error) {
			args := map[string]interface{}{"url": url}
			if r.mockMode {
				resp := "This is a mock summary. The article discusses important topics relevant to the linked content."
				r.record("ai", "summarize", args, resp, nil)
				return resp, nil
			}
			resp, err := client.Summarize(ctx, url)
			r.record("ai", "summarize", args, resp, err)
			return resp, err
		},
	}
}

// ConsoleCapture captures console log calls.
type ConsoleCapture struct {
	logs *[]ConsoleLog
	mu   sync.Mutex
}

func (c *ConsoleCapture) log(level string, args ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	strArgs := make([]string, len(args))
	for i, arg := range args {
		strArgs[i] = fmt.Sprintf("%v", arg)
	}
	*c.logs = append(*c.logs, ConsoleLog{
		Level: level,
		Args:  strArgs,
		Time:  time.Now(),
	})
}

func (c *ConsoleCapture) CreateAPI() map[string]interface{} {
	return map[string]interface{}{
		"log":   func(args ...interface{}) { c.log("log", args...) },
		"info":  func(args ...interface{}) { c.log("info", args...) },
		"warn":  func(args ...interface{}) { c.log("warn", args...) },
		"error": func(args ...interface{}) { c.log("error", args...) },
		"debug": func(args ...interface{}) { c.log("debug", args...) },
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
