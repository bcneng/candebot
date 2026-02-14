package suggest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/bcneng/candebot/slackx"
	"github.com/slack-go/slack"
)

const channelsJSONURL = "https://raw.githubusercontent.com/bcneng/website/refs/heads/main/data/channels.json"

// Channel represents a community Slack channel from channels.json.
type Channel struct {
	Name        string   `json:"name"`
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// ChannelSuggester fetches channels from the BcnEng website and posts
// weekly suggestions to a target channel.
type ChannelSuggester struct {
	httpClient  *http.Client
	slackClient *slack.Client
	channelsURL string
}

// NewChannelSuggester creates a new ChannelSuggester.
func NewChannelSuggester(httpClient *http.Client, slackClient *slack.Client) *ChannelSuggester {
	return &ChannelSuggester{
		httpClient:  httpClient,
		slackClient: slackClient,
		channelsURL: channelsJSONURL,
	}
}

// FetchChannels fetches the full list of channels from channels.json.
func (s *ChannelSuggester) FetchChannels() ([]Channel, error) {
	resp, err := s.httpClient.Get(s.channelsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching channels.json: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching channels.json: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading channels.json body: %w", err)
	}

	var channels []Channel
	if err := json.Unmarshal(body, &channels); err != nil {
		return nil, fmt.Errorf("parsing channels.json: %w", err)
	}

	return channels, nil
}

// FilterSuggestable filters out channels that should not be suggested,
// such as core channels that everyone already knows about.
func FilterSuggestable(channels []Channel) []Channel {
	var result []Channel
	for _, ch := range channels {
		if ch.Category == "core" {
			continue
		}
		if hasTag(ch.Tags, "default") {
			continue
		}
		result = append(result, ch)
	}
	return result
}

// SelectRandom picks n random channels from the given list.
// If n >= len(channels), all channels are returned (shuffled).
func SelectRandom(channels []Channel, n int) []Channel {
	if len(channels) == 0 {
		return nil
	}

	shuffled := make([]Channel, len(channels))
	copy(shuffled, channels)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	if n >= len(shuffled) {
		return shuffled
	}
	return shuffled[:n]
}

// FormatMessage creates a Slack message with channel suggestions.
func FormatMessage(channels []Channel) string {
	if len(channels) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(":wave: *Weekly Channel Suggestions*\n\n")
	sb.WriteString("Here are some channels you might want to check out:\n\n")

	for _, ch := range channels {
		sb.WriteString(fmt.Sprintf("• <#%s> — %s\n", ch.ID, ch.Description))
	}

	sb.WriteString("\nJoin the conversation! :speech_balloon:")
	return sb.String()
}

// PostSuggestion fetches channels, picks random ones, and posts the suggestion
// to the specified target channel.
func (s *ChannelSuggester) PostSuggestion(targetChannelID string, numChannels int) error {
	channels, err := s.FetchChannels()
	if err != nil {
		return err
	}

	suggestable := FilterSuggestable(channels)
	selected := SelectRandom(suggestable, numChannels)
	if len(selected) == 0 {
		return fmt.Errorf("no suggestable channels found")
	}

	msg := FormatMessage(selected)
	return slackx.Send(s.slackClient, "", targetChannelID, msg, false)
}

// StartWeekly starts a goroutine that posts channel suggestions weekly.
// It blocks until the context is cancelled.
func (s *ChannelSuggester) StartWeekly(ctx context.Context, targetChannelID string, numChannels int) {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	log.Printf("[INFO] Channel suggester started, will post to %s every 7 days", targetChannelID)

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] Channel suggester stopped")
			return
		case <-ticker.C:
			if err := s.PostSuggestion(targetChannelID, numChannels); err != nil {
				log.Printf("[ERROR] Channel suggester failed to post: %v", err)
			} else {
				log.Println("[INFO] Channel suggester posted weekly suggestion")
			}
		}
	}
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
