package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func eventsAPIHandler(botClient, adminClient *slack.Client) http.HandlerFunc {
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
				go checkLanguage(botClient, event)

				if event.ChannelType == "im" {
					// Direct Message

					// TODO implement commands
					log.Println("Direct message:", event.Text)
					return
				}

				switch event.Channel {
				case channelHiringJobBoard:
					// This regex check ensures that the message contains all the required fields. Even though posted with
					// a workflow, people can still introduce wrong values.
					if !isValidJobOffer(event.Text) {
						link, err := botClient.GetPermalink(&slack.PermalinkParameters{
							Channel: event.Channel,
							Ts:      event.TimeStamp,
						})
						if err != nil {
							log.Printf("error fetching permalink for channel %s and ts %s\n", channelHiringJobBoardWrongFormatNotification, event.TimeStamp)
						}

						_ = send(
							botClient,
							channelHiringJobBoardWrongFormatNotification,
							fmt.Sprintf("new Job post with invalid format: %s", link),
							true,
						)
					}

					if !isStaff(event.User) {
						// If the message was not posted by any member of the Staff, delete it. Messages posted by workflows are not affected.
						_, _, err := adminClient.DeleteMessage(event.Channel, event.TimeStamp)
						if err != nil {
							log.Printf("error deleting message: Error: %s\n", err.Error())
						}

						log.Printf("message has been removed: %s: %s\n", event.Message.User, event.Message.Text)
						_ = sendEphemeral(botClient, event.Channel, event.Message.User, "Regular posting on this channel is not allowed. Submit an offer by using the `New Job Post` workflow (:zap: button)")
					}
				case channelCandebotTesting:
					// Playground here
				}
			case *slackevents.AppMentionEvent:
				// TODO
				text := strings.TrimPrefix(event.Text, fmt.Sprintf("<@%s> ", candebotUser))
				log.Println("Mentioned or DM. Text", text)
			}
		}
	}
}
