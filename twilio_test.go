package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Magic Test Numbers
// https://www.twilio.com/docs/iam/test-credentials#magic-input

func mustSetEnv(t *testing.T) {
	t.Helper()
	if err := os.Setenv("TWILIO_ACCOUNT_SID", "testAccountSID"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("TWILIO_AUTH_TOKEN", "testAuthToken"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("TWILIO_PHONE_NUMBER", "+15005550006"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("TWILIO_CONVERSATIONS_SID", "CHXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"); err != nil {
		t.Fatal(err)
	}
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
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			_, err = fmt.Fprint(w, http.StatusBadRequest)
			require.NoError(t, err)
			_, err = fmt.Fprint(w, bytes.NewBufferString(`
{
    "code": 50407,
    "message": "Invalid messaging binding address",
    "more_info": "https://www.twilio.com/docs/errors/50407",
    "status": 400
}
`[1:],
			))
			require.NoError(t, err)
		}))

		tSvc := NewTwilioService(twilioServiceOptions{
			accountSID:   accountSID,
			authToken:    authToken,
			fromPhoneNum: fromPhoneNum,
			apiBase:      mockTwilioAPI.URL,
		})

		messageBody := "Welcome to Op Spark! Click this link for more info: https://opsk.org/bh213v34fa"

		err := tSvc.sendSMSInConversation(messageBody, conversationSid)
		require.NoErrorf(t, err, "twilio service: sendSMS: %v")
	})

}

func TestFindConversationsByNumber(t *testing.T) {
	t.Skip("TODO")

	t.Run("finds all the conversation for the given phone number", func(t *testing.T) {
		mustSetEnv(t)

		tSvc := NewTwilioService(twilioServiceOptions{
			accountSID:   os.Getenv("TWILIO_ACCOUNT_SID"),
			authToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
			fromPhoneNum: os.Getenv("TWILIO_PHONE_NUMBER"),
		})

		_, err := tSvc.findConversationsByNumber("+15005550006")
		require.NoError(t, err)

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

		err := tSvc.run(context.Background(), &su, slog.Default())
		require.NoError(t, err)

		require.NotEmpty(t, su.conversationID)
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
			RegisterFunc: func(ctx context.Context, su Signup) (Signup, error) {
				// return invalid number error
				return su, ErrInvalidNumber{err: fmt.Errorf("invalid number: %s", signup.Cell)}
			},
		}

		server := &signupServer{
			service: service,
			logger:  slog.Default(),
		}

		req := httptest.NewRequest(http.MethodPost, "/", signupToJSON(t, signup))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		server.HandleSignUp(res, req)

		// check that the response is a 400
		require.Equal(t, http.StatusBadRequest, res.Code)
		// check that the response body is the expected error
		var errResp badReqBodyResp

		err := json.Unmarshal([]byte(res.Body.Bytes()), &errResp)
		require.NoError(t, err)
		want := fmt.Sprintln(`{"message":"Invalid Phone Number","field":"phone"}`)
		require.Equal(t, want, res.Body.String())
	})
}
