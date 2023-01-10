package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	signup "github.com/operationspark/service-signup"
	"google.golang.org/api/idtoken"
)

type (
	smoke struct {
		// Greenlight API base. Used to fetch open Info Sessions.
		glAPIurl string
		// Selected Info Session to use for the sign up smoke test.
		selectedSession openSession
		// This services HTTP trigger URL.
		signupAPIurl string
		// Email address to used for the test signup.
		toEmail string
		// Twilio number that accepts test SMS messages.
		toNum string
	}

	openSession struct {
		ID           string `json:"_id"`
		LocationType string `json:"locationType"`
		Times        struct {
			Start struct {
				DateTime time.Time `json:"dateTime"`
			} `json:"start"`
		} `json:"times"`
	}
)

func main() {}

func newSmokeTest() *smoke {
	return &smoke{
		glAPIurl:     "https://greenlight.operationspark.org/api",
		signupAPIurl: "https://us-central1-operationspark-org.cloudfunctions.net/session-signups",
		toEmail:      os.Getenv("TEST_TO_EMAIL"),
		toNum:        os.Getenv("TEST_TO_NUM"),
	}
}

func (s *smoke) fetchInfoSessions() error {
	type response struct {
		Sessions []openSession `json:"sessions"`
	}
	resp, err := http.Get(s.glAPIurl + "/sessions/open?programId=5sTmB97DzcqCwEZFR&limit=1")
	if err != nil {
		return fmt.Errorf("GET: %w", err)
	}

	d := json.NewDecoder(resp.Body)
	var respBody response
	err = d.Decode(&respBody)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if resp.StatusCode >= 400 {
		signup.HandleHTTPError(resp)
		return fmt.Errorf("http error: %s", resp.Status)
	}

	if len(respBody.Sessions) == 0 {
		return errors.New("no open Info Sessions provided from Greenlight to select")
	}

	// Select next upcoming session
	s.selectedSession = respBody.Sessions[0]
	return nil
}

func (s *smoke) postSignup() error {
	su := signup.Signup{
		NameFirst:     "Halle",
		NameLast:      "Bot",
		Email:         s.toEmail,
		Cell:          s.toNum,
		SessionID:     s.selectedSession.ID,
		StartDateTime: s.selectedSession.Times.Start.DateTime,
	}
	var body bytes.Buffer
	e := json.NewEncoder(&body)
	err := e.Encode(&su)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// Use Google Auth to trigger cloud function
	req, err := makeAuthenticatedReq(http.MethodPost, s.signupAPIurl, &body)
	if err != nil {
		return fmt.Errorf("auth'd req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http POST: %w", err)
	}

	err = signup.HandleHTTPError(resp)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	return nil
}

// MakeAuthenticatedReq makes an HTTP request using Google Service Account credentials.
func makeAuthenticatedReq(method string, url string, body io.Reader) (*http.Request, error) {
	audience := url
	creds := os.Getenv("GCP_SA_CREDS_JSON")
	opts := idtoken.WithCredentialsJSON([]byte(creds))
	ts, err := idtoken.NewTokenSource(context.Background(), audience, opts)
	if err != nil {
		return nil, fmt.Errorf("newTokenSource: %w", err)
	}
	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("token: %w", err)
	}
	req, err := http.NewRequest(method, audience, body)
	token.SetAuthHeader(req)
	return req, err
}
