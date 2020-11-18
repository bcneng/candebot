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
				// TODO consider removing this
				if len(event.User) == 0 || len(event.BotID) > 0 {
					break
				}

				// TODO consider removing this
				if event.SubType != "" || event.ThreadTimeStamp != "" {
					// We only want messages posted by humans. We also skip join/leave channel messages, etc by doing this.
					// Thread messages are also skipped.
					break
				}

				// behaviors that apply to all channels
				go checkLanguage(botContext.Client, event)

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
					// This regex check ensures that the message contains all the required fields. Even though posted with
					// a workflow, people can still introduce wrong values.
					if !isValidJobOffer(event.Text) {
						link, err := botContext.Client.GetPermalink(&slack.PermalinkParameters{
							Channel: event.Channel,
							Ts:      event.TimeStamp,
						})
						if err != nil {
							log.Printf("error fetching permalink for channel %s and ts %s\n", channelHiringJobBoardWrongFormatNotification, event.TimeStamp)
						}

						_ = slackx.Send(botContext.Client, "", channelHiringJobBoardWrongFormatNotification, fmt.Sprintf("new Job post with invalid format: %s", link), true)
					}

					if !botContext.IsStaff(event.User) {
						// If the message was not posted by any member of the Staff, delete it. Messages posted by workflows are not affected.
						_, _, err := botContext.AdminClient.DeleteMessage(event.Channel, event.TimeStamp)
						if err != nil {
							log.Printf("error deleting message: Error: %s\n", err.Error())
						}

						log.Printf("message has been removed: %s: %s\n", event.Message.User, event.Message.Text)
						_ = slackx.SendEphemeral(botContext.Client, event.ThreadTimeStamp, event.Channel, event.Message.User, "Regular posting on this channel is not allowed. Submit an offer by using the `New Job Post` workflow (:zap: button)")
					}
				case channelCandebotTesting:
					// Playground here
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
