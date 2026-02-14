package bot

import (
	"net/http"

	"github.com/asaskevich/EventBus"
	"github.com/bcneng/candebot/internal/privacy"
	"github.com/bcneng/candebot/slackx"
	"github.com/bcneng/candebot/suggest"
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
	RateLimiter         *RateLimiter
	ChannelResolver     *slackx.ChannelResolver
	TrackingDetector    *privacy.TrackingDetector
	ChannelSuggester    *suggest.ChannelSuggester

	Bus EventBus.Bus

	CLI bool // true if runs from CLI

	staffLookupMap map[string]struct{}
}

func (c *Context) IsStaff(userID string) bool {
	if c.staffLookupMap == nil {
		c.staffLookupMap = make(map[string]struct{}, len(c.Config.Staff.Members)) // It is fine to not lock.
		for _, u := range c.Config.Staff.Members {
			c.staffLookupMap[u] = struct{}{}
		}
	}

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
