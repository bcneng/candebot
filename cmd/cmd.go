package cmd

import (
	"log"

	"github.com/alecthomas/kong"
	"github.com/slack-go/slack"
)

type SlackContext struct {
	User            string
	Channel         string
	Text            string
	Timestamp       string
	ThreadTimestamp string
}

type BotContext struct {
	Client       *slack.Client
	AdminClient  *slack.Client
	StaffMembers []string
	Version      string

	CLI bool // true if runs from CLI

	staffLookupMap map[string]struct{}
}

func (c *BotContext) IsStaff(userID string) bool {
	if c.staffLookupMap == nil {
		c.staffLookupMap = make(map[string]struct{}, len(c.StaffMembers)) // It is fine to not lock.
		for _, u := range c.StaffMembers {
			c.staffLookupMap[u] = struct{}{}
		}

	}

	_, ok := c.staffLookupMap[userID]

	return ok
}

type CLI struct {
	Coc           Coc           `cmd help:"Link to the Code Of Conduct of BcnEng"`
	Netiquette    Netiquette    `cmd help:"Link to the Netiquette of BcnEng"`
	Staff         Staff         `cmd help:"Info about the staff behind BcnEng"`
	Version       Version       `cmd help:"Info about the staff behind BcnEng"`
	Candebirthday CandeBirthday `cmd help:"Days until @sdecandelario birthday!"`
	Echo          Echo          `cmd help:"Sends a message as Candebot" placeholder:"echo #general Hi folks!"`
	Help          Help          `cmd`
}

func NewCLI(args []string, options ...kong.Option) (*CLI, *kong.Context, error) {
	defaultOpts := []kong.Option{
		kong.Name("candebot"),
		kong.ConfigureHelp(kong.HelpOptions{
			NoAppSummary: true,
			Summary:      false,
			Tree:         false,
			Compact:      true,
		}),
		kong.Exit(func(_ int) {}), // Override exit func to do just nothing
	}

	cli := new(CLI)
	parser, err := kong.New(cli, append(defaultOpts, options...)...)
	if err != nil {
		return nil, nil, err
	}

	kongCli, err := parser.Parse(args)
	if err != nil {
		if len(args) == 0 || len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
			// for help on the main app, do not print error nor usage because it's already handled by Kong.
			return nil, nil, err
		}
		log.Println("Error parsing kong command: ", err.Error())
		parser.Errorf("%s", err)
		if err, ok := err.(*kong.ParseError); ok {
			_ = err.Context.PrintUsage(false)
		}

		return nil, nil, err
	}

	return cli, kongCli, nil
}
