package jsruntime

import (
	"fmt"

	"github.com/slack-go/slack"
)

// SlackClient provides Slack functionality to JS handlers.
type SlackClient struct {
	client      *slack.Client
	adminClient *slack.Client
}

// NewSlackClient creates a new Slack client for JS handlers.
func NewSlackClient(client, adminClient *slack.Client) *SlackClient {
	return &SlackClient{
		client:      client,
		adminClient: adminClient,
	}
}

// SendMessage sends a message to a channel.
func (s *SlackClient) SendMessage(channel, text string, opts map[string]interface{}) (map[string]interface{}, error) {
	if channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	msgOpts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
	}

	// Handle thread timestamp
	if threadTS, ok := opts["threadTimestamp"].(string); ok && threadTS != "" {
		msgOpts = append(msgOpts, slack.MsgOptionTS(threadTS))
	}

	// Handle broadcast (reply_broadcast)
	if broadcast, ok := opts["broadcast"].(bool); ok && broadcast {
		msgOpts = append(msgOpts, slack.MsgOptionBroadcast())
	}

	// Handle unfurl options
	if unfurlLinks, ok := opts["unfurlLinks"].(bool); ok {
		msgOpts = append(msgOpts, slack.MsgOptionDisableLinkUnfurl())
		if !unfurlLinks {
			msgOpts = append(msgOpts, slack.MsgOptionDisableLinkUnfurl())
		}
	}

	channelID, timestamp, err := s.client.PostMessage(channel, msgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return map[string]interface{}{
		"channel":   channelID,
		"timestamp": timestamp,
	}, nil
}

// SendEphemeral sends an ephemeral message to a user in a channel.
func (s *SlackClient) SendEphemeral(channel, user, text string, opts map[string]interface{}) error {
	if channel == "" || user == "" {
		return fmt.Errorf("channel and user are required")
	}

	msgOpts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
	}

	// Handle thread timestamp
	if opts != nil {
		if threadTS, ok := opts["threadTimestamp"].(string); ok && threadTS != "" {
			msgOpts = append(msgOpts, slack.MsgOptionTS(threadTS))
		}
	}

	_, err := s.client.PostEphemeral(channel, user, msgOpts...)
	if err != nil {
		return fmt.Errorf("failed to send ephemeral: %w", err)
	}

	return nil
}

// AddReaction adds a reaction to a message.
func (s *SlackClient) AddReaction(channel, timestamp, emoji string) error {
	if channel == "" || timestamp == "" || emoji == "" {
		return fmt.Errorf("channel, timestamp, and emoji are required")
	}

	err := s.client.AddReaction(emoji, slack.ItemRef{
		Channel:   channel,
		Timestamp: timestamp,
	})
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	return nil
}

// RemoveReaction removes a reaction from a message.
func (s *SlackClient) RemoveReaction(channel, timestamp, emoji string) error {
	if channel == "" || timestamp == "" || emoji == "" {
		return fmt.Errorf("channel, timestamp, and emoji are required")
	}

	err := s.client.RemoveReaction(emoji, slack.ItemRef{
		Channel:   channel,
		Timestamp: timestamp,
	})
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}

	return nil
}

// GetUserInfo gets information about a user.
func (s *SlackClient) GetUserInfo(userID string) (map[string]interface{}, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	user, err := s.client.GetUserInfo(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return map[string]interface{}{
		"id":          user.ID,
		"name":        user.Name,
		"realName":    user.RealName,
		"displayName": user.Profile.DisplayName,
		"email":       user.Profile.Email,
		"isBot":       user.IsBot,
		"isAdmin":     user.IsAdmin,
		"isOwner":     user.IsOwner,
		"timezone":    user.TZ,
		"avatar":      user.Profile.Image192,
	}, nil
}

// GetChannelInfo gets information about a channel.
func (s *SlackClient) GetChannelInfo(channelID string) (map[string]interface{}, error) {
	if channelID == "" {
		return nil, fmt.Errorf("channelID is required")
	}

	channel, err := s.client.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID: channelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %w", err)
	}

	return map[string]interface{}{
		"id":         channel.ID,
		"name":       channel.Name,
		"topic":      channel.Topic.Value,
		"purpose":    channel.Purpose.Value,
		"isPrivate":  channel.IsPrivate,
		"isArchived": channel.IsArchived,
		"memberCount": channel.NumMembers,
	}, nil
}

// DeleteMessage deletes a message (requires admin token).
func (s *SlackClient) DeleteMessage(channel, timestamp string) error {
	if channel == "" || timestamp == "" {
		return fmt.Errorf("channel and timestamp are required")
	}

	if s.adminClient == nil {
		return fmt.Errorf("admin client not available")
	}

	_, _, err := s.adminClient.DeleteMessage(channel, timestamp)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// UpdateMessage updates an existing message.
func (s *SlackClient) UpdateMessage(channel, timestamp, text string) error {
	if channel == "" || timestamp == "" {
		return fmt.Errorf("channel and timestamp are required")
	}

	_, _, _, err := s.client.UpdateMessage(channel, timestamp, slack.MsgOptionText(text, false))
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// CreateSlackAPI creates the Slack API object to be exposed to JS handlers.
func CreateSlackAPI(client *SlackClient) map[string]interface{} {
	return map[string]interface{}{
		"sendMessage": func(channel, text string, opts map[string]interface{}) (map[string]interface{}, error) {
			return client.SendMessage(channel, text, opts)
		},
		"sendEphemeral": func(channel, user, text string, opts map[string]interface{}) error {
			return client.SendEphemeral(channel, user, text, opts)
		},
		"addReaction": func(channel, timestamp, emoji string) error {
			return client.AddReaction(channel, timestamp, emoji)
		},
		"removeReaction": func(channel, timestamp, emoji string) error {
			return client.RemoveReaction(channel, timestamp, emoji)
		},
		"getUserInfo": func(userID string) (map[string]interface{}, error) {
			return client.GetUserInfo(userID)
		},
		"getChannelInfo": func(channelID string) (map[string]interface{}, error) {
			return client.GetChannelInfo(channelID)
		},
		"deleteMessage": func(channel, timestamp string) error {
			return client.DeleteMessage(channel, timestamp)
		},
		"updateMessage": func(channel, timestamp, text string) error {
			return client.UpdateMessage(channel, timestamp, text)
		},
	}
}
