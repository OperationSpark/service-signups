package signup

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/operationspark/service-signup/signing"
)

type SnapMail struct {
	url           string // Service HTTP endpoint
	client        *http.Client
	signingSecret []byte // Secret used to sign the request body
}

type Payload struct {
	Email         string    `json:"email"`
	NameFirst     string    `json:"nameFirst"`
	NameLast      string    `json:"nameLast"`
	SessionCohort string    `json:"sessionCohort"`
	SessionID     string    `json:"sessionId"`
	StartDateTime time.Time `json:"startDateTime,omitempty"`
	Mobile        string    `json:"mobile"`
}

type signupEvent struct {
	EventType string  `json:"eventType"`
	Payload   Payload `json:"payload"`
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

// IsRequired returns false because the snapMail webhook data can be retrieved from elsewhere and is not required for most students.
func (sm *SnapMail) isRequired() bool {
	return false
}

func (sm *SnapMail) run(ctx context.Context, signup *Signup, logger *slog.Logger) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	event := signupEvent{
		EventType: "SESSION_SIGNUP",
		Payload: Payload{
			Email:         signup.Email,
			NameFirst:     signup.NameFirst,
			NameLast:      signup.NameLast,
			SessionID:     signup.SessionID,
			SessionCohort: signup.Cohort,
			StartDateTime: signup.StartDateTime,
			Mobile:        signup.Cell,
		},
	}

	payload, err := json.Marshal(&event)
	if err != nil {
		return err
	}

	signature, err := signing.Sign(payload, sm.signingSecret, crypto.SHA256, signing.EncodingHex)
	if err != nil {
		return fmt.Errorf("createSignature: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sm.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("http.NewRequestWithContext: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", string(signature))

	resp, err := sm.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do: %w", err)
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

func WithSigningSecret(secret string) snapMailOption {
	return func(sm *SnapMail) {
		sm.signingSecret = []byte(secret)
	}
}
