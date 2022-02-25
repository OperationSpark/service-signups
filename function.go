package signups

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/schema"
	"github.com/operationspark/slack-session-signups/slack"
)

var SLACK_WEBHOOK_URL = os.Getenv("SLACK_WEBHOOK_URL")
var decoder = schema.NewDecoder()

// handleJson unmarshalls a JSON payload from a signUp request into a Signup.
func handleJson(w http.ResponseWriter, r *http.Request) (Signup, error) {
	b, err := io.ReadAll(r.Body)
	s := Signup{}
	if err != nil {
		return s, err
	}

	err = json.Unmarshal(b, &s)
	if err != nil {
		return s, err
	}

	return s, nil
}

// handleForm unmarshalls a FormData payload from a signUp request into a Signup
func handleForm(w http.ResponseWriter, r *http.Request) (Signup, error) {
	s := Signup{}
	err := r.ParseForm()
	if err != nil {
		return s, err
	}

	err = decoder.Decode(&s, r.PostForm)
	if err != nil {
		return s, err
	}

	return s, nil
}

// HandleSignUp parses Info Session sign up requests from operationspark.org.
// If successful, it sends webhooks to Greenlight, Slack, other services.
func HandleSignUp(w http.ResponseWriter, r *http.Request) {
	fmt.Println(SLACK_WEBHOOK_URL)
	slackMsg := slack.Message{}

	switch r.Header.Get("Content-Type") {
	case "application/json":
		s, err := handleJson(w, r)
		if err != nil {
			http.Error(w, "Error reading sign up body", http.StatusInternalServerError)
			panic(err)
		}
		slackMsg = slack.Message{Text: s.Summary()}

	case "application/x-www-form-urlencoded":
		s, err := handleForm(w, r)
		if err != nil {
			http.Error(w, "Error reading Form Body", http.StatusInternalServerError)
			panic(err)
		}
		fmt.Println(s)
		slackMsg = slack.Message{Text: s.Summary()}

	default:
		http.Error(w, "Unacceptable Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	err := slack.SendWebhook(SLACK_WEBHOOK_URL, slackMsg)
	if err != nil {
		http.Error(w, "Error sending Slack webhook", http.StatusInternalServerError)
		panic(err)
	}

}
