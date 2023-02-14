package handlers

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/bot"
	"github.com/bcneng/candebot/cmd"
	"github.com/bcneng/candebot/inclusion"
	"github.com/bcneng/candebot/slackx"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack/slackevents"
	"log"
	"strings"
	"time"
)

func MessageEventHandler(botCtx bot.Context, e slackevents.EventsAPIInnerEvent) error {
	event := e.Data.(*slackevents.MessageEvent)
	if event.BotID == botCtx.Config.Bot.ID {
		// Skip own (bot) command replies
		return nil
	}

	if event.SubType == "" || event.SubType == "message_replied" {
		// behaviors that apply to all messages posted by users both in channels or threads
		go checkLanguage(botCtx, event)
	}

	if event.ChannelType == "im" {
		log.Println("Direct message:", event.Text)
		botCommand(botCtx, bot.SlackContext{
			User:            event.User,
			Channel:         event.Channel,
			Text:            event.Text,
			Timestamp:       event.TimeStamp,
			ThreadTimestamp: event.ThreadTimeStamp,
		})

		return nil
	}

	switch event.Channel {
	case botCtx.Config.Channels.Jobs:
		// Staff members are allowed to post messages
		if botCtx.IsStaff(event.User) {
			return nil
		}

		// Users are allowed to only post messages in threads
		if event.ThreadTimeStamp == "" {
			log.Printf("Someone wrote a random message in %s and will be removed. %s %s", event.Channel, event.Text, event.TimeStamp)
			_, _, _ = botCtx.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)
			return nil
		}
	case botCtx.Config.Channels.Playground:
		// playground
	}

	return nil
}

func botCommand(botCtx bot.Context, slackCtx bot.SlackContext) {
	text := strings.TrimSpace(strings.TrimPrefix(slackCtx.Text, fmt.Sprintf("<@%s>", botCtx.Config.Bot.UserID)))
	args := strings.Split(text, " ") // TODO strings.Split is not valid for quoted strings that contain spaces (E.g. echo command)
	w := bytes.NewBuffer([]byte{})
	defer func() {
		if w.Len() > 0 {
			_ = slackx.SendEphemeral(botCtx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, w.String())
		}
	}()

	_, kongCLI, err := cmd.NewCLI(botCtx.Config.Bot.Name, args, kong.Writers(w, w))
	if err != nil {
		return
	}

	log.Println("Running command:", text)
	if err := kongCLI.Run(botCtx, slackCtx); err != nil {
		_ = slackx.SendEphemeral(botCtx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, err.Error())
		return
	}
}

func checkLanguage(botCtx bot.Context, event *slackevents.MessageEvent) {
	filter := inclusion.Filter(event.Text)
	if filter == nil {
		return
	}

	// Send reply as Slack ephemeral message
	_ = slackx.SendEphemeral(botCtx.Client, event.ThreadTimeStamp, event.Channel, event.User, filter.Reply)

	// Sending metrics
	botCtx.Harvester.RecordMetric(telemetry.Count{
		Name: fmt.Sprintf("%s.%s", strings.ToLower(botCtx.Config.Bot.Name), "inclusion.message_filtered"),
		Attributes: map[string]interface{}{
			"channel": event.Channel,
			"filter":  filter.Filter,
		},
		Value:     1,
		Timestamp: time.Now(),
	})
}
