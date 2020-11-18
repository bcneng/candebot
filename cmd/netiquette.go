package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/slackx"
)

const msgNetiquette = "Please find our Netiquette here: https://bcneng.org/netiquette"

type Netiquette struct{}

func (c *Netiquette) Run(cliCtx *kong.Context, ctx BotContext, slackCtx SlackContext) error {
	if ctx.CLI {
		_, err := cliCtx.Stdout.Write([]byte(msgNetiquette))
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msgNetiquette, false)
}
