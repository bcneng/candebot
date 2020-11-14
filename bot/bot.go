package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bcneng/candebot/inclusion"
	"github.com/shomali11/slacker"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	msgCOC        = "Please find our Code Of Conduct here: https://bcneng.org/coc"
	msgNetiquette = "Please find our Netiquette here: https://bcneng.org/netiquette"
)

const (
	sdecandelarioBirthday = "17/09/2019"
)

const (
	channelHiringJobBoard                        = "C30CUFT2B"
	channelHiringJobBoardWrongFormatNotification = "G983W7L9F"
	channelCandebotTesting                       = "CK32YCX5M"
)

const candebotUser = "UJNQU8N5Q"

var staff = []string{
	"U2Y6QQHST", //<@gonzaloserrano>
	"U2WPLA0KA", //<@smoya>
	"U3256HZH9", //<@mavi>
	"U36H6F3CN", //<@sdecandelario>
	"U7PQZMZ4L", //<@koe>
}

// Cache and optimizations
var (
	staffMap             map[string]struct{}
	channelNameToIDCache map[string]string
)

var jobOfferRegex = regexp.MustCompile(`(?is)^:computer:\s([^-]{1,50})\s@\s([^-]{1,50})\s-\s:moneybag:\s([^-]{1,10})?\s?-\s([^-]{1,20})\s-\s:round_pushpin:\s(.+)\s-\s:link:\s((?:http:\/\/www\.|https:\/\/www\.|http:\/\/|https:\/\/)?[a-z0-9]+(?:[\-\.]{1}[a-z0-9]+)*\.[a-z]{2,5}(?::[0-9]{1,5})?(?:\/.*)?)\s-\s:raised_hands:\sMore\sinfo\sDM\s([^-]+)$`)

// WakeUp wakes up Candebot.
func WakeUp(ctx context.Context, conf Config) error {
	bot := slacker.NewClient(conf.BotUserToken, slacker.WithDebug(conf.Debug))
	registerCommands(conf, bot)

	adminClient := slack.New(conf.UserToken)
	go registerServer(conf, bot.Client(), adminClient)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	return bot.Listen(ctx)
}

func registerServer(conf Config, botClient, adminClient *slack.Client) {
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

	// New Events API. This should eventually replace any usage of RTM (including slacker library).
	http.HandleFunc("/events", eventsAPIHandler(botClient, adminClient))

	log.Println("[INFO] Slash server listening on port", conf.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil))
}
func registerCommands(conf Config, bot *slacker.Slacker) {
	bot.DefaultCommand(func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
		msg := "Say what?, try typing `help` to see all the things I can do for you ;)"
		_ = sendEphemeral(bot.Client(), botContext.Event().Channel, botContext.Event().User, msg)
	})

	bot.Command("coc", &slacker.CommandDefinition{
		Description: "Link to the Code Of Conduct of BcnEng",
		Handler: func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			response.Reply(msgCOC)
		},
	})

	bot.Command("netiquette", &slacker.CommandDefinition{
		Description: "Link to the netiquette of BcnEng",
		Handler: func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			response.Reply(msgNetiquette)
		},
	})

	dob, _ := time.Parse("2/1/2006", sdecandelarioBirthday) // nolint: errcheck
	bot.Command("candebirthday", &slacker.CommandDefinition{
		Description: "Days until @sdecandelario birthday!",
		Handler: func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			d := calculateTimeUntilBirthday(dob)

			var msg string
			if d.Hours() == 0 {
				msg = ":birthdaypartyparrot: :party: :birthday: HAPPY BIRTHDAY <@sdecandelario>! :birthday: :party: :birthdaypartyparrot:"
			} else {
				msg = fmt.Sprintf(":birthday: %d days until <@sdecandelario> birthday! :birthday:", int(d.Hours()/24))
			}

			response.Reply(msg)
		},
	})

	bot.Command("staff", &slacker.CommandDefinition{
		Description: "Info about the staff behind BcnEng",
		Handler: func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			// Shuffle the order of members list
			shuffledMembers := staff
			rand.Shuffle(len(shuffledMembers), func(i, j int) {
				shuffledMembers[i], shuffledMembers[j] = shuffledMembers[j], shuffledMembers[i]
			})

			members := strings.Join(shuffledMembers, ">\n• <@")
			m := fmt.Sprintf("Here is the list of the current staff members: \n\n• <@%s>", members)

			response.Reply(m)
		},
	})

	bot.Command("echo <channel> <message>", &slacker.CommandDefinition{
		Description: "Sends a message as Candebot",
		Example:     "echo #general Hi folks!",
		AuthorizationFunc: func(botContext slacker.BotContext, request slacker.Request) bool {
			return isStaff(botContext.Event().User)
		},
		Handler: func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			channel := strings.TrimPrefix(request.Param("channel"), "#")
			msg := request.Param("message")

			if channel == "" || msg == "" {
				_ = sendEphemeral(bot.Client(), botContext.Event().Channel, botContext.Event().User, "Channel and message are required.")
				return
			}

			// Fixes the lack of support of multi word params.
			if i := strings.Index(channel, " "); i > 0 {
				msg = channel[i:] + " " + msg
				channel = channel[0:i]
			}

			channelID, err := findChannelIDByName(bot.Client(), channel)
			if err != nil {
				log.Println(err.Error())
				_ = sendEphemeral(bot.Client(), botContext.Event().Channel, botContext.Event().User, "Internal error. Try again.")
				return
			}

			err = send(bot.Client(), channelID, msg, false)
			if err != nil {
				log.Println(err.Error())
				_ = sendEphemeral(bot.Client(), botContext.Event().Channel, botContext.Event().User, "Internal error. Try again.")
				return
			}
		},
	})

	bot.Command("version", &slacker.CommandDefinition{
		Handler: func(botContext slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			response.Reply("`" + conf.Version + "`")
		},
	})
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

func findChannelIDByName(client *slack.Client, channel string) (string, error) {
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

func sendEphemeral(c *slack.Client, channelID, userID, msg string) error {
	_, err := c.PostEphemeral(channelID, userID, slack.MsgOptionText(msg, true), slack.MsgOptionAsUser(true))
	if err != nil {
		log.Println("error sending ephemeral msg in channel ", channelID)
	}

	return err
}

func send(c *slack.Client, channelID, msg string, scape bool) error {
	_, _, err := c.PostMessage(channelID, slack.MsgOptionText(msg, scape), slack.MsgOptionAsUser(true))
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

func calculateTimeUntilBirthday(t time.Time) time.Duration {
	loc, _ := time.LoadLocation("Europe/Madrid")
	n := time.Now().In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	birthday := time.Date(today.Year(), t.Month(), t.Day(), 0, 0, 0, 0, n.Location())

	if birthday.Before(today) {
		// birthday next year!
		birthday = birthday.AddDate(1, 0, 0)
	}

	return birthday.Sub(today)
}

func isStaff(userID string) bool {
	if staffMap == nil {
		staffMap = make(map[string]struct{}, len(staff)) // It is fine to not lock.
		for _, u := range staff {
			staffMap[u] = struct{}{}
		}

	}

	_, ok := staffMap[userID]

	return ok
}

func checkLanguage(botClient *slack.Client, event *slackevents.MessageEvent) {
	if reply := inclusion.Filter(event.Text); reply != "" {
		_ = sendEphemeral(botClient, event.Channel, event.User, reply)
	}
}
