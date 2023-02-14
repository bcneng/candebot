package cmd

import (
	"log"
	"strings"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Coc           Coc           `cmd:"" help:"Link to the Code Of Conduct of BcnEng"`
	Netiquette    Netiquette    `cmd:"" help:"Link to the Netiquette of BcnEng"`
	Staff         Staff         `cmd:"" help:"Info about the staff behind BcnEng"`
	Version       Version       `cmd:"" help:"Info about the staff behind BcnEng"`
	Candebirthday CandeBirthday `cmd:"" help:"Days until @sdecandelario birthday!"`
	Echo          Echo          `cmd:"" help:"Sends a message from the bot user" placeholder:"echo #general Hi folks!"`
	Contest       Contest       `cmd:"" help:"Runs a contest on Twitter"`
	Help          Help          `cmd:""`
}

func NewCLI(name string, args []string, options ...kong.Option) (*CLI, *kong.Context, error) {
	defaultOpts := []kong.Option{
		kong.Name(strings.ToLower(name)),
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
