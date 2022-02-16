package main

import (
	"fmt"
	"os"

	"github.com/operationspark/slack-session-signups/slack"
)

var SLACK_WEBHOOK_URL = os.Getenv("SLACK_WEBHOOK_URL")

func main() {
	fmt.Println(SLACK_WEBHOOK_URL)

	// TODO: Get message data from webhook body
	msg := slack.Message{Text: "Hello"}

	err := slack.SendWebhook(SLACK_WEBHOOK_URL, msg)

	if err != nil {
		fmt.Print(err)
	}
}
