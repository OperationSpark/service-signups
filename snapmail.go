package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type SnapMail struct {
	url    string // Service HTTP endpoint
	client *http.Client
}

type signupEvent struct {
	Email         string    `json:"email"`
	NameFirst     string    `json:"nameFirst"`
	NameLast      string    `json:"nameLast"`
	SessionID     string    `json:"sessionId"`
	StartDateTime time.Time `json:"startDateTime,omitempty"`
}

func NewSnapMail(url string) *SnapMail {
	return &SnapMail{
		client: http.DefaultClient,
		url:    url,
	}
}

func (sm *SnapMail) name() string {
	return "SNAP Mailer"
}

func (sm *SnapMail) run(ctx context.Context, signup Signup) error {
	event := signupEvent{
		Email:         signup.Email,
		NameFirst:     signup.NameFirst,
		NameLast:      signup.NameLast,
		SessionID:     signup.SessionID,
		StartDateTime: signup.StartDateTime,
	}

	payload, err := json.Marshal(&event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sm.url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := sm.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}
	return nil
}
