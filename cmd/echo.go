package cmd

import (
	"errors"
	"fmt"
	"github.com/bcneng/candebot/bot"
	"strings"

	"github.com/bcneng/candebot/slackx"
)

type Echo struct {
	Channel string `arg:"" required:"false"`
	Message string `arg:"" required:"false"`
}

func (e *Echo) Run(ctx bot.Context, slackCtx bot.SlackContext) error {
	if !ctx.IsStaff(slackCtx.User) && !ctx.CLI {
		return errors.New("this action is only allowed to Staff members")
	}

	channel := strings.TrimPrefix(e.Channel, "#")
	if channel == "" || e.Message == "" {
		_ = slackx.SendEphemeral(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, "Channel and message are required")
		return errors.New("channel and message are required")
	}

	// Fixes the lack of support of multi word params.
	if i := strings.Index(channel, " "); i > 0 {
		e.Message = channel[i:] + " " + e.Message
		channel = channel[0:i]
	}

	channelID, err := slackx.FindChannelIDByName(ctx.Client, channel)
	if err != nil {
		_ = slackx.SendEphemeral(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, fmt.Sprintf("Error during channel lookup. Error: %s", err.Error()))
		return err
	}

	if err := slackx.Send(ctx.Client, "", channelID, e.Message, false); err != nil {
		_ = slackx.SendEphemeral(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, fmt.Sprintf("Error sending message. Error: %s", err.Error()))
		return err
	}

	_ = slackx.SendEphemeral(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, "Message echoed successfully!")

	return nil
}
