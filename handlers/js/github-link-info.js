// GitHub Link Info Handler
// When someone posts a GitHub issue/PR link, adds context via reaction or thread

var name = "github-link-info";
var description = "Adds reactions to messages containing GitHub issue/PR links";
var channels = ["*"]; // All channels - use with caution
var priority = 200; // Run after other handlers
var enabled = false; // Disabled by default - enable if you want this feature
var timeout = 10000; // 10 second timeout for API calls

// Regex to match GitHub issue/PR URLs
var githubPattern = /https?:\/\/github\.com\/([^\/]+)\/([^\/]+)\/(issues|pull)\/(\d+)/g;

function handle(message) {
    // Find GitHub links in the message
    var matches = [];
    var match;

    // Reset regex lastIndex for global matching
    githubPattern.lastIndex = 0;

    while ((match = githubPattern.exec(message.text)) !== null) {
        matches.push({
            url: match[0],
            owner: match[1],
            repo: match[2],
            type: match[3], // "issues" or "pull"
            number: match[4]
        });
    }

    if (matches.length === 0) {
        return { handled: false };
    }

    console.info("Found " + matches.length + " GitHub link(s) in message");

    // Add appropriate emoji based on link type
    for (var i = 0; i < matches.length; i++) {
        var link = matches[i];
        var emoji = link.type === "pull" ? "git-pull-request" : "github";

        try {
            slack.addReaction(message.channel, message.timestamp, emoji);
        } catch (e) {
            // Reaction might already exist or emoji might not be available
            console.debug("Could not add reaction:", e);
        }
    }

    return { handled: true };
}
