package jsruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// AIClient provides AI functionality to JS handlers using Google Gemini.
type AIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// GeminiRequest represents a request to the Gemini API.
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContent represents content in a Gemini request/response.
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part of content (text, image, etc).
type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

// GeminiGenerationConfig configures generation parameters.
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiResponse represents a response from the Gemini API.
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []GeminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// NewAIClient creates a new AI client for JS handlers.
func NewAIClient(apiKey string, timeout time.Duration) *AIClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &AIClient{
		apiKey: apiKey,
		model:  "gemini-2.0-flash-lite", // Fast, cost-effective model
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// IsConfigured returns true if the AI client has an API key configured.
func (c *AIClient) IsConfigured() bool {
	return c.apiKey != ""
}

// Generate generates text from a prompt using Gemini.
func (c *AIClient) Generate(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("AI API key not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 500, // Keep summaries concise
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	if len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// Summarize fetches a URL and returns an AI-generated summary.
func (c *AIClient) Summarize(ctx context.Context, url string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("AI API key not configured")
	}

	// Fetch the URL content
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Candebot/1.0 (Link Summarizer)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body with limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Extract text from HTML
	content := extractTextFromHTML(string(body))
	if len(content) < 100 {
		return "", fmt.Errorf("content too short to summarize")
	}

	// Truncate if too long
	if len(content) > 12000 {
		content = content[:12000]
	}

	// Generate summary
	prompt := "Provide a brief TL;DR summary (2-3 sentences max) of the following web page content. " +
		"Focus on the main topic and key points. Be concise and informative. " +
		"Do not use markdown formatting.\n\nContent:\n" + content

	return c.Generate(ctx, prompt)
}

// extractTextFromHTML extracts readable text from HTML content.
func extractTextFromHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fallback: strip tags with regex
		return stripHTMLTags(htmlContent)
	}

	var textParts []string
	var extractText func(*html.Node)

	extractText = func(n *html.Node) {
		// Skip script, style, nav, footer, header elements
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "header", "aside", "noscript":
				return
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				textParts = append(textParts, text)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)

	// Join and clean up whitespace
	result := strings.Join(textParts, " ")
	// Collapse multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	result = spaceRegex.ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

// stripHTMLTags is a fallback for when HTML parsing fails.
func stripHTMLTags(s string) string {
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	result := tagRegex.ReplaceAllString(s, " ")
	spaceRegex := regexp.MustCompile(`\s+`)
	result = spaceRegex.ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}

// CreateAIAPI creates the AI API object to be exposed to JS handlers.
func CreateAIAPI(client *AIClient, ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"generate": func(prompt string) (string, error) {
			return client.Generate(ctx, prompt)
		},
		"summarize": func(url string) (string, error) {
			return client.Summarize(ctx, url)
		},
	}
}
