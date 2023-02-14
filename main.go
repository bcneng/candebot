package main

import (
	"context"
	"flag"
	"github.com/asaskevich/EventBus"
	"github.com/bcneng/candebot/handlers"
	"github.com/slack-go/slack/slackevents"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bcneng/candebot/bot"
)

// Version is the bot version. Usually the git commit hash. Passed during building.
var Version = "unknown"

type initConfig struct {
	ConfigFilePath string `env:"CONFIG_FILE_PATH"`
	EnvVarsPrefix  string `env:"ENV_VARS_PREFIX"`
}

var initConf = initConfig{}

func init() {
	flag.StringVar(&initConf.ConfigFilePath, "config", "./.bot.toml", "path to config file (TOML)")
	flag.StringVar(&initConf.EnvVarsPrefix, "env-prefix", "BOT_", "path to config file (TOML)")

	flag.Parse()
}

func main() {
	var conf bot.Config
	conf.Version = Version

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO Prefix and filepath from argument
	err := bot.LoadConfigFromFileAndEnvVars(ctx, initConf.EnvVarsPrefix, initConf.ConfigFilePath, &conf)
	if err != nil {
		log.Fatal(err)
	}

	bus := EventBus.New()
	subscribe(bus, slackevents.Message, handlers.MessageEventHandler)
	subscribe(bus, slackevents.AppMention, handlers.AppMentionEventHandler)

	ensureInterruptionsGracefullyShutdown(cancel)
	if err := bot.WakeUp(ctx, conf, bus); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func subscribe(bus EventBus.Bus, t slackevents.EventsAPIType, h bot.EventHandler) {
	if err := bus.Subscribe(string(t), bot.CreateEventHandler(t, h)); err != nil {
		log.Fatal(err)
	}
}

func ensureInterruptionsGracefullyShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		log.Println("Shutting down Candebot...")

		cancel()
		time.Sleep(time.Second)
		os.Exit(0)
	}()
}
