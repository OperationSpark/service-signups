package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	signup "github.com/operationspark/service-signup"
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
	err := e.Encode(su)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	// TODO: Use Google Auth to trigger cloud function
	resp, err := http.Post(s.signupAPIurl, "application/json", &body)
	if err != nil {
		return fmt.Errorf("http POST: %w", err)
	}

	err = signup.HandleHTTPError(resp)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	return nil
}
