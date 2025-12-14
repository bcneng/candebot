// Example Handler - demonstrates the full handler interface
// File names starting with underscore are ignored by the loader

var handler = {
    // === HANDLER METADATA ===
    name: "example-handler",
    description: "Example handler that demonstrates all available features",
    channels: ["playground", "offtopic-*"],  // Glob patterns supported
    priority: 100,    // Lower runs first (default: 100)
    enabled: false,   // Set to true to enable
    timeout: 5000,    // Execution timeout in ms (default: 5000)

    // === SKIP OPTIONS (auto-skip before handle is called) ===
    skipBots: true,      // Skip messages from bots
    skipThreads: false,  // Skip thread replies
    skipDMs: false,      // Skip direct messages

    // === MAIN HANDLER ===
    handle: function(message) {
        // Message object:
        // {
        //   channel: "C12345678",           // Channel ID
        //   channelName: "general",         // Channel name
        //   channelType: "channel",         // "channel", "im", "mpim", "group"
        //   user: "U12345678",              // User ID
        //   text: "Hello world",            // Message text
        //   timestamp: "1234567890.123456", // Message timestamp
        //   threadTimestamp: "",            // Thread parent timestamp
        //   isThread: false,                // true if in a thread
        //   isDM: false,                    // true if direct message
        //   isBot: false,                   // true if from a bot
        //   isStaff: false,                 // true if user is staff
        //   botId: "",                      // Bot ID if from a bot
        //   subType: ""                     // Message subtype
        // }

        console.log("Received:", message.text);

        // === MESSAGE HELPER METHODS ===
        // message.reply(text, opts)         - Send reply to channel
        // message.replyEphemeral(text, opts) - Send ephemeral reply (only visible to user)
        // message.react(emoji)              - Add reaction to message
        // message.delete()                  - Delete the message

        if (message.text.includes("hello")) {
            message.react("wave");
            message.replyEphemeral("Hello there!");
            return HANDLED;
        }

        // === SLACK API (for advanced use) ===
        // slack.sendMessage(channel, text, opts)
        // slack.sendEphemeral(channel, user, text, opts)
        // slack.addReaction(channel, timestamp, emoji)
        // slack.removeReaction(channel, timestamp, emoji)
        // slack.deleteMessage(channel, timestamp)
        // slack.getUserInfo(userId) -> { id, name, realName, isBot, ... }
        // slack.getChannelInfo(channelId) -> { id, name, isPrivate, ... }

        // === HTTP API ===
        // http.get(url, opts) -> { status, body, headers }
        // http.post(url, body, opts) -> { status, body, headers }

        // === AI API (Gemini) ===
        // ai.generate(prompt) -> string response

        // === STATE API (auto-serializes objects) ===
        // state.cache.get(key)    - Get value (in-memory, lost on restart)
        // state.cache.set(key, value)
        // state.store.get(key)    - Get value (persisted)
        // state.store.set(key, value)

        // === RETURN VALUES ===
        // return SKIP;     // { handled: false } - didn't handle
        // return HANDLED;  // { handled: true } - handled, continue to next handler
        // return STOP;     // { handled: true, stopPropagation: true } - stop all handlers

        return SKIP;
    }
};
