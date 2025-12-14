// TL;DR Handler
// Summarizes linked content using AI and posts as a thread reply

// Regex to extract URLs from messages
var urlPattern = /https?:\/\/[^\s<>]+/g;

// Social networks and sites to skip
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
    try {
        var match = url.match(/^https?:\/\/(?:www\.)?([^\/]+)/i);
        return match ? match[1].toLowerCase() : null;
    } catch (e) {
        return null;
    }
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
        // Extract URLs from message
        var urls = [];
        var match;
        urlPattern.lastIndex = 0;

        while ((match = urlPattern.exec(message.text)) !== null) {
            var url = match[0].replace(/[.,;:!?)>\]]+$/, ''); // Clean trailing punctuation
            if (!shouldSkip(url)) {
                urls.push(url);
            }
        }

        if (urls.length === 0) {
            return SKIP;
        }

        console.info("Found " + urls.length + " URL(s) to summarize");

        // Process first non-social URL
        var url = urls[0];
        console.log("Fetching content from: " + url);

        // Fetch page content
        var response = http.get(url);
        if (response.status !== 200) {
            console.warn("Failed to fetch URL: " + url + " (status: " + response.status + ")");
            return SKIP;
        }

        // Extract text content (in real implementation, would parse HTML)
        var content = response.body;
        if (!content || content.length < 100) {
            console.warn("Content too short to summarize");
            return SKIP;
        }

        // Truncate content if too long (Gemini has token limits)
        if (content.length > 15000) {
            content = content.substring(0, 15000);
        }

        // Generate summary using Gemini
        var prompt = "Provide a brief TL;DR summary (2-3 sentences max) of the following web page content. " +
                     "Focus on the main topic and key points. Be concise and informative. " +
                     "Do not use markdown formatting. Content:\n\n" + content;

        var summary = ai.generate(prompt);

        if (!summary) {
            console.warn("AI failed to generate summary");
            return SKIP;
        }

        // Reply in thread with summary
        var domain = extractDomain(url);
        message.reply(":memo: *TL;DR* (" + domain + ")\n" + summary);

        return HANDLED;
    }
};
