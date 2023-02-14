package cmd

import (
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/bot"
	"github.com/bcneng/candebot/slackx"
)

type Coc struct{}

func (c *Coc) Run(cliCtx *kong.Context, ctx bot.Context, slackCtx bot.SlackContext) error {
	msg := fmt.Sprintf("Please find our Code Of Conduct here: %s", ctx.Config.Links.COC)
	if ctx.CLI {
		_, err := cliCtx.Stdout.Write([]byte(msg))
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msg, false)
}
