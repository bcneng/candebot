package bot

import (
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
)

func fakeUserName(id string) string {
	return map[string]string{"U1": "alice", "U2": "bob"}[id]
}

func TestSlackToMarkdown(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text", "hello world", "hello world"},
		{"user mention", "ping <@U1>", "ping @alice"},
		{"user mention with label", "ping <@U1|al>", "ping @al"},
		{"channel reference", "see <#C123|general>", "see #general"},
		{"channel reference without label", "see <#C123>", "see #C123"},
		{"special mention", "<!here> look", "@here look"},
		{"user group", "<!subteam^S1|@staff> look", "@staff look"},
		{"link with label", "<https://bcneng.org|BcnEng>", "[BcnEng](https://bcneng.org)"},
		{"bare link", "<https://bcneng.org>", "https://bcneng.org"},
		{"html entities", "a &lt; b &amp;&amp; c &gt; d", "a < b && c > d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, slackToMarkdown(tt.in, fakeUserName))
		})
	}
}

func TestRenderThreadMarkdown(t *testing.T) {
	messages := []slack.Message{
		{Msg: slack.Msg{User: "U1", Timestamp: "1700000000.000100", Text: "Spec for <@U2>"}},
		{Msg: slack.Msg{
			User:      "U2",
			Timestamp: "1700000060.000200",
			Text:      "LGTM",
			Files:     []slack.File{{Name: "diagram.png", Permalink: "https://files.example/diagram.png"}},
		}},
		{Msg: slack.Msg{BotID: "B1", Username: "candebot", Timestamp: "1700000120.000300", Text: "Done"}},
	}

	got := renderThreadMarkdown(messages, threadExportMeta{
		ChannelName: "general",
		Link:        "https://bcneng.slack.com/archives/C1/p1700000000000100",
		ExportedAt:  time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
	}, fakeUserName)

	want := `# Slack thread export

- Channel: #general
- Link: https://bcneng.slack.com/archives/C1/p1700000000000100
- Messages: 3
- Exported at: 2026-09-29T10:00:00Z

---

**alice** · 2023-11-14 22:13 UTC

Spec for @bob

---

**bob** · 2023-11-14 22:14 UTC

LGTM

Files:
- [diagram.png](https://files.example/diagram.png)

---

**candebot** · 2023-11-14 22:15 UTC

Done
`
	require.Equal(t, want, got)
}
