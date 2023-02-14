package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bcneng/candebot/bot"
	"io/ioutil"
	"net/http"

	"github.com/bcneng/candebot/slackx"
)

type Contest struct {
	TweetID         string `arg:"" required:"false"`
	Pick            string `arg:"" required:"false"`
	AccountToFollow string `arg:"" optional:"false"`
}

func (c *Contest) Run(ctx bot.Context, slackCtx bot.SlackContext) error {
	if !ctx.IsStaff(slackCtx.User) && !ctx.CLI {
		return errors.New("this action is only allowed to Staff members")
	}

	r, _ := http.NewRequest(http.MethodGet, ctx.Config.TwitterContestURL, nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.Config.TwitterContestToken))

	q := r.URL.Query()
	q.Add("api_key", ctx.Config.Twitter.Credentials.APIKey)
	q.Add("api_key_secret", ctx.Config.Twitter.Credentials.APIKeySecret)
	q.Add("tweet_id", c.TweetID)
	q.Add("pick", c.Pick)
	q.Add("account_to_follow", c.AccountToFollow)
	r.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		_ = slackx.SendEphemeral(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, "error making request to twitter-contest")
		return err
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "\t") != nil {
		_ = slackx.SendEphemeral(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, slackCtx.User, "error prettifying json")
		return err
	}

	return slackx.Send(ctx.Client, slackCtx.ThreadTimestamp, slackCtx.Channel, pretty.String(), false)
}
