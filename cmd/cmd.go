package cmd

import (
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
}

func NewCLI(args []string, options ...kong.Option) (*CLI, *kong.Context, error) {
	helpOpts := kong.HelpOptions{
		NoAppSummary: true,
		Summary:      true,
		Tree:         false,
		Compact:      true,
	}
	opts := []kong.Option{
		kong.Description("Candebot, your lovely opinionated slack bot. Ready to serve and protect."),
		kong.ConfigureHelp(helpOpts),
		kong.UsageOnError(),
	}

	opts = append(opts, options...)

	cli := new(CLI)
	parser, err := kong.New(cli, options...)
	if err != nil {
		return nil, nil, err
	}
	parser.Exit = func(_ int) {} // Override exit func to do just nothing

	kongCli, err := parser.Parse(args)
	if err != nil {
		if err, ok := err.(*kong.ParseError); ok {
			parser.FatalIfErrorf(kong.DefaultHelpPrinter(helpOpts, err.Context))
		}

		return nil, nil, err
	}

	return cli, kongCli, nil
}
