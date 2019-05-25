package main

import (
	"context"
	"log"

	"github.com/shomali11/slacker"

	"github.com/kelseyhightower/envconfig"
)

const version = "0.0.1-alpha"

type specification struct {
	BotUserToken string `required:"true" split_words:"true"`
	Debug        bool
}

func main() {
	var s specification
	err := envconfig.Process("candebot", &s)
	if err != nil {
		log.Fatal(err.Error())
	}

	bot := slacker.NewClient(s.BotUserToken, slacker.WithDebug(s.Debug))

	bot.DefaultCommand(func(request slacker.Request, response slacker.ResponseWriter) {
		response.Reply("Say what?, try typing `help` to see all the things I can do for you ;)")
	})

	bot.Command("coc", &slacker.CommandDefinition{
		Description: "Link to the Code Of Conduct of BcnEng",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply("Please find our Code Of Conduct here: https://bcneng.github.io/coc/")
		},
	})

	bot.Command("netiquette", &slacker.CommandDefinition{
		Description: "Link to the netiquette of BcnEng",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply("Please find our Netiquette here: https://bcneng.github.io/netiquette/")
		},
	})

	bot.Command("staff", &slacker.CommandDefinition{
		Description: "Info about the staff behind BcnEng",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			m := `
Here is the list of the current staff members: 

• Owners
    • <@gonzaloserrano>
    • <@smoya>
• Admins
    • <@mavi>
    • <@sdecandelario>
• Collaborators
    • <@UAG4H8GMD>
    • <@U7PQZMZ4L>
`

			response.Reply(m)
		},
	})

	bot.Command("version", &slacker.CommandDefinition{
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply("`" + version + "`")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
