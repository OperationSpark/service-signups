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
