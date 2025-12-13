// Keyword Alert Handler
// Monitors channels for specific keywords and can notify via webhook or reactions

var config = {
    keywords: [
        { word: "urgent", reaction: "rotating_light" },
        { word: "help", reaction: "raised_hands" },
        { word: "bug", reaction: "bug" },
        { word: "security", reaction: "lock" },
        { word: "down", reaction: "warning" },
        { word: "outage", reaction: "fire" }
    ],
    // webhookUrl: "https://hooks.slack.com/services/xxx/yyy/zzz",
    onlyNewThreads: true
};

var handler = {
    name: "keyword-alert",
    description: "Monitors messages for specific keywords and adds reactions",
    channels: [],
    priority: 150,
    enabled: false,
    skipBots: true,

    handle: function(message) {
        if (config.onlyNewThreads && message.isThread) {
            return SKIP;
        }

        var text = message.text.toLowerCase();
        var matchedKeywords = [];

        for (var i = 0; i < config.keywords.length; i++) {
            var kw = config.keywords[i];
            if (text.indexOf(kw.word.toLowerCase()) !== -1) {
                matchedKeywords.push(kw);
            }
        }

        if (matchedKeywords.length === 0) {
            return SKIP;
        }

        for (var j = 0; j < matchedKeywords.length; j++) {
            message.react(matchedKeywords[j].reaction);
        }

        if (config.webhookUrl) {
            http.post(config.webhookUrl, {
                text: "Keyword alert triggered",
                attachments: [{
                    color: "warning",
                    fields: [
                        { title: "Channel", value: message.channel, short: true },
                        { title: "User", value: message.user, short: true },
                        { title: "Keywords", value: matchedKeywords.map(function(k) { return k.word; }).join(", "), short: true },
                        { title: "Message", value: message.text.substring(0, 200) }
                    ]
                }]
            }, { headers: { "Content-Type": "application/json" } });
        }

        console.info("Keyword alert triggered for:", matchedKeywords.map(function(k) { return k.word; }).join(", "));
        return HANDLED;
    }
};
