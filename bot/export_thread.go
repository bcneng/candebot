package bot

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack"

	"github.com/bcneng/candebot/slackx"
)

// exportThread fetches the whole thread the given message belongs to, renders it as Markdown
// and sends it as a file to the requester via DM.
func exportThread(botContext Context, requesterID, channelID, channelName, threadTS string) {
	dm, _, _, err := botContext.Client.OpenConversation(&slack.OpenConversationParameters{Users: []string{requesterID}})
	if err != nil {
		log.Printf("Thread export: could not open DM with %s: %v", requesterID, err)
		return
	}

	notify := func(msg string) {
		_ = slackx.Send(botContext.Client, "", dm.ID, msg, false)
	}

	messages, err := fetchThreadMessages(botContext.Client, channelID, threadTS)
	if err != nil {
		// The bot can only read channels it is a member of. Fall back to the admin token.
		log.Printf("Thread export: bot client could not read thread %s/%s (%v), retrying with admin client", channelID, threadTS, err)
		messages, err = fetchThreadMessages(botContext.AdminClient, channelID, threadTS)
	}
	if err != nil {
		log.Printf("Thread export: could not read thread %s/%s: %v", channelID, threadTS, err)
		notify(fmt.Sprintf("Could not export the thread %s: %s", slackx.LinkToMessage(channelID, threadTS), err))
		return
	}

	names := newUserNameResolver(botContext.Client)
	content := renderThreadMarkdown(messages, threadExportMeta{
		ChannelName: channelName,
		Link:        slackx.LinkToMessage(channelID, threadTS),
		ExportedAt:  time.Now().UTC(),
	}, names.Name)

	source := channelName
	if source == "" {
		source = channelID
	}
	filename := fmt.Sprintf("thread-%s-%s.md", source, strings.Replace(threadTS, ".", "", 1))
	_, err = botContext.Client.UploadFileV2(slack.UploadFileV2Parameters{
		Reader:         bytes.NewReader([]byte(content)),
		FileSize:       len(content),
		Filename:       filename,
		Title:          filename,
		Channel:        dm.ID,
		InitialComment: fmt.Sprintf("Export of %s (%d messages)", slackx.LinkToMessage(channelID, threadTS), len(messages)),
	})
	if err != nil {
		log.Printf("Thread export: could not upload file for %s/%s: %v", channelID, threadTS, err)
		notify(fmt.Sprintf("Could not upload the export of %s: %s", slackx.LinkToMessage(channelID, threadTS), err))
		return
	}

	log.Printf("Thread export finished successfully, %d messages exported: %s/%s", len(messages), channelID, threadTS)

	// Sending metrics
	botContext.Harvester.RecordMetric(telemetry.Count{
		Name: fmt.Sprintf("%s.%s", strings.ToLower(botContext.Config.Bot.Name), "thread.exported"),
		Attributes: map[string]interface{}{
			"messages": len(messages),
		},
		Value:     1,
		Timestamp: time.Now(),
	})
}

func fetchThreadMessages(client *slack.Client, channelID, threadTS string) ([]slack.Message, error) {
	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
	}

	var messages []slack.Message
	for more := true; more; {
		var replies []slack.Message
		var err error
		replies, more, params.Cursor, err = client.GetConversationReplies(params)
		if err != nil {
			return nil, err
		}
		messages = append(messages, replies...)
	}

	return messages, nil
}

type userNameResolver struct {
	client *slack.Client
	cache  map[string]string
}

func newUserNameResolver(client *slack.Client) *userNameResolver {
	return &userNameResolver{client: client, cache: map[string]string{}}
}

// Name returns the display name of the user, falling back to the user ID if it cannot be resolved.
func (r *userNameResolver) Name(userID string) string {
	if name, ok := r.cache[userID]; ok {
		return name
	}

	name := userID
	if u, err := r.client.GetUserInfo(userID); err != nil {
		log.Printf("Thread export: could not resolve user %s: %v", userID, err)
	} else {
		switch {
		case u.Profile.DisplayName != "":
			name = u.Profile.DisplayName
		case u.RealName != "":
			name = u.RealName
		case u.Name != "":
			name = u.Name
		}
	}

	r.cache[userID] = name
	return name
}

type threadExportMeta struct {
	ChannelName string
	Link        string
	ExportedAt  time.Time
}

func renderThreadMarkdown(messages []slack.Message, meta threadExportMeta, userName func(string) string) string {
	var b strings.Builder

	_, _ = b.WriteString("# Slack thread export\n\n")
	if meta.ChannelName != "" {
		fmt.Fprintf(&b, "- Channel: #%s\n", meta.ChannelName)
	}
	fmt.Fprintf(&b, "- Link: %s\n", meta.Link)
	fmt.Fprintf(&b, "- Messages: %d\n", len(messages))
	fmt.Fprintf(&b, "- Exported at: %s\n", meta.ExportedAt.Format(time.RFC3339))

	for _, m := range messages {
		_, _ = b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "**%s** · %s\n\n", messageAuthor(m, userName), formatSlackTS(m.Timestamp))

		if text := slackToMarkdown(m.Text, userName); text != "" {
			_, _ = b.WriteString(text)
			_, _ = b.WriteString("\n")
		}

		if len(m.Files) > 0 {
			_, _ = b.WriteString("\nFiles:\n")
			for _, f := range m.Files {
				fmt.Fprintf(&b, "- [%s](%s)\n", f.Name, f.Permalink)
			}
		}
	}

	return b.String()
}

func messageAuthor(m slack.Message, userName func(string) string) string {
	switch {
	case m.User != "":
		return userName(m.User)
	case m.Username != "":
		return m.Username
	case m.BotProfile != nil && m.BotProfile.Name != "":
		return m.BotProfile.Name
	case m.BotID != "":
		return m.BotID
	default:
		return "unknown"
	}
}

func formatSlackTS(ts string) string {
	sec, _, _ := strings.Cut(ts, ".")
	unix, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		return ts
	}

	return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04 MST")
}

var slackEntityRegexp = regexp.MustCompile(`<([^<>]+)>`)

// slackToMarkdown converts Slack mrkdwn entities (mentions, channel references, links) to plain Markdown.
func slackToMarkdown(text string, userName func(string) string) string {
	text = slackEntityRegexp.ReplaceAllStringFunc(text, func(entity string) string {
		target, label, hasLabel := strings.Cut(entity[1:len(entity)-1], "|")

		switch {
		case strings.HasPrefix(target, "@"):
			if hasLabel {
				return "@" + label
			}
			return "@" + userName(target[1:])
		case strings.HasPrefix(target, "#"):
			if hasLabel && label != "" {
				return "#" + label
			}
			return "#" + target[1:]
		case strings.HasPrefix(target, "!subteam^"):
			if hasLabel {
				return label
			}
			return "@group"
		case strings.HasPrefix(target, "!"):
			if hasLabel {
				return label
			}
			return "@" + target[1:]
		case hasLabel:
			return fmt.Sprintf("[%s](%s)", label, target)
		default:
			return target
		}
	})

	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(text)
}
