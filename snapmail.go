package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/api/idtoken"
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

func NewSnapMail(url string) *SnapMail {
	client := http.DefaultClient
	var err error
	// In the CI environment, we don't have access to the GCP Service Account and we don't want to fail the build.
	// The only reason this is an issue is because we need to use init() to register the function with the functions-framework.
	// If we could avoid that, we could avoid this check.
	if os.Getenv("CI") != "" {
		client, err = idtoken.NewClient(context.Background(), url)
		if err != nil {
			log.Fatal(err)
		}
	}
	return &SnapMail{
		client: client,
		url:    url,
	}
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
