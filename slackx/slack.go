package slackx

import (
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
)

var channelNameToIDCache map[string]string

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

func FindChannelIDByName(client *slack.Client, channel string) (string, error) {
	if channelNameToIDCache == nil {
		channelNameToIDCache = make(map[string]string)
	}

	id, ok := channelNameToIDCache[channel]
	if ok {
		return id, nil
	}

	var cursor string
	var channels []slack.Channel
	for {
		var err error
		channels, cursor, err = client.GetConversations(&slack.GetConversationsParameters{Cursor: cursor, ExcludeArchived: true})
		if err != nil {
			return "", err
		}

		if cursor == "" {
			break
		}
	}

	for _, c := range channels {
		if c.Name == channel {
			channelNameToIDCache[channel] = c.ID // It is fine to not lock.

			return c.ID, nil
		}
	}

	return "", fmt.Errorf("channel %s not found", channel)
}

func LinkToMessage(channelID, msgTimestamp string) string {
	return fmt.Sprintf("https://bcneng.slack.com/archives/%s/p%s", channelID, strings.Replace(msgTimestamp, ".", "", 1))
}
