package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/bcneng/candebot/cmd"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack"
)

const (
	msgCOC        = "Please find our Code Of Conduct here: https://bcneng.org/coc"
	msgNetiquette = "Please find our Netiquette here: https://bcneng.org/netiquette"
)

const (
	channelHiringJobBoard  = "C30CUFT2B"
	channelStaff           = "G983W7L9F"
	channelCandebotTesting = "CK32YCX5M"
)

const (
	candebotUser  = "UJNQU8N5Q"
	candebotBotID = "BJNQBKGJF"
)

var staff = []string{
	"U2Y6QQHST", //<@gonzaloserrano>
	"U2WPLA0KA", //<@smoya>
	"U3256HZH9", //<@mavi>
	"U36H6F3CN", //<@sdecandelario>
	"UHHJ97JBF", //<@cristina_verdi>
	"U2XDM2L0G", //<@ronnylt>
}

// WakeUp wakes up Candebot.
func WakeUp(_ context.Context, conf Config) error {
	client := slack.New(conf.Bot.UserToken)
	adminClient := slack.New(conf.Bot.AdminToken)

	cliContext := cmd.BotContext{
		Client:              client,
		AdminClient:         adminClient,
		SigningSecret:       conf.Bot.Server.SigningSecret,
		StaffMembers:        staff,
		TwitterCredentials:  conf.Twitter.Credentials,
		TwitterContestToken: conf.TwitterContestToken,
		Version:             conf.Version,
	}

	if conf.NewRelicLicenseKey != "" {
		h, err := telemetry.NewHarvester(telemetry.ConfigAPIKey(conf.NewRelicLicenseKey))
		if err != nil {
			return err
		}
		cliContext.Harvester = h
	} else {
		log.Println("[WARN] No metrics will be sent to NR as there is no License Key configured")
	}

	return serve(conf, cliContext)
}

func serve(conf Config, cliContext cmd.BotContext) error {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

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

	http.HandleFunc("/events", eventsAPIHandler(cliContext))
	http.HandleFunc("/interact", interactAPIHandler(cliContext))

	log.Println("[INFO] Slash server listening on port", conf.Bot.Server.Port)

	return http.ListenAndServe(fmt.Sprintf(":%d", conf.Bot.Server.Port), nil)
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
