package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/asaskevich/EventBus"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack"
)

// WakeUp wakes up the bot.
func WakeUp(_ context.Context, conf Config, bus EventBus.Bus) error {
	cliContext := Context{
		Client:      slack.New(conf.Bot.UserToken),
		AdminClient: slack.New(conf.Bot.AdminToken),
		Config:      conf,
		Version:     conf.Version,
		Bus:         bus,
	}

	if conf.NewRelicLicenseKey != "" {
		h, err := telemetry.NewHarvester(
			telemetry.ConfigAPIKey(conf.NewRelicLicenseKey),
			telemetry.ConfigCommonAttributes(map[string]interface{}{
				fmt.Sprintf("%s_version", strings.ToLower(conf.Bot.Name)): conf.Version,
			}),
		)
		if err != nil {
			return err
		}
		cliContext.Harvester = h
	} else {
		log.Println("[WARN] No metrics will be sent to NR as there is no License Key configured")
	}

	return serve(conf, cliContext)
}

func serve(conf Config, cliContext Context) error {
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
			msg := &slack.Msg{Text: fmt.Sprintf("Please find our Code Of Conduct here: %s", conf.Links.COC)}
			writeSlashResponse(w, msg)
		case "/netiquette":
			msg := &slack.Msg{Text: fmt.Sprintf("Please find our Netiquette here: %s", conf.Links.Netiquette)}
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
