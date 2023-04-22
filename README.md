# Candebot

[![Go Report Card](https://goreportcard.com/badge/github.com/bcneng/candebot?style=flat-square)](https://goreportcard.com/report/github.com/bcneng/candebot)

![634885518034_68b9e6615588d0cf48f6_512](https://user-images.githubusercontent.com/1083296/93481058-7108b880-f8fe-11ea-859e-5cb3b90927d3.jpg)

Our lovely opinionated Slack bot. Find it in [BcnEng Slack workspace](https://slack.bcneng.org) as `@candebot`.

## Features
- Commands emulating a command line tool. Via Slack slash (`/`) or mentioning the bot. See [/cmd](cmd). For example:
  - `coc` - Shows the Code of Conduct.
  - `netiquette` - Shows the Netiquette.
  - `staff` - Shows the list of staff members.
  - `echo` - Sending messages as the bot user. Only available to admins.
  - `candebirthday` - Days until [@sdecandelario](https://bcneng.slack.com/archives/D9BU155J9) birthday! Something people cares.
- Filter stopwords in messages. Suggest more inclusive alternatives to the user. See [/inclusion](inclusion).
- Submission and validation of job posts. Posted in the `#hiring-job-board` channel via a form.
- Message actions. For example:
  - Deleting a message and the whole thread. Only available to admins.
  - Report messages to the admins.

## Configuration
Candebot can be configured via Toml file + environment variables.
The need for setting up environment variables when using the Toml file is due to the fact that some of the configurations is sensitive and should not be stored in a file.

### Environment variables
All environment variables are prefixed with `BOT_`. For example, `BOT_BOT_USER_TOKEN`. If you want to change the prefix, you can set `-env-prefix <prefix>` flag when running the bot.

The following environment variables are needed in order to run the bot:

- `BOT_BOT_USER_TOKEN` - Slack bot user token. Used to authenticate the bot user.
- `BOT_BOT_ADMIN_TOKEN` - Slack user token with admin rights. Used to authenticate the bot user when performing admin actions.
- `BOT_BOT_SERVER_SIGNING_SECRET` - Slack app signing secret. Used to verify the authenticity of the requests.

There are more environment variables that can be set. Please, check [/bot/config.go](.bot/config.go).

### Toml File
By default, `./.bot.toml` is used as the configuration file. If you want to change the path, you can set `-config <filepath>` flag when running the bot.

Please, use the [following file](.bot.toml) as a reference.

## Installation

```
go get -u github.com/bcneng/candebot
```

## Usage

```
BOT_BOT_USER_TOKEN=<slack-bot-user-token> \
BOT_BOT_ADMIN_TOKEN=<slack-user-with-admin-rights-token> \ 
BOT_BOT_SERVER_SIGNING_SECRET=<slack-app-signing-secret> \

candebot
```

You can get your bot user token by creating a Slack app via https://api.slack.com/apps.

## Deployment

There is no preference for deployment. You can deploy it in any way you want. For example, using Docker.
The files required will always be:

- Compiled binary of the bot.
- `.bot.toml` file with the configuration.

