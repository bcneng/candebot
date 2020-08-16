module github.com/bcneng/candebot

go 1.14

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/shomali11/proper v0.0.0-20190608032528-6e70a05688e7 // indirect
	github.com/shomali11/slacker v0.0.0-20200610181250-3156f073f291
	github.com/slack-go/slack v0.6.5
	github.com/stretchr/testify v1.3.0
	golang.org/x/text v0.3.3
)

replace github.com/shomali11/slacker => github.com/smoya/slacker v0.0.0-20200728103316-563cbd9d3c10
