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
					if botContext.IsStaff(event.User) {
						// If the message was written by the Staff, nothing to check here!
						return
					}

					if event.SubType == "" {
						// Message (not in threads) published by a user should be removed
						log.Println("Someone wrote a random message in #hiring-job-board and will be removed.", event.Channel, event.Text, event.TimeStamp)
						_, _, _ = botContext.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)
						return
					}

					if event.SubType == "bot_message" && !isValidJobOffer(event.Text) {
						// At this point, messages are written by the workflow bot.
						log.Println("Invalid job spec format. message will be removed.", event.Channel, event.Text, event.TimeStamp)
						_, _, _ = botContext.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)

						d := strings.Split(event.Text, "More info DM ")
						if len(d) != 2 {
							log.Printf("Impossible to get the sender from workflow message: %s\n", event.Text)
							return
						}

						sender := strings.Replace(strings.Replace(d[1], "<@", "", 1), ">", "", 1)
						_ = slackx.SendEphemeral(botContext.Client, event.ThreadTimeStamp, event.Channel, sender, fmt.Sprintf("The Job post you've submitted seems invalid. Please review your message:\n%s", event.Text))
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

func isValidJobOffer(text string) bool {
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		if !jobOfferRegex.MatchString(l) {
			return false
		}
	}

	return true
}

func checkLanguage(botClient *slack.Client, event *slackevents.MessageEvent) {
	if reply := inclusion.Filter(event.Text); reply != "" {
		_ = slackx.SendEphemeral(botClient, event.ThreadTimeStamp, event.Channel, event.User, reply)
	}
}
