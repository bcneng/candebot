package slackx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
)

func TestFindChannelIDByName_FromChannelsJSON(t *testing.T) {
	channelsJSON := `[
		{"name": "general", "id": "C123456"},
		{"name": "candebot-testing", "id": "C789012"},
		{"name": "random", "id": "C345678"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(channelsJSON))
	}))
	defer server.Close()

	resolver := NewChannelResolver(server.Client(), slack.New("test-token"))
	resolver.jsonURL = server.URL

	id, err := resolver.FindChannelIDByName("candebot-testing")
	require.NoError(t, err)
	require.Equal(t, "C789012", id)

	// Verify it's cached
	resolver.mu.RLock()
	cachedID, ok := resolver.cache["candebot-testing"]
	resolver.mu.RUnlock()
	require.True(t, ok, "expected channel to be cached")
	require.Equal(t, "C789012", cachedID)
}

func TestFindChannelIDByName_CacheHit(t *testing.T) {
	resolver := NewChannelResolver(&http.Client{}, slack.New("test-token"))

	// Pre-populate cache
	resolver.cache["cached-channel"] = "C999999"

	id, err := resolver.FindChannelIDByName("cached-channel")
	require.NoError(t, err)
	require.Equal(t, "C999999", id)
}

func TestFindChannelIDByName_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resolver := NewChannelResolver(server.Client(), slack.New("test-token"))
	resolver.jsonURL = server.URL

	// Should fallback to Slack API (which will fail in this test, but that's expected)
	_, err := resolver.FindChannelIDByName("test-channel")
	require.Error(t, err, "expected error from Slack API fallback")
}

func TestFindChannelIDByName_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	resolver := NewChannelResolver(server.Client(), slack.New("test-token"))
	resolver.jsonURL = server.URL

	// Should fallback to Slack API (which will fail in this test, but that's expected)
	_, err := resolver.FindChannelIDByName("test-channel")
	require.Error(t, err, "expected error from Slack API fallback")
}

func TestFindChannelIDByName_ChannelNotInJSON(t *testing.T) {
	channelsJSON := `[
		{"name": "general", "id": "C123456"},
		{"name": "random", "id": "C345678"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(channelsJSON))
	}))
	defer server.Close()

	resolver := NewChannelResolver(server.Client(), slack.New("test-token"))
	resolver.jsonURL = server.URL

	// Should fallback to Slack API (which will fail in this test, but that's expected)
	_, err := resolver.FindChannelIDByName("missing-channel")
	require.Error(t, err, "expected error from Slack API fallback")
}
