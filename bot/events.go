package bot

import (
	"bytes"
	"encoding/json"
	"github.com/slack-go/slack/slackevents"
	"log"
	"net/http"
)

// EventHandler handles a Slack event
type EventHandler func(Context, slackevents.EventsAPIInnerEvent) error

// CreateEventHandler creates a handler aware of errors (they are logged)
func CreateEventHandler(t slackevents.EventsAPIType, f EventHandler) EventHandler {
	return func(c Context, e slackevents.EventsAPIInnerEvent) error {
		if slackevents.EventsAPIType(e.Type) != t {
			log.Printf("unexpected event type. Should be %q, but was %q", t, e.Type)
		}

		if e.Data == nil {
			log.Printf("event %q data is nil", t)
		}

		if err := f(c, e); err != nil {
			log.Println(err)
		}

		return nil
	}
}

func eventsAPIHandler(botCtx Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		if err := botCtx.VerifyRequest(r, buf.Bytes()); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Fail to verify SigningSecret: %v", err)
		}

		eventsAPIEvent, err := slackevents.ParseEvent(buf.Bytes(), slackevents.OptionNoVerifyToken())
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			if err := json.Unmarshal(buf.Bytes(), &r); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			_, _ = w.Write([]byte(r.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			botCtx.Bus.Publish(eventsAPIEvent.InnerEvent.Type, botCtx, eventsAPIEvent.InnerEvent)
		}
	}
}
