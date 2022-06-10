# Candebot

[![Go Report Card](https://goreportcard.com/badge/github.com/bcneng/candebot?style=flat-square)](https://goreportcard.com/report/github.com/bcneng/candebot)

![634885518034_68b9e6615588d0cf48f6_512](https://user-images.githubusercontent.com/1083296/93481058-7108b880-f8fe-11ea-859e-5cb3b90927d3.jpg)

Our lovely opinionated slack bot. Find it in BcnEng slack as `@candebot`.

## Installation

```
go get -u github.com/bcneng/candebot
```

## Usage

```

CANDEBOT_BOT_USER_TOKEN=<slack-bot-user-token> \
CANDEBOT_SIGNING_SECRET=<slack-app-signing-secret> \
CANDEBOT_TWITTER_APIKEY=<twitter-api-key> \
CANDEBOT_TWITTER_APIKEYSECRET=<twitter-api-key-secret> \
CANDEBOT_TWITTER_CONTEST_TOKEN=<twitter-api-token> \
CANDEBOT_USER_TOKEN=<slack-user-token> \ 
candebot
```

You can get your bot user token by creating a Slack app via https://api.slack.com/apps.

## Deployment

TODO

