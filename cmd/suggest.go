package cmd

import (
	"errors"

	"github.com/bcneng/candebot/bot"
	"github.com/bcneng/candebot/slackx"
	"github.com/bcneng/candebot/suggest"
)

type Suggest struct{}

func (c *Suggest) Run(ctx bot.Context, slackCtx bot.SlackContext) error {
	if !ctx.IsStaff(slackCtx.User) {
		return errors.New("this action is only allowed to Staff members")
	}

	if ctx.ChannelSuggester == nil {
		return errors.New("channel suggester is not configured")
	}

	channels, err := ctx.ChannelSuggester.FetchChannels()
	if err != nil {
		return err
	}

	suggestable := suggest.FilterSuggestable(channels)
	selected := suggest.SelectRandom(suggestable, ctx.Config.ChannelSuggester.NumChannels)
	if len(selected) == 0 {
		return errors.New("no suggestable channels found")
	}

	msg := suggest.FormatMessage(selected)
	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msg, false)
}
