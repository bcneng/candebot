package cmd

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/slackx"
)

type Version struct{}

func (c *Version) Run(cliCtx *kong.Context, ctx BotContext, slackCtx SlackContext) error {
	msg := fmt.Sprintf("`%s`", ctx.Version)
	if ctx.CLI {
		_, err := cliCtx.Stdout.Write([]byte(msg))
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msg, false)
}
