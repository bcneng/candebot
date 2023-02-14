package bot

import (
	"context"
	"fmt"
	"github.com/bcneng/twitter-contest/twitter"
	"github.com/pelletier/go-toml/v2"
	"github.com/sethvargo/go-envconfig"
	"os"
	"path"
	"strings"
)

// LoadConfigFromFile reads a TOML file and unmarshals that config into the given Config struct.
func LoadConfigFromFile(filepath string, conf *Config) error {
	extension := strings.ToLower(path.Ext(filepath))
	switch extension {
	case ".toml":
		data, err := os.ReadFile(filepath)
		if err != nil {
			return err
		}

		return LoadConfigFromBytes(data, conf)
	}
	return fmt.Errorf("%s config file extension not supported", extension)
}

// LoadConfigFromEnvVars reads config from env vars and maps that into the given Config struct.
func LoadConfigFromEnvVars(ctx context.Context, prefix string, conf *Config) error {
	l := envconfig.PrefixLookuper(prefix, envconfig.OsLookuper())
	return envconfig.ProcessWith(ctx, conf, l)
}

// LoadConfigFromBytes loads config from a raw []byte TOML file
func LoadConfigFromBytes(data []byte, conf *Config) error {
	return toml.Unmarshal(data, conf)
}

// LoadConfigFromFileAndEnvVars reads config and mapts that into the given Config in the following order:
// 1. Loads from Toml file.
// 2. Loads from env vars.
// Note: Values from env vars that were set by the TOML file will be overwritten.
func LoadConfigFromFileAndEnvVars(ctx context.Context, envVarsPrefix, filepath string, conf *Config) error {
	if err := LoadConfigFromFile(filepath, conf); err != nil {
		return err
	}

	return LoadConfigFromEnvVars(ctx, envVarsPrefix, conf)
}

type Config struct {
	Bot                 ConfigBot      `env:",prefix=BOT_"`
	Staff               ConfigStaff    `env:",prefix=STAFF_"`
	Channels            ConfigChannels `env:",prefix=CHANNELS_"`
	Links               ConfigLinks    `env:",prefix=LINKS_"`
	Twitter             ConfigTwitter  `env:",prefix=TWITTER_"`
	TwitterContestToken string         `env:"TWITTER_CONTEST_TOKEN"`
	TwitterContestURL   string         `env:"TWITTER_CONTEST_URL"`
	NewRelicLicenseKey  string         `env:"NEW_RELIC_LICENSE_KEY"`
	Debug               bool           `env:"DEBUG"`
	Version             string         `env:"VERSION"`
}

type ConfigBot struct {
	ID         string          `env:"ID,required"`
	UserID     string          `env:"USER_ID,required"`
	Name       string          `env:"NAME,required"`
	UserToken  string          `env:"USER_TOKEN,required"`
	AdminToken string          `env:"ADMIN_TOKEN,required"`
	Server     ConfigBotServer `env:",prefix=SERVER_"`
}

type ConfigStaff struct {
	Members []string `env:"MEMBERS"`
}

type ConfigChannels struct {
	Reports    string `env:"REPORTS"`
	Playground string `env:"PLAYGROUND"`
	Jobs       string `env:"JOBS"`
	Staff      string `env:"STAFF"`
}

type ConfigLinks struct {
	COC        string `env:"COC"`
	Netiquette string `env:"NETIQUETTE"`
}

type ConfigBotServer struct {
	Port          int    `env:"PORT,default=8080"`
	SigningSecret string `env:"SIGNING_SECRET,required"`
}

type ConfigTwitter struct {
	twitter.Credentials
	APIKey       string `env:"API_KEY"`
	APIKeySecret string `env:"API_KEY_SECRET"`
}
