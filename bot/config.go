package bot

import "github.com/bcneng/twitter-contest/twitter"

type Config struct {
	Port                int                 `default:"8080"`
	SigningSecret       string              `required:"true" split_words:"true"`
	BotUserToken        string              `required:"true" split_words:"true"`
	UserToken           string              `required:"true" split_words:"true"`
	Twitter             twitter.Credentials `required:"true"`
	TwitterContestToken string              `required:"true" split_words:"true"`
	NewRelicLicenseKey  string              `split_words:"true"` // NEW_RELIC_LICENSE_KEY
	Debug               bool
	Version             string
}
