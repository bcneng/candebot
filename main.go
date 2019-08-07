package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nlopes/slack"
	"log"
	"net/http"
	"regexp"

	"github.com/shomali11/slacker"

	"github.com/kelseyhightower/envconfig"
)

// Version is the candebot version. Usually the git commit hash. Passed during building.
var Version = "unknown"

const (
	msgCOC        = "Please find our Code Of Conduct here: https://bcneng.github.io/coc/"
	msgNetiquette = "Please find our Netiquette here: https://bcneng.github.io/netiquette/"
)

const (
	hiringJobBoardChannelID = "C30CUFT2B"
	hiringJobBoardWrongFormatNotificationChannelID = "G983W7L9F"
)

type specification struct {
	Port         int    `default:"8080"`
	BotUserToken string `required:"true" split_words:"true"`
	Debug        bool
}

var channels = make(map[string]*slack.Channel)

func main() {
	var s specification
	err := envconfig.Process("candebot", &s)
	if err != nil {
		log.Fatal(err.Error())
	}

	bot := slacker.NewClient(s.BotUserToken, slacker.WithDebug(s.Debug))
	bot.EventHandler(eventHandler(bot.Client()))

	registerCommands(bot)
	go registerSlashCommands(s)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func eventHandler(c *slack.Client) slacker.EventHandler {
	return func(ctx context.Context, s *slacker.Slacker, msg slack.RTMEvent) error {
	switch event := msg.Data.(type) {
	case *slack.MessageEvent:
		if event.Channel == hiringJobBoardChannelID {
			r, _ := regexp.Compile(`([^-]{1,})\@([^-]{1,})\-([^-]{1,})\-([^-]{1,})\-([^-]{1,})(\-[^-]{1,}){0,}`)
			matched := r.MatchString(event.Text)
			if !matched {
				link, err := c.GetPermalink(&slack.PermalinkParameters{
					Channel: event.Channel,
					Ts:      event.Timestamp,
				})
				if err != nil {
					log.Printf("error fetching permalink for channel %s and ts %s\n", hiringJobBoardWrongFormatNotificationChannelID, event.Timestamp)
				}

				_ = send(
					c,
					hiringJobBoardWrongFormatNotificationChannelID,
					fmt.Sprintf("new Job post with invalid format: %s", link),
				)
			}
		}
	}
	return slacker.DefaultEventHandler(ctx,  s, msg)
}
}

func registerSlashCommands(s specification) {
	http.HandleFunc("/slash", func(w http.ResponseWriter, r *http.Request) {
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// TODO verify request

		switch s.Command {
		case "/coc":
			msg := &slack.Msg{Text: msgCOC}
			writeSlashResponse(w, msg)
		case "/netiquette":
			msg := &slack.Msg{Text: msgNetiquette}
			writeSlashResponse(w, msg)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	log.Println("[INFO] Slash server listening on port", s.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil))
}

func registerCommands(bot *slacker.Slacker) {
	bot.DefaultCommand(func(request slacker.Request, response slacker.ResponseWriter) {
		c, err := channel(bot.Client(), request.Event().Channel)
		if err != nil {
			return
		}

		msg := "Say what?, try typing `help` to see all the things I can do for you ;)"
		if !c.IsGeneral {
			response.Reply(msg)

			return
		}

		_ = sendEphemeral(bot.Client(), request.Event().Channel, request.Event().User, msg)

	})

	bot.Command("coc", &slacker.CommandDefinition{
		Description: "Link to the Code Of Conduct of BcnEng",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply(msgCOC)
		},
	})

	bot.Command("netiquette", &slacker.CommandDefinition{
		Description: "Link to the netiquette of BcnEng",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply(msgNetiquette)
		},
	})

	bot.Command("staff", &slacker.CommandDefinition{
		Description: "Info about the staff behind BcnEng",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			m := `
Here is the list of the current staff members:

• Owners
   • <@gonzaloserrano>
   • <@smoya>
• Admins
   • <@mavi>
   • <@sdecandelario>
   • <@UAG4H8GMD>
   • <@U7PQZMZ4L>
`

			response.Reply(m)
		},
	})

	bot.Command("version", &slacker.CommandDefinition{
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply("`" + Version + "`")
		},
	})
}

func channel(c *slack.Client, id string) (channel *slack.Channel, err error) {
	if channel, ok := channels[id]; ok {
		return channel, nil
	}

	channel, err = c.GetChannelInfo(id)
	if err != nil {
		log.Println("error on retrieving channel info: ", err.Error())
	}

	channels[id] = channel
	return
}

func sendEphemeral(c *slack.Client, channelID, userID, msg string) error {
	_, err := c.PostEphemeral(channelID, userID, slack.MsgOptionText(msg, true), slack.MsgOptionAsUser(true))

	if err != nil {
		log.Println("error sending ephemeral msg in channel ", channelID)
	}

	return err
}

func send(c *slack.Client, channelID, msg string) error {
	_, _, err := c.PostMessage(channelID, slack.MsgOptionText(msg, true), slack.MsgOptionAsUser(true))

	if err != nil {
		log.Println("error sending msg in channel ", channelID)
	}

	return err
}

func writeSlashResponse(w http.ResponseWriter, msg *slack.Msg) {
	b, err := json.Marshal(msg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
