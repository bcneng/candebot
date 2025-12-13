// GitHub Link Info Handler
// When someone posts a GitHub issue/PR link, adds context via reaction or thread

// Regex to match GitHub issue/PR URLs
var githubPattern = /https?:\/\/github\.com\/([^\/]+)\/([^\/]+)\/(issues|pull)\/(\d+)/g;

var handler = {
    name: "github-link-info",
    description: "Adds reactions to messages containing GitHub issue/PR links",
    channels: ["*"],
    priority: 200,
    enabled: false,
    timeout: 10000,

    handle: function(message) {
        var matches = [];
        var match;
        githubPattern.lastIndex = 0;

        while ((match = githubPattern.exec(message.text)) !== null) {
            matches.push({
                type: match[3], // "issues" or "pull"
                number: match[4]
            });
        }

        if (matches.length === 0) {
            return SKIP;
        }

        console.info("Found " + matches.length + " GitHub link(s) in message");

        for (var i = 0; i < matches.length; i++) {
            var emoji = matches[i].type === "pull" ? "git-pull-request" : "github";
            message.react(emoji);
        }

        return HANDLED;
    }
};
