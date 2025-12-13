// Welcome Threads Handler
// Automatically welcomes users when they start a new thread in specific channels

var handler = {
    name: "welcome-threads",
    description: "Welcomes users when they start a new thread in introduction channels",
    channels: ["introductions", "introduce-yourself", "new-members"],
    priority: 100,
    enabled: true,
    timeout: 5000,

    handle: function(message) {
        // Only respond to thread starters (messages without a thread timestamp that aren't in threads)
        if (message.isThread || message.threadTimestamp) {
            return { handled: false };
        }

        // Skip if it's a bot message
        if (message.botId) {
            return { handled: false };
        }

        // Add a wave reaction to welcome posts
        try {
            slack.addReaction(message.channel, message.timestamp, "wave");
            console.info("Added wave reaction to introduction from user:", message.user);
        } catch (e) {
            console.error("Failed to add reaction:", e);
        }

        return { handled: true };
    }
};
