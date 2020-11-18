package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/slackx"
)

const msgCOC = "Please find our Code Of Conduct here: https://bcneng.org/coc"

type Coc struct{}

func (c *Coc) Run(cliCtx *kong.Context, ctx BotContext, slackCtx SlackContext) error {
	if ctx.CLI {
		_, err := cliCtx.Stdout.Write([]byte(msgCOC))
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msgCOC, false)
}
