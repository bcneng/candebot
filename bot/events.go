package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/cmd"
	"github.com/bcneng/candebot/inclusion"
	"github.com/bcneng/candebot/slackx"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func eventsAPIHandler(botContext cmd.BotContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		if err := botContext.VerifyRequest(r, buf.Bytes()); err != nil {
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
					go checkLanguage(botContext.Client, event)
				}

				if event.ChannelType == "im" {
					log.Println("Direct message:", event.Text)
					botCommand(botContext, cmd.SlackContext{
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
					// Staff memebers are allowed to post messages
					if botContext.IsStaff(event.User) {
						return
					}

					// Users are allowed to only post messages in threads
					if event.ThreadTimeStamp == "" {
						log.Println("Someone wrote a random message in #hiring-job-board and will be removed.", event.Channel, event.Text, event.TimeStamp)
						_, _, _ = botContext.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)
						return
					}
				case channelCandebotTesting:
					// playground
				}
			case *slackevents.AppMentionEvent:
				log.Println("Mention message:", event.Text)
				botCommand(botContext, cmd.SlackContext{
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

func checkLanguage(botClient *slack.Client, event *slackevents.MessageEvent) {
	if reply := inclusion.Filter(event.Text); reply != "" {
		_ = slackx.SendEphemeral(botClient, event.ThreadTimeStamp, event.Channel, event.User, reply)
	}
}
