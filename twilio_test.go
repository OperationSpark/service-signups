package signup

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

		mockTwilioAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(128)
			assertNilError(t, err)

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, http.StatusBadRequest)
			fmt.Fprint(w, bytes.NewBufferString(`{
    "code": 50407,
    "message": "Invalid messaging binding address",
    "more_info": "https://www.twilio.com/docs/errors/50407",
    "status": 400
}`))

		}))

		tSvc := NewTwilioService(twilioServiceOptions{
			accountSID:   accountSID,
			authToken:    authToken,
			fromPhoneNum: fromPhoneNum,
			apiBase:      mockTwilioAPI.URL,
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

func TestInvalidNumErr(t *testing.T) {
	// test that the error is returned when the number is invalid
	t.Run("returns an error when the number is invalid", func(t *testing.T) {
		signup := Signup{
			NameFirst:        "Henri",
			NameLast:         "Testaroni",
			Email:            "henri@email.com",
			Cell:             "555-555-5555",
			Referrer:         "instagram",
			ReferrerResponse: "",
		}

		service := &MockSignupService{
			RegisterFunc: func(context.Context, Signup) error {
				// return invalid number error
				return ErrInvalidNumber{err: fmt.Errorf("invalid number: %s", signup.Cell)}
			},
		}

		server := &signupServer{service}

		req := httptest.NewRequest(http.MethodPost, "/", signupToJson(t, signup))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		server.HandleSignUp(res, req)

		// check that the response is a 400
		assertStatus(t, 400, http.StatusBadRequest)
		fmt.Print(res)
		// check that the response body is the expected error
		assertEqual(t, res.Body.String(), `400{Invalid Phone Number phone}`)

	})
}
