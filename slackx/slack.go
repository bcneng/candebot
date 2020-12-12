package slackx

import (
	"fmt"
	"log"

	"github.com/slack-go/slack"
)

var channelNameToIDCache map[string]string

func SendEphemeral(c *slack.Client, threadTS, channelID, userID, msg string, opts ...slack.MsgOption) error {
	if userID == "" || channelID == "" {
		return nil
	}

	_, err := c.PostEphemeral(channelID, userID, append(opts, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true), slack.MsgOptionTS(threadTS))...)
	if err != nil {
		log.Printf("error sending ephemeral msg in channel %q: %s\n", channelID, err.Error())
	}

	return err
}

func Send(c *slack.Client, threadTS, channelID, msg string, scape bool, opts ...slack.MsgOption) error {
	if channelID == "" {
		return nil
	}
	_, _, err := c.PostMessage(channelID, append(opts, slack.MsgOptionText(msg, scape), slack.MsgOptionAsUser(true), slack.MsgOptionTS(threadTS))...)
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

	chans, err := client.GetChannels(true, slack.GetChannelsOptionExcludeMembers())
	if err != nil {
		return "", err
	}

	for _, c := range chans {
		if c.Name == channel {
			return c.ID, nil
		}
	}

	privateChans, err := client.GetGroups(true)
	if err != nil {
		return "", err
	}

	for _, c := range privateChans {
		if c.Name == channel {
			channelNameToIDCache[channel] = c.ID // It is fine to not lock.

			return c.ID, nil
		}
	}

	return "", fmt.Errorf("channel %s not found", channel)
}
