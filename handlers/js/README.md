# Candebot Extensible Message Handlers

This directory contains JavaScript handlers that extend candebot's functionality without modifying the core source code.

## Quick Start

1. Create a new `.js` file in this directory
2. Define handler metadata and a `handle` function
3. Restart the bot (handlers are loaded at startup)

## Handler Structure

```javascript
// Required: Name your handler
var name = "my-handler";

// Optional: Describe what it does
var description = "A description of what this handler does";

// Required: Specify which channels this handler runs in
// Empty array = no channels (disabled)
var channels = ["general", "offtopic-*", "/^hiring-/"];

// Optional: Execution priority (lower = runs first, default: 100)
var priority = 100;

// Optional: Enable/disable the handler (default: true)
var enabled = true;

// Optional: Timeout in milliseconds (default: 5000)
var timeout = 5000;

// Required: The main handler function
function handle(message) {
    // Your logic here

    // Return result
    return {
        handled: true,           // Did this handler process the message?
        stopPropagation: false   // Prevent other handlers from running?
    };
}
```

## Channel Patterns

Handlers specify which channels they apply to using the `channels` array:

| Pattern | Description | Example |
|---------|-------------|---------|
| `"general"` | Exact match | Only `#general` |
| `"offtopic-*"` | Glob pattern | `#offtopic-games`, `#offtopic-random` |
| `"/^hiring-/"` | Regex (in slashes) | `#hiring-frontend`, `#hiring-backend` |
| `"*"` | Wildcard | All channels |

**Note:** An empty `channels` array means the handler won't match any channel (opt-in required).

## Message Object

The `handle` function receives a message object with these properties:

```javascript
{
    type: "message",           // Event type
    channel: "C12345678",      // Channel ID
    channelName: "general",    // Channel name
    channelType: "channel",    // "channel", "im", "mpim", "group"
    user: "U12345678",         // User ID who sent the message
    text: "Hello world",       // Message text
    timestamp: "1234567890.123456",
    threadTimestamp: "",       // Parent thread timestamp (empty if not in thread)
    isThread: false,           // true if message is in a thread
    isDM: false,               // true if direct message
    botId: "",                 // Bot ID (if message from a bot)
    subType: ""                // Message subtype
}
```

## Available APIs

### Slack API (`slack`)

```javascript
// Send a message to a channel
slack.sendMessage(channel, text, options)
// Options: { threadTimestamp: "...", broadcast: true/false }

// Send an ephemeral message (only visible to one user)
slack.sendEphemeral(channel, user, text, options)
// Options: { threadTimestamp: "..." }

// Add a reaction to a message
slack.addReaction(channel, timestamp, emoji)

// Remove a reaction from a message
slack.removeReaction(channel, timestamp, emoji)

// Get user information
var user = slack.getUserInfo(userId)
// Returns: { id, name, realName, displayName, email, isBot, isAdmin, isOwner, timezone, avatar }

// Get channel information
var channel = slack.getChannelInfo(channelId)
// Returns: { id, name, topic, purpose, isPrivate, isArchived, memberCount }

// Delete a message (requires admin permissions)
slack.deleteMessage(channel, timestamp)

// Update a message
slack.updateMessage(channel, timestamp, newText)
```

### HTTP API (`http`)

```javascript
// GET request
var response = http.get(url, options)

// POST request
var response = http.post(url, body, options)

// PUT request
var response = http.put(url, body, options)

// DELETE request
var response = http.delete(url, options)

// Generic fetch (similar to browser fetch API)
var response = http.fetch(url, options)
// Options: { method: "GET", headers: {...}, body: ... }

// Response object:
{
    status: 200,               // HTTP status code
    statusText: "200 OK",      // Status text
    ok: true,                  // true if status 2xx
    headers: {...},            // Response headers
    body: "...",               // Raw response body as string
    json: {...}                // Parsed JSON (if Content-Type is application/json)
}
```

### Console/Log API (`console` or `log`)

```javascript
console.log("Normal log")
console.info("Info message")
console.warn("Warning message")
console.error("Error message")
console.debug("Debug message")
```

## Examples

### React to Mentions of Keywords

```javascript
var name = "keyword-reactor";
var channels = ["*"];
var enabled = true;

function handle(message) {
    if (message.text.toLowerCase().indexOf("urgent") !== -1) {
        slack.addReaction(message.channel, message.timestamp, "rotating_light");
        return { handled: true };
    }
    return { handled: false };
}
```

### Send Welcome Message in Introduction Channels

```javascript
var name = "welcome-new-members";
var channels = ["introductions", "new-members"];
var enabled = true;

function handle(message) {
    // Only respond to new threads (not replies)
    if (message.isThread) {
        return { handled: false };
    }

    slack.addReaction(message.channel, message.timestamp, "wave");
    return { handled: true };
}
```

### Fetch External Data

```javascript
var name = "github-status";
var channels = ["engineering"];
var enabled = true;

function handle(message) {
    if (message.text.indexOf("!ghstatus") === -1) {
        return { handled: false };
    }

    var response = http.get("https://www.githubstatus.com/api/v2/status.json");

    if (response.ok && response.json) {
        var status = response.json.status.description;
        slack.sendMessage(message.channel, "GitHub Status: " + status, {
            threadTimestamp: message.timestamp
        });
    }

    return { handled: true };
}
```

## Configuration

The handler system can be configured via environment variables or the `.bot.toml` config file:

```toml
[handlers]
dir = "handlers/js"  # Directory containing handler files
enabled = true            # Enable/disable the entire handler system
default_timeout = 5000    # Default timeout in milliseconds
```

Or via environment variables:
```
BOT_HANDLERS_DIR=handlers/js
BOT_HANDLERS_ENABLED=true
BOT_HANDLERS_DEFAULT_TIMEOUT=5000
```

## Security Considerations

1. **Sandboxed Execution**: Handlers run in a JavaScript sandbox (goja) with limited access
2. **No File System Access**: Handlers cannot read or write files
3. **Controlled HTTP Access**: HTTP requests are proxied through Go's HTTP client with:
   - Blocked localhost/internal addresses
   - Configurable allowed/blocked hosts
   - Request timeouts
4. **Execution Timeouts**: Each handler has a maximum execution time
5. **No Eval**: Dynamic code execution is restricted

## File Naming Conventions

- `example.js` - Normal handler file (loaded automatically)
- `_example.js` - Starts with underscore, ignored by loader (use for templates/docs)
- `example.js.disabled` - Ends with `.disabled`, ignored (disabled handler)

## Debugging

Handler logs appear in the bot's standard output with the handler name prefix:

```
[handler:my-handler] [INFO] Processing message...
[handler:my-handler] [ERROR] Failed to send message
```

## Contributing New Handlers

1. Create your handler in `handlers/js/`
2. Test it in a playground channel first (`enabled = false` or limited `channels`)
3. Submit a PR with your handler
4. Include documentation in comments explaining what it does
