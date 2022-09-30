package signup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type greenlightService struct {
	url string // URL to POST webhooks.
}

func NewGreenlightService(url string) *greenlightService {
	return &greenlightService{
		url: url,
	}
}

func (g greenlightService) run(su Signup) error {
	return g.postWebhook(su)
}

func (g greenlightService) name() string {
	return "greenlight service"
}

// PostWebhook sends a webhook to Greenlight (POST /signup).
// The webhook creates a Info Session Signup record in the Greenlight database.
func (g greenlightService) postWebhook(su Signup) error {
	body, err := json.Marshal(su)
	if err != nil {
		return fmt.Errorf("greenlight postWebhook JSON marshall: %v", err)
	}

	resp, err := http.Post(g.url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("greenlight postWebhook POST: %v", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("greenlight postWebhook HTTP: %v", resp.Status)
	}
	return nil
}
