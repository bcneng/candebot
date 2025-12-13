// Example Handler - demonstrates the full handler interface
// File names starting with underscore are ignored by the loader (use for documentation)

var handler = {
    name: "example-handler",
    description: "Example handler that demonstrates all available features",
    channels: ["playground", "offtopic-*"], // Matches 'playground' and any channel starting with 'offtopic-'
    priority: 100, // Lower runs first (default: 100)
    enabled: false, // Set to true to enable this handler
    timeout: 5000, // Execution timeout in milliseconds (default: 5000)

    // The main handler function - receives message data
    handle: function(message) {
        // Message object contains:
        // {
        //   type: "message",
        //   channel: "C12345678",      // Channel ID
        //   channelName: "general",    // Channel name (may be same as ID)
        //   channelType: "channel",    // "channel", "im", "mpim", "group"
        //   user: "U12345678",         // User ID who sent the message
        //   text: "Hello world",       // Message text
        //   timestamp: "1234567890.123456",
        //   threadTimestamp: "",       // Thread parent timestamp (empty if not in thread)
        //   isThread: false,           // true if message is in a thread
        //   isDM: false,               // true if direct message
        //   isStaff: false,            // true if user is a staff member
        //   botId: "",                 // Bot ID if message from a bot
        //   subType: ""                // Message subtype (empty for normal messages)
        // }

        console.log("Received message:", message.text);
        console.info("In channel:", message.channel);

        // === SLACK API ===

        // Send a message to a channel
        // slack.sendMessage(channel, text, options)
        // Options: { threadTimestamp: "...", broadcast: true/false, unfurlLinks: true/false }

        // Send an ephemeral message (only visible to one user)
        // slack.sendEphemeral(channel, user, text, options)
        // Options: { threadTimestamp: "..." }

        // Add/remove reactions
        // slack.addReaction(channel, timestamp, emoji)
        // slack.removeReaction(channel, timestamp, emoji)

        // Get user info
        // var user = slack.getUserInfo(userId)
        // Returns: { id, name, realName, displayName, email, isBot, isAdmin, isOwner, timezone, avatar }

        // Get channel info
        // var channel = slack.getChannelInfo(channelId)
        // Returns: { id, name, topic, purpose, isPrivate, isArchived, memberCount }

        // Delete a message (requires admin permissions)
        // slack.deleteMessage(channel, timestamp)

        // Update a message
        // slack.updateMessage(channel, timestamp, newText)

        // === HTTP API ===

        // HTTP GET request
        // var response = http.get(url, options)
        // Options: { headers: { "Authorization": "Bearer token" } }

        // HTTP POST request
        // var response = http.post(url, body, options)
        // Body can be string, object (will be JSON-encoded), or any value

        // HTTP PUT request
        // var response = http.put(url, body, options)

        // HTTP DELETE request
        // var response = http.delete(url, options)

        // Generic HTTP fetch (similar to browser fetch API)
        // var response = http.fetch(url, options)
        // Options: { method: "GET/POST/PUT/DELETE", body: ..., headers: {...} }

        // Response object:
        // {
        //   status: 200,
        //   statusText: "200 OK",
        //   ok: true,              // true if status 2xx
        //   headers: {...},        // Response headers
        //   body: "...",           // Raw response body as string
        //   json: {...}            // Parsed JSON if Content-Type is application/json
        // }

        // === CONSOLE/LOG API ===

        // console.log(...args)   - Standard log
        // console.info(...args)  - Info level
        // console.warn(...args)  - Warning level
        // console.error(...args) - Error level
        // console.debug(...args) - Debug level
        // Also available as: log.info(), log.warn(), etc.

        // === RETURN VALUE ===

        // Return an object to control handler behavior:
        return {
            handled: true,           // Set to true if this handler processed the message
            stopPropagation: false   // Set to true to prevent other handlers from running
        };
    }
};
