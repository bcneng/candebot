package bot

import (
	"net/http"

	"github.com/asaskevich/EventBus"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack"
)

type Context struct {
	Client              *slack.Client
	AdminClient         *slack.Client
	Config              Config
	SigningSecret       string
	Version             string
	TwitterContestToken string
	Harvester           *telemetry.Harvester

	Bus EventBus.Bus

	CLI bool // true if runs from CLI

	staffLookupMap map[string]struct{}
}

func (c *Context) IsStaff(userID string) bool {
	_, ok := c.staffLookupMap[userID]
	return ok
}

func (c *Context) VerifyRequest(r *http.Request, body []byte) error {
	// Verify signing secret
	sv, err := slack.NewSecretsVerifier(r.Header, c.Config.Bot.Server.SigningSecret)
	if err != nil {
		return err
	}
	_, _ = sv.Write(body)
	return sv.Ensure()
}

type SlackContext struct {
	User            string
	Channel         string
	Text            string
	Timestamp       string
	ThreadTimestamp string
}
