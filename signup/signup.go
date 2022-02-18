package signup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/operationspark/slack-session-signups/slack"
)

var SLACK_WEBHOOK_URL = os.Getenv("SLACK_WEBHOOK_URL")

type Session struct {
	Id string `json:"id"`
}

type Referral struct {
	Value          string `json:"value"`
	AdditionalInfo string `json:"additionalInfo"`
}

type SignUp struct {
	Session      Session  `json:"session"`
	Email        string   `json:"email"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	Phone        string   `json:"phone"`
	ReferencedBy Referral `json:"referencedBy"`
}

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

	msg := strings.Join([]string{
		fmt.Sprintf("%s %s has signed up for %s", s.FirstName, s.LastName, s.Session.Id),
		fmt.Sprintf("Ph: %s)", s.Phone),
		fmt.Sprintf("email: %s)", s.Email),
	}, "\n")
	payload := slack.Message{Text: msg}

	err = slack.SendWebhook(SLACK_WEBHOOK_URL, payload)
	if err != nil {
		http.Error(w, "Error sending Slack webhook", http.StatusInternalServerError)
		panic(err)
	}

}
