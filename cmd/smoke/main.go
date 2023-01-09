package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	signup "github.com/operationspark/service-signup"
)

type (
	smoke struct {
		// Greenlight API base. Used to fetch open Info Sessions.
		glAPIurl string
		// This services HTTP trigger URL.
		signupAPIurl string
		// Twilio number that accepts test SMS messages.
		toNum string
		// Selected Info Session to use for the sign up smoke test.
		selectedSession openSession
	}

	openSession struct {
		ID           string `json:"_id"`
		LocationType string `json:"locationType"`
	}
)

func main() {}

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
