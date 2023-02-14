package cmd

import (
	"fmt"
	"github.com/bcneng/candebot/bot"
	"time"

	"github.com/alecthomas/kong"
	"github.com/bcneng/candebot/slackx"
)

const sdecandelarioBirthday = "17/09/2019"

type CandeBirthday struct{}

func (c *CandeBirthday) Run(cliCtx *kong.Context, ctx bot.Context, slackCtx bot.SlackContext) error {
	dob, _ := time.Parse("2/1/2006", sdecandelarioBirthday) // nolint: errcheck
	d := calculateTimeUntilBirthday(dob)

	var msg string
	if d.Hours() == 0 {
		msg = ":birthdaypartyparrot: :party: :birthday: HAPPY BIRTHDAY <@sdecandelario>! :birthday: :party: :birthdaypartyparrot:"
	} else {
		msg = fmt.Sprintf(":birthday: %d days until <@sdecandelario> birthday! :birthday:", int(d.Hours()/24))
	}

	if ctx.CLI {
		_, err := cliCtx.Stdout.Write([]byte(msg))
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, msg, false)
}

func calculateTimeUntilBirthday(t time.Time) time.Duration {
	loc, _ := time.LoadLocation("Europe/Madrid")
	n := time.Now().In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	birthday := time.Date(today.Year(), t.Month(), t.Day(), 0, 0, 0, 0, n.Location())

	if birthday.Before(today) {
		// birthday next year!
		birthday = birthday.AddDate(1, 0, 0)
	}

	return birthday.Sub(today)
}
