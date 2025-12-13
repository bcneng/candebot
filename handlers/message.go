package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/bot"
	"github.com/bcneng/candebot/cmd"
	"github.com/bcneng/candebot/inclusion"
	"github.com/bcneng/candebot/internal/jsruntime"
	"github.com/bcneng/candebot/internal/privacy"
	"github.com/bcneng/candebot/slackx"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack/slackevents"
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
		go checkTracking(botCtx, event)
		// Execute JS handlers asynchronously
		go executeJSHandlers(botCtx, event)
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

	if botCtx.RateLimiter != nil && event.ThreadTimeStamp == "" {
		isStaff := botCtx.IsStaff(event.User)
		shouldCheckStaff := botCtx.RateLimiter.ShouldCheckStaff(event.Channel)

		if !isStaff || shouldCheckStaff {
			allowed, nextAllowedTime := botCtx.RateLimiter.CheckLimit(event.Channel, event.User)
			if !allowed {
				waitDuration := time.Until(nextAllowedTime)
				msg := fmt.Sprintf(
					"Your message has been deleted because you've reached the rate limit for this channel.\n\n"+
						"You can post again in approximately %s.",
					waitDuration.Round(time.Second),
				)
				_ = slackx.SendEphemeral(botCtx.Client, event.ThreadTimeStamp, event.Channel, event.User, msg)

				_, _, _ = botCtx.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)
				return nil
			}
		}
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

func checkTracking(botCtx bot.Context, event *slackevents.MessageEvent) {
	if botCtx.TrackingDetector == nil {
		return
	}

	if !botCtx.TrackingDetector.ShouldCheck(event.Channel) {
		return
	}

	tracked := privacy.FindTrackedURLs(event.Text)
	if len(tracked) == 0 {
		return
	}

	warning := privacy.FormatWarningMessage(tracked)
	_ = slackx.SendEphemeral(botCtx.Client, event.ThreadTimeStamp, event.Channel, event.User, warning)

	if botCtx.Harvester != nil {
		for _, t := range tracked {
			botCtx.Harvester.RecordMetric(telemetry.Count{
				Name: fmt.Sprintf("%s.%s", strings.ToLower(botCtx.Config.Bot.Name), "tracking.param_detected"),
				Attributes: map[string]interface{}{
					"channel":  event.Channel,
					"platform": t.Platform,
					"param":    t.Param,
				},
				Value:     1,
				Timestamp: time.Now(),
			})
		}
	}
}

// executeJSHandlers runs all matching JS handlers for the message event.
func executeJSHandlers(botCtx bot.Context, event *slackevents.MessageEvent) {
	if botCtx.JSRuntime == nil {
		return
	}

	// Resolve channel name if possible
	channelName := event.Channel
	if botCtx.ChannelResolver != nil {
		// For now, use channel ID as the name for matching
		// Handlers can use both channel ID and name patterns
		channelName = event.Channel
	}

	// Build message data for JS handlers
	message := jsruntime.MessageData{
		Type:            "message",
		Channel:         event.Channel,
		ChannelName:     channelName,
		ChannelType:     event.ChannelType,
		User:            event.User,
		Text:            event.Text,
		Timestamp:       event.TimeStamp,
		ThreadTimestamp: event.ThreadTimeStamp,
		IsThread:        event.ThreadTimeStamp != "",
		IsDM:            event.ChannelType == "im",
		BotID:           event.BotID,
		SubType:         event.SubType,
	}

	// Execute handlers with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := botCtx.JSRuntime.ExecuteHandlers(ctx, channelName, message)

	// Log results for debugging
	for i, result := range results {
		if result.Error != "" {
			log.Printf("[JS Handler %d] Error: %s", i, result.Error)
		}
	}
}
