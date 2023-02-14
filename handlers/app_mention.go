package handlers

import (
	"github.com/bcneng/candebot/bot"
	"github.com/slack-go/slack/slackevents"
	"log"
)

func AppMentionEventHandler(botCtx bot.Context, e slackevents.EventsAPIInnerEvent) error {
	event := e.Data.(*slackevents.AppMentionEvent)

	log.Println("Mention message:", event.Text)
	botCommand(botCtx, bot.SlackContext{
		User:            event.User,
		Channel:         event.Channel,
		Text:            event.Text,
		Timestamp:       event.TimeStamp,
		ThreadTimestamp: event.ThreadTimeStamp,
	})

	return nil
}
