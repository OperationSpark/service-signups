package signup

import (
	"bytes"
	"context"
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

func (sl slackService) run(ctx context.Context, su Signup) error {
	return sendWebhook(ctx, sl.webhookURL, message{Text: su.Summary()})
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
func sendWebhook(ctx context.Context, url string, msg message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshall: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}

	return nil
}
