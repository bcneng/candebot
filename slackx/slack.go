package slackx

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/slack-go/slack"
)

// ChannelResolver resolves channel names to IDs using channels.json and Slack API fallback.
type ChannelResolver struct {
	httpClient  *http.Client
	slackClient *slack.Client
	jsonURL     string
	cache       map[string]string
	mu          sync.RWMutex
}

// NewChannelResolver creates a new channel resolver.
func NewChannelResolver(httpClient *http.Client, slackClient *slack.Client) *ChannelResolver {
	return &ChannelResolver{
		httpClient:  httpClient,
		slackClient: slackClient,
		jsonURL:     "https://raw.githubusercontent.com/bcneng/website/refs/heads/main/data/channels.json",
		cache:       make(map[string]string),
	}
}

func SendEphemeral(c *slack.Client, threadTS, channelID, userID, msg string, opts ...slack.MsgOption) error {
	if userID == "" || channelID == "" {
		return nil
	}

	_, err := c.PostEphemeral(channelID, userID, append(opts, slack.MsgOptionText(msg, false), slack.MsgOptionTS(threadTS))...)
	if err != nil {
		log.Printf("error sending ephemeral msg in channel %q: %s\n", channelID, err.Error())
	}

	return err
}

func Send(c *slack.Client, threadTS, channelID, msg string, scape bool, opts ...slack.MsgOption) error {
	if channelID == "" {
		return nil
	}
	_, _, err := c.PostMessage(channelID, append(opts, slack.MsgOptionText(msg, scape), slack.MsgOptionTS(threadTS))...)
	if err != nil {
		log.Println("error sending msg in channel ", channelID)
	}

	return err
}

var publicPrivate = []string{"public_channel", "private_channel"}

type channelData struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// FindChannelIDByName resolves a channel name to its ID.
// It first checks the cache, then channels.json, and finally falls back to the Slack API.
func (r *ChannelResolver) FindChannelIDByName(channel string) (string, error) {
	// Check cache with read lock
	r.mu.RLock()
	id, ok := r.cache[channel]
	r.mu.RUnlock()
	if ok {
		return id, nil
	}

	// Try fetching from channels.json first to avoid Slack API rate limits
	if id, found := r.findInChannelsJSON(channel); found {
		return id, nil
	}

	// Fallback to Slack API if not found in channels.json
	return r.findViaSlackAPI(channel)
}

func (r *ChannelResolver) findInChannelsJSON(channel string) (string, bool) {
	resp, err := r.httpClient.Get(r.jsonURL)
	if err != nil {
		return "", false
	}
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}

	var channelsData []channelData
	if err := json.Unmarshal(body, &channelsData); err != nil {
		return "", false
	}

	for _, ch := range channelsData {
		if ch.Name != channel {
			continue
		}

		r.mu.Lock()
		r.cache[channel] = ch.ID
		r.mu.Unlock()
		return ch.ID, true
	}

	return "", false
}

func (r *ChannelResolver) findViaSlackAPI(channel string) (string, error) {
	var cursor string
	for {
		channels, nextCursor, err := r.slackClient.GetConversations(&slack.GetConversationsParameters{
			Cursor:          cursor,
			ExcludeArchived: true,
			Types:           publicPrivate,
		})
		if err != nil {
			return "", err
		}

		for _, ch := range channels {
			if ch.Name != channel {
				continue
			}

			r.mu.Lock()
			r.cache[channel] = ch.ID
			r.mu.Unlock()
			return ch.ID, nil
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return "", fmt.Errorf("channel %s not found", channel)
}

func LinkToMessage(channelID, msgTimestamp string) string {
	return fmt.Sprintf("https://bcneng.slack.com/archives/%s/p%s", channelID, strings.Replace(msgTimestamp, ".", "", 1))
}
