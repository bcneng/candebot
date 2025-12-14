// TL;DR Handler
// Summarizes linked content using AI and posts as a thread reply

var urlPattern = /https?:\/\/[^\s<>]+/g;

var skipDomains = [
    'twitter.com', 'x.com',
    'facebook.com', 'fb.com', 'fb.me',
    'instagram.com',
    'linkedin.com',
    'tiktok.com',
    'reddit.com',
    'threads.net',
    'mastodon.social',
    'bsky.app',
    'youtube.com', 'youtu.be',
    'twitch.tv',
    'discord.com', 'discord.gg'
];

function extractDomain(url) {
    var match = url.match(/^https?:\/\/(?:www\.)?([^\/]+)/i);
    return match ? match[1].toLowerCase() : null;
}

function shouldSkip(url) {
    var domain = extractDomain(url);
    if (!domain) return true;
    for (var i = 0; i < skipDomains.length; i++) {
        if (domain === skipDomains[i] || domain.endsWith('.' + skipDomains[i])) {
            return true;
        }
    }
    return false;
}

var handler = {
    name: "tldr",
    description: "Summarizes linked content and posts a TL;DR in a thread",
    channels: ["*"],
    priority: 300,
    enabled: true,
    timeout: 30000,
    skipBots: true,
    skipThreads: true,

    handle: function(message) {
        var urls = [];
        var match;
        urlPattern.lastIndex = 0;

        while ((match = urlPattern.exec(message.text)) !== null) {
            var url = match[0].replace(/[.,;:!?)>\]]+$/, '');
            if (!shouldSkip(url)) {
                urls.push(url);
            }
        }

        if (urls.length === 0) return SKIP;

        var url = urls[0];
        console.info("Summarizing: " + url);

        var summary = ai.summarize(url);
        if (!summary) return SKIP;

        var domain = extractDomain(url);
        message.reply(":memo: *TL;DR* (" + domain + ")\n" + summary);

        return HANDLED;
    }
};
