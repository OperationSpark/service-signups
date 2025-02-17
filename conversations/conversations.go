package conversations

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/operationspark/service-signup/signing"
)

type Service struct {
	client           http.Client
	messengerAPIBase string
	signingSecret    []byte
}

type Option func(*Service)

func NewService(opts ...Option) *Service {
	s := &Service{
		client: http.Client{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// WithMessengerAPIBase sets the base URL for the Messenger API.
// Ex:
//
//	WithMessengerAPIBase("http://localhost:8080/api/v0")
func WithMessengerAPIBase(base string) Option {
	return func(s *Service) {
		s.messengerAPIBase = base
	}
}

// WithSigningSecret sets the secret key used to sign request bodies sent to the Messenger API.
func WithSigningSecret(token string) Option {
	return func(s *Service) {
		s.signingSecret = []byte(token)
	}
}

func (s Service) Run(ctx context.Context, conversationID, signupID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return s.linkConversation(ctx, conversationID, signupID)
}

func (s Service) Name() string {
	return "link-signup-conversation"
}

func (s Service) signPayload(payload []byte) ([]byte, error) {
	return signing.Sign(payload, s.signingSecret, crypto.SHA512)
}

type linkRequest struct {
	GLSignupID string `json:"greenlightSignupId"`
}

func (s Service) linkConversation(ctx context.Context, conversationID string, signupID string) error {
	body, err := json.Marshal(linkRequest{
		GLSignupID: signupID,
	})
	if err != nil {
		return fmt.Errorf("marshal link request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/conversations/%s/link", s.messengerAPIBase, conversationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create link request: %w", err)
	}

	signature, err := s.signPayload([]byte(body))
	if err != nil {
		return fmt.Errorf("sign payload: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-auth-signature-512", string(signature))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("messenger API link request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("messenger API link request: %s", resp.Status)
	}
	return nil
}
