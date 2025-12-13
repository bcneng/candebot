// Keyword Alert Handler
// Monitors channels for specific keywords and can notify via webhook or reactions

// Configure your keywords and reactions here
var config = {
    // Keywords to watch for (case-insensitive)
    keywords: [
        { word: "urgent", reaction: "rotating_light" },
        { word: "help", reaction: "raised_hands" },
        { word: "bug", reaction: "bug" },
        { word: "security", reaction: "lock" },
        { word: "down", reaction: "warning" },
        { word: "outage", reaction: "fire" }
    ],

    // Optional webhook URL for external notifications
    // webhookUrl: "https://hooks.slack.com/services/xxx/yyy/zzz",

    // Only alert on new threads (not replies)
    onlyNewThreads: true
};

var handler = {
    name: "keyword-alert",
    description: "Monitors messages for specific keywords and adds reactions",
    channels: [], // Empty by default - configure with your channels
    priority: 150,
    enabled: false, // Enable and configure before use
    timeout: 5000,

    handle: function(message) {
        // Skip if only monitoring new threads and this is a reply
        if (config.onlyNewThreads && message.isThread) {
            return { handled: false };
        }

        // Skip bot messages
        if (message.botId) {
            return { handled: false };
        }

        var text = message.text.toLowerCase();
        var matchedKeywords = [];

        // Check for keywords
        for (var i = 0; i < config.keywords.length; i++) {
            var kw = config.keywords[i];
            if (text.indexOf(kw.word.toLowerCase()) !== -1) {
                matchedKeywords.push(kw);
            }
        }

        if (matchedKeywords.length === 0) {
            return { handled: false };
        }

        // Add reactions for matched keywords
        for (var j = 0; j < matchedKeywords.length; j++) {
            try {
                slack.addReaction(message.channel, message.timestamp, matchedKeywords[j].reaction);
            } catch (e) {
                console.debug("Could not add reaction:", e);
            }
        }

        // Send webhook notification if configured
        if (config.webhookUrl) {
            try {
                var payload = {
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
                };

                http.post(config.webhookUrl, payload, {
                    headers: { "Content-Type": "application/json" }
                });
            } catch (e) {
                console.error("Failed to send webhook:", e);
            }
        }

        console.info("Keyword alert triggered for:", matchedKeywords.map(function(k) { return k.word; }).join(", "));

        return { handled: true };
    }
};
