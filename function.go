package signup

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

func handleJson(w http.ResponseWriter, r *http.Request) (SignUp, error) {
	b, err := io.ReadAll(r.Body)
	s := SignUp{}
	if err != nil {
		return s, err
	}

	err = json.Unmarshal(b, &s)
	if err != nil {
		return s, err
	}

	return s, nil
}

func handleForm(w http.ResponseWriter, r *http.Request) (SignUp, error) {
	s := SignUp{}
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

// HandleSignUp handles Info Session sign up requests from operationspark.org
func HandleSignUp(w http.ResponseWriter, r *http.Request) {
	fmt.Println(SLACK_WEBHOOK_URL)

	ct := r.Header.Get("Content-Type")
	if ct == "application/json" {
		s, err := handleJson(w, r)
		if err != nil {
			http.Error(w, "Error reading sign up body", http.StatusInternalServerError)
			panic(err)
		}

		payload := slack.Message{Text: s.Summary()}
		err = slack.SendWebhook(SLACK_WEBHOOK_URL, payload)
		if err != nil {
			http.Error(w, "Error sending Slack webhook", http.StatusInternalServerError)
			panic(err)
		}
		return
	}

	if ct == "application/x-www-form-urlencoded" {
		s, err := handleForm(w, r)
		if err != nil {
			http.Error(w, "Error reading Form Body", http.StatusInternalServerError)
			panic(err)
		}
		fmt.Println(s)
		return
	}
	http.Error(w, "Unacceptable Content-Type", http.StatusUnsupportedMediaType)
}
