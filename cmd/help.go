package cmd

import (
	"github.com/alecthomas/kong"
)

type Help struct{}

func (c *Help) Run(cliCtx *kong.Context, _ BotContext, _ SlackContext) error {
	// Don't need to explicitly run the command since help does not executes Run because it exits early. See kong/help.go
	_, _ = cliCtx.Parse([]string{"--help"})
	return nil
}
