package signup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/operationspark/slack-session-signups/slack"
)

var SLACK_WEBHOOK_URL = os.Getenv("SLACK_WEBHOOK_URL")

// HandleSignUp handles Info Session sign up requests from operationspark.org
func HandleSignUp(w http.ResponseWriter, r *http.Request) {
	fmt.Println(SLACK_WEBHOOK_URL)

	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading sign up body", http.StatusInternalServerError)
		panic(err)
	}
	s := SignUp{}

	err = json.Unmarshal(b, &s)
	if err != nil {
		http.Error(w, "Error parsing JSON body", http.StatusInternalServerError)
		panic(err)
	}

	payload := slack.Message{Text: s.Summary()}

	err = slack.SendWebhook(SLACK_WEBHOOK_URL, payload)
	if err != nil {
		http.Error(w, "Error sending Slack webhook", http.StatusInternalServerError)
		panic(err)
	}

}
