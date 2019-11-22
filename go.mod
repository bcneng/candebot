module github.com/bcneng/candebot

go 1.12

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/nlopes/slack v0.6.1-0.20191106133607-d06c2a2b3249
	github.com/shomali11/commander v0.0.0-20190608032441-141478e8c069 // indirect
	github.com/shomali11/proper v0.0.0-20190608032528-6e70a05688e7 // indirect
	github.com/shomali11/slacker v0.0.0-20190608032631-289f4e2732b6
)

replace github.com/shomali11/slacker => github.com/smoya/slacker v0.0.0-20190806212550-ff90171ac023
