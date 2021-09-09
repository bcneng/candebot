package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bcneng/candebot/slackx"
)

const TwitterContestURL = "https://bcneng-twitter-contest.netlify.app/.netlify/functions/contest"

type Contest struct {
	TweetID         string `arg:"" required:"false"`
	Pick            string `arg:"" required:"false"`
	AccountToFollow string `arg:"" optional:"false"`
}

func (c *Contest) Run(ctx BotContext, slackCtx SlackContext) error {
	if !ctx.IsStaff(slackCtx.User) && !ctx.CLI {
		return errors.New("this action is only allowed to Staff members")
	}

	r, _ := http.NewRequest(http.MethodGet, TwitterContestURL, nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.TwitterContestToken))

	q := r.URL.Query()
	q.Add("api_key", ctx.TwitterCredentials.APIKey)
	q.Add("api_key_secret", ctx.TwitterCredentials.APIKeySecret)
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
