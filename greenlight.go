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

func (g greenlightService) run(ctx context.Context, su *Signup) error {
	return g.postWebhook(ctx, su)
}

// IsRequired return true because the signup record in Greenlight is needed by staff.
func (g greenlightService) isRequired() bool {
	return true
}

func (g greenlightService) name() string {
	return "greenlight service"
}

type signupResp struct {
	Status   string `json:"status"`
	SignupID string `json:"signupId"`
}

// PostWebhook sends a webhook to Greenlight (POST /signup).
// The webhook creates a Info Session Signup record in the Greenlight database.
func (g greenlightService) postWebhook(ctx context.Context, su *Signup) error {
	reqBody, err := json.Marshal(su)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.url,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("newRequest: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Greenlight-Signup-Api-Key", g.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}

	suResp := &signupResp{}
	if err := json.NewDecoder(resp.Body).Decode(suResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	su.id = &suResp.SignupID
	return nil
}
