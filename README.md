# sre-news-cli

Get today's news, filter important, add nasdaq opening (or holiday name) and send to slack.

## Example
![Slack](/slack.png)

## How to run

`export SRE_NEWS_API_URL=https://news…`

`export SRE_SLACK_WEBHOOK=https://hooks.slack.com/services/…`

`go run main.go`