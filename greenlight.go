package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type greenlightService struct {
	url    string // URL to POST webhooks.
	apiKey string // Token to make Greenlight API requests.
}

func NewGreenlightService(url, apiKey string) *greenlightService {
	return &greenlightService{
		url:    url,
		apiKey: apiKey,
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

	req, err := http.NewRequestWithContext(
		context.TODO(),
		http.MethodPost,
		g.url,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("greenlight newRequest: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Greenlight-Signup-Api-Key", g.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("greenlight postWebhook POST: %v", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("greenlight postWebhook HTTP: %v", resp.Status)
	}
	return nil
}
