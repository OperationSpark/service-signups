package signups

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

// SignUp (verb) sends a webhook to Greenlight (POST /signup).
// The webhook creates a Info Session Signup record in the Greenlight database.
func (s *Signup) SignUp() error {
	url, ok := os.LookupEnv("GREENLIGHT_WEBHOOK_URL")
	if !ok {
		return errors.New("'GREENLIGHT_WEBHOOK_URL' env var not set")
	}

	body, err := json.Marshal(s)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("could not post to Greenlight\n%s", resp.Status)
	}
	return nil
}
