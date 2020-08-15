package bot

type Config struct {
	Port         int    `default:"8080"`
	BotUserToken string `required:"true" split_words:"true"`
	UserToken    string `required:"true" split_words:"true"`
	Debug        bool
	Version      string
}
