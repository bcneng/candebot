package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/bot"
)

type Help struct{}

func (c *Help) Run(cliCtx *kong.Context, _ bot.Context, _ bot.SlackContext) error {
	// Don't need to explicitly run the command since help does not execute Run because it exits early. See kong/help.go
	_, _ = cliCtx.Parse([]string{"--help"})
	return nil
}
