package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/cmd"
	"github.com/bcneng/candebot/inclusion"
	"github.com/bcneng/candebot/slackx"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack/slackevents"
)

func eventsAPIHandler(botCtx cmd.BotContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		if err := botCtx.VerifyRequest(r, buf.Bytes()); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Fail to verify SigningSecret: %v", err)
		}

		body := buf.String()
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			if err := json.Unmarshal([]byte(body), &r); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			_, _ = w.Write([]byte(r.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			switch event := innerEvent.Data.(type) {
			case *slackevents.MessageEvent:
				if event.BotID == candebotBotID {
					// Skip own (bot) command replies
					return
				}

				if event.SubType == "" || event.SubType == "message_replied" {
					// behaviors that apply to all messages posted by users both in channels or threads
					go checkLanguage(botCtx, event)
				}

				if event.ChannelType == "im" {
					log.Println("Direct message:", event.Text)
					botCommand(botCtx, cmd.SlackContext{
						User:            event.User,
						Channel:         event.Channel,
						Text:            event.Text,
						Timestamp:       event.TimeStamp,
						ThreadTimestamp: event.ThreadTimeStamp,
					})

					return
				}

				switch event.Channel {
				case channelHiringJobBoard:
					// Staff members are allowed to post messages
					if botCtx.IsStaff(event.User) {
						return
					}

					// Users are allowed to only post messages in threads
					if event.ThreadTimeStamp == "" {
						log.Println("Someone wrote a random message in #hiring-job-board and will be removed.", event.Channel, event.Text, event.TimeStamp)
						_, _, _ = botCtx.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)
						return
					}
				case channelCandebotTesting:
					// playground
				}
			case *slackevents.AppMentionEvent:
				log.Println("Mention message:", event.Text)
				botCommand(botCtx, cmd.SlackContext{
					User:            event.User,
					Channel:         event.Channel,
					Text:            event.Text,
					Timestamp:       event.TimeStamp,
					ThreadTimestamp: event.ThreadTimeStamp,
				})
			}
		}
	}
}

func botCommand(botCtx cmd.BotContext, slackCtx cmd.SlackContext) {
	text := strings.TrimSpace(strings.TrimPrefix(slackCtx.Text, fmt.Sprintf("<@%s>", candebotUser)))
	args := strings.Split(text, " ") // TODO strings.Split is not valid for quoted strings that contain spaces (E.g. echo command)
	w := bytes.NewBuffer([]byte{})
	defer func() {
		if w.Len() > 0 {
			_ = slackx.SendEphemeral(botCtx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, w.String())
		}
	}()

	_, kongCLI, err := cmd.NewCLI(args, kong.Writers(w, w))
	if err != nil {
		return
	}

	log.Println("Running command:", text)
	if err := kongCLI.Run(botCtx, slackCtx); err != nil {
		_ = slackx.SendEphemeral(botCtx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, err.Error())
		return
	}
}

func checkLanguage(botCtx cmd.BotContext, event *slackevents.MessageEvent) {
	filter := inclusion.Filter(event.Text)
	if filter == nil {
		return
	}

	// Send reply as Slack ephemeral message
	_ = slackx.SendEphemeral(botCtx.Client, event.ThreadTimeStamp, event.Channel, event.User, filter.Reply)

	// Sending metrics
	botCtx.Harvester.RecordMetric(telemetry.Count{
		Name: "candebot.inclusion.message_filtered",
		Attributes: map[string]interface{}{
			"channel":          event.Channel,
			"filter":           filter.Filter,
			"candebot_version": botCtx.Version,
		},
		Value:     1,
		Timestamp: time.Now(),
	})
}
