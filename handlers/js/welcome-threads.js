// Welcome Threads Handler
// Automatically welcomes users when they start a new thread in specific channels

var handler = {
    name: "welcome-threads",
    description: "Welcomes users when they start a new thread in introduction channels",
    channels: ["introductions", "introduce-yourself", "new-members"],
    priority: 100,
    enabled: true,
    skipBots: true,
    skipThreads: true,

    handle: function(message) {
        message.react("wave");
        console.info("Added wave reaction to introduction from user:", message.user);
        return HANDLED;
    }
};
