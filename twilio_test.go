package signup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Magic Test Numbers
// https://www.twilio.com/docs/iam/test-credentials#magic-input

func setEnv() {
	os.Setenv("TWILIO_ACCOUNT_SID", "testAccountSID")
	os.Setenv("TWILIO_AUTH_TOKEN", "testAuthToken")
	os.Setenv("TWILIO_PHONE_NUMBER", "+15005550006")
	os.Setenv("TWILIO_CONVERSATIONS_SID", "CHXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
}

func TestSendSMSInConversation(t *testing.T) {
	t.Skip("TODO")

	t.Run("sends an SMS message through the Conversations API", func(t *testing.T) {
		accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
		authToken := os.Getenv("TWILIO_AUTH_TOKEN")
		fromPhoneNum := os.Getenv("TWILIO_PHONE_NUMBER")
		conversationSid := os.Getenv("TWILIO_CONVERSATIONS_SID")

		mockApi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(128)

			assertEqual(t, r.Form.Get("ShortenUrls"), "true")

			gotBody := r.Form.Get("Body")
			wantBody := "Mon Mar 14 @ 12:00p CDT"
			if !strings.Contains(gotBody, wantBody) {
				t.Fatalf("SMS body does not contain the correct Info Session data and time.\n\nBody:\n%s\n\n", gotBody)
				fmt.Fprint(w, http.StatusInternalServerError)
			}
		}))

		tSvc := NewTwilioService(twilioServiceOptions{
			accountSID:   accountSID,
			authToken:    authToken,
			fromPhoneNum: fromPhoneNum,
			apiBase:      mockApi.URL,
		})

		messageBody := "Welcome to Op Spark! Click this link for more info: https://opsk.org/bh213v34fa"

		err := tSvc.sendSMSInConversation(messageBody, conversationSid)
		if err != nil {
			t.Fatalf("twilio service: sendSMS: %v", err)
		}
	})
}

func TestFindConversationsByNumber(t *testing.T) {
	t.Skip("TODO")

	t.Run("finds all the conversation for the given phone number", func(t *testing.T) {
		setEnv()

		tSvc := NewTwilioService(twilioServiceOptions{
			accountSID:   os.Getenv("TWILIO_ACCOUNT_SID"),
			authToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
			fromPhoneNum: os.Getenv("TWILIO_PHONE_NUMBER"),
		})

		_, err := tSvc.findConversationsByNumber("+15005550006")
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(tSvc.apiBase)
	})
}

func TestTwilioRun(t *testing.T) {
	t.Skip("TODO")

	t.Run("creates a new conversation when sending new messages", func(t *testing.T) {
		tSvc := NewTwilioService(twilioServiceOptions{
			accountSID:       os.Getenv("TWILIO_ACCOUNT_SID"),
			authToken:        os.Getenv("TWILIO_AUTH_TOKEN"),
			fromPhoneNum:     os.Getenv("TWILIO_PHONE_NUMBER"),
			conversationsSid: os.Getenv("TWILIO_CONVERSATIONS_SID"),
		})

		su := Signup{
			NameFirst:     "Rick",
			NameLast:      "Sanchez",
			Cell:          "+15005550006",
			Cohort:        "is-oct-17-22-5-30pm",
			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-10-17T22:30:00.000Z"),
		}

		err := tSvc.run(context.Background(), su)
		if err != nil {
			t.Fatal(err)
		}

	})
}
