package signup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type slackService struct {
	url string // Slack Webhook URL.
}

func (sl slackService) run(su Signup) error {
	return sendWebhook(sl.url, message{Text: su.Summary()})
}

func (sl slackService) name() string {
	return "slack service"
}

func NewSlackService(url string) *slackService {
	return &slackService{url}
}

type message struct {
	Text string `json:"text"`
}

// SendWebhook POSTs a message to the OS Signups Slack App webhook.
// This incoming webhook posts a message to the #signups channel.
// https://api.slack.com/apps/A0338E8UFFV/incoming-webhooks
func sendWebhook(url string, msg message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("error sending Slack message: %s", resp.Status)
	}

	return nil
}
