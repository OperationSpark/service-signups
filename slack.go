package signup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type slackService struct {
	// Slack Incoming Webhook URL.
	// https://hooks.slack.com/services/:workspaceID/:botID/:webhookID
	// Can be found on the App's Incoming Webhooks page.
	// https://api.slack.com/apps/A0338E8UFFV/incoming-webhooks?
	webhookURL string
}

func (sl slackService) run(su Signup) error {
	return sendWebhook(sl.webhookURL, message{Text: su.Summary()})
}

func (sl slackService) name() string {
	return "slack service"
}

func NewSlackService(webhookURL string) *slackService {
	return &slackService{webhookURL}
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
		return fmt.Errorf("marshall: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("post request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}

	return nil
}
