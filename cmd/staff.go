package cmd

import (
	"fmt"
	"github.com/bcneng/candebot/bot"
	"math/rand"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/slackx"
)

type Staff struct{}

func (c *Staff) Run(cliCtx *kong.Context, ctx bot.Context, slackCtx bot.SlackContext) error {
	// Shuffle the order of members list
	shuffledMembers := ctx.Config.Staff.Members
	rand.Shuffle(len(shuffledMembers), func(i, j int) {
		shuffledMembers[i], shuffledMembers[j] = shuffledMembers[j], shuffledMembers[i]
	})

	members := strings.Join(shuffledMembers, ">\n• <@")
	msg := fmt.Sprintf("Here is the list of the current staff members: \n\n• <@%s>", members)

	if ctx.CLI {
		_, err := cliCtx.Stdout.Write([]byte(msg))
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msg, false)
}
