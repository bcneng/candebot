package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bcneng/candebot/bot"
)

// Version is the candebot version. Usually the git commit hash. Passed during building.
var Version = "unknown"

func main() {
	var conf bot.Config
	conf.Version = Version

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO Prefix and filepath from argument
	err := bot.LoadConfigFromFileAndEnvVars(ctx, "CANDEBOT_", ".candebot.toml", &conf)
	if err != nil {
		log.Fatal(err)
	}

	ensureInterruptionsGracefullyShutdown(cancel)
	if err := bot.WakeUp(ctx, conf); err != nil && err != context.Canceled {
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
