package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type SnapMail struct {
	url    string // Service HTTP endpoint
	client *http.Client
}

type Payload struct {
	Email         string    `json:"email"`
	NameFirst     string    `json:"nameFirst"`
	NameLast      string    `json:"nameLast"`
	SessionID     string    `json:"sessionId"`
	StartDateTime time.Time `json:"startDateTime,omitempty"`
}

type signupEvent struct {
	Type    string  `json:"eventType"`
	Payload Payload `json:"payload"`
}

type snapMailOption func(*SnapMail)

func NewSnapMail(apiBase string, opts ...snapMailOption) *SnapMail {
	endpoint, err := url.Parse(apiBase)
	if err != nil {
		log.Fatal(fmt.Errorf("SNAP mail URL parse: %v", err))
	}

	sm := &SnapMail{
		client: http.DefaultClient,
		url:    endpoint.JoinPath("/events").String(),
	}
	for _, opt := range opts {
		opt(sm)
	}
	return sm
}

func (sm *SnapMail) name() string {
	return "SNAP Mailer"
}

func (sm *SnapMail) run(ctx context.Context, signup Signup) error {
	event := signupEvent{
		Type: "SESSION_SIGNUP",
		Payload: Payload{
			Email:         signup.Email,
			NameFirst:     signup.NameFirst,
			NameLast:      signup.NameLast,
			SessionID:     signup.SessionID,
			StartDateTime: signup.StartDateTime,
		},
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

func WithClient(client *http.Client) snapMailOption {
	return func(sm *SnapMail) {
		sm.client = client
	}
}
