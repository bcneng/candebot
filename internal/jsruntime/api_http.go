package jsruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient provides HTTP functionality to JS handlers.
type HTTPClient struct {
	client       *http.Client
	allowedHosts []string
	blockedHosts []string
	timeout      time.Duration
}

// NewHTTPClient creates a new HTTP client for JS handlers.
func NewHTTPClient(config RuntimeConfig) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: time.Duration(config.DefaultTimeout) * time.Millisecond,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		allowedHosts: config.AllowedHosts,
		blockedHosts: config.BlockedHosts,
		timeout:      time.Duration(config.DefaultTimeout) * time.Millisecond,
	}
}

// isHostAllowed checks if a host is allowed for HTTP requests.
func (c *HTTPClient) isHostAllowed(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(parsed.Hostname())

	// Check blocked hosts first
	for _, blocked := range c.blockedHosts {
		if strings.ToLower(blocked) == host {
			return fmt.Errorf("host %q is blocked", host)
		}
	}

	// If allowed hosts is empty, allow all (except blocked)
	if len(c.allowedHosts) == 0 {
		return nil
	}

	// Check if host is in allowed list
	for _, allowed := range c.allowedHosts {
		if strings.ToLower(allowed) == host {
			return nil
		}
	}

	return fmt.Errorf("host %q is not in allowed list", host)
}

// Get performs an HTTP GET request.
func (c *HTTPClient) Get(ctx context.Context, urlStr string, opts map[string]interface{}) (map[string]interface{}, error) {
	return c.Fetch(ctx, urlStr, mergeOpts(opts, map[string]interface{}{"method": "GET"}))
}

// Post performs an HTTP POST request.
func (c *HTTPClient) Post(ctx context.Context, urlStr string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error) {
	merged := mergeOpts(opts, map[string]interface{}{"method": "POST", "body": body})
	return c.Fetch(ctx, urlStr, merged)
}

// Put performs an HTTP PUT request.
func (c *HTTPClient) Put(ctx context.Context, urlStr string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error) {
	merged := mergeOpts(opts, map[string]interface{}{"method": "PUT", "body": body})
	return c.Fetch(ctx, urlStr, merged)
}

// Delete performs an HTTP DELETE request.
func (c *HTTPClient) Delete(ctx context.Context, urlStr string, opts map[string]interface{}) (map[string]interface{}, error) {
	return c.Fetch(ctx, urlStr, mergeOpts(opts, map[string]interface{}{"method": "DELETE"}))
}

// Fetch performs a generic HTTP request (similar to JS fetch API).
func (c *HTTPClient) Fetch(ctx context.Context, urlStr string, opts map[string]interface{}) (map[string]interface{}, error) {
	if err := c.isHostAllowed(urlStr); err != nil {
		return nil, err
	}

	method := "GET"
	if m, ok := opts["method"].(string); ok {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	if body, ok := opts["body"]; ok && body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			// Assume JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBytes)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("User-Agent", "Candebot-Handler/1.0")

	// Set custom headers
	if headers, ok := opts["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if vs, ok := v.(string); ok {
				req.Header.Set(k, vs)
			}
		}
	}

	// Set Content-Type for POST/PUT if not set and body exists
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		if _, ok := opts["body"].(string); ok {
			req.Header.Set("Content-Type", "text/plain")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (limit to 10MB)
	limitReader := io.LimitReader(resp.Body, 10*1024*1024)
	respBody, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := map[string]interface{}{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
		"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
		"headers":    headerToMap(resp.Header),
		"body":       string(respBody),
	}

	// Try to parse JSON body
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonBody interface{}
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			result["json"] = jsonBody
		}
	}

	return result, nil
}

func headerToMap(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func mergeOpts(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// CreateHTTPAPI creates the HTTP API object to be exposed to JS handlers.
func CreateHTTPAPI(client *HTTPClient, ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"get": func(url string, opts map[string]interface{}) (map[string]interface{}, error) {
			return client.Get(ctx, url, opts)
		},
		"post": func(url string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error) {
			return client.Post(ctx, url, body, opts)
		},
		"put": func(url string, body interface{}, opts map[string]interface{}) (map[string]interface{}, error) {
			return client.Put(ctx, url, body, opts)
		},
		"delete": func(url string, opts map[string]interface{}) (map[string]interface{}, error) {
			return client.Delete(ctx, url, opts)
		},
		"fetch": func(url string, opts map[string]interface{}) (map[string]interface{}, error) {
			return client.Fetch(ctx, url, opts)
		},
	}
}
