package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Message struct {
	Text string `json:"text"`
}

// SendWebhook POSTs a message to the OS Signups Slack App webhook.
// This incoming webhook posts a message to the #signups channel.
// https://api.slack.com/apps/A0338E8UFFV/incoming-webhooks
func SendWebhook(url string, msg Message) error {
	if os.Getenv("DISABLE_SLACK") == "true" {
		return nil
	}
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
