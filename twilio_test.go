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

	"github.com/operationspark/service-signup/greenlight"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)

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
		require.NoErrorf(t, err, "twilio service: sendSMS: %v")
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

		req := httptest.NewRequest(http.MethodPost, "/", signupToJson(t, signup))
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

func TestRunTwilioReal(t *testing.T) {
	t.Skip("Skipping TestRunTwilioReal for now.")
	// Ensure environment variables are set with real Twilio credentials
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	fromPhoneNum := os.Getenv("TWILIO_PHONE_NUMBER")
	conversationsSid := os.Getenv("TWILIO_CONVERSATIONS_SID")

	if accountSID == "" || authToken == "" || fromPhoneNum == "" || conversationsSid == "" {
		t.Skip("Twilio credentials are not set. Skipping real API test.")
	}

	// Set up the Twilio service with real credentials
	tSvc := NewTwilioService(twilioServiceOptions{
		accountSID:       accountSID,
		authToken:        authToken,
		fromPhoneNum:     fromPhoneNum,
		conversationsSid: conversationsSid,
	})

	// Create a sample Signup object
	// input the phone number, email, first name, and last name as needed for your test
	// For example, you can use a real phone number and email for testing
	su := returnSignup("", "", "", "", t)

	// Call the run method
	err := tSvc.run(context.Background(), &su, slog.Default())
	require.NoError(t, err)

	// Verify that the conversation ID was set
	require.NotEmpty(t, su.conversationID)
	t.Logf("Conversation ID: %s", *su.conversationID)
}

func returnSignup(phoneNumber, email, nameFirst, nameLast string, t *testing.T) Signup {
	// Create a sample Signup object
	return Signup{
		ProgramID:         "5sTmB97DzcqCwEZFR",
		NameFirst:         nameFirst,
		NameLast:          nameLast,
		Email:             email,
		Cell:              phoneNumber,
		Referrer:          "Word of mouth",
		ReferrerResponse:  "email blast",
		StartDateTime:     mustMakeTime(t, time.RFC3339, "2025-03-24T18:00:00.000Z"),
		Cohort:            "_dev_is-sep-28-23-12pm",
		SessionID:         "rBqAXvr8Zotw7JpSe",
		LocationType:      "HYBRID",
		UserLocation:      "Louisiana",
		AttendingLocation: "VIRTUAL",
		SMSOptIn:          true,
		GooglePlace: greenlight.GooglePlace{
			PlaceID: "ChIJ7YchCHSmIIYRYsAEPZN_E0o",
			Name:    "Operation Spark",
			Address: "514 Franklin Ave, New Orleans, LA 70117, USA",
			Phone:   "+1 504-534-8277",
			Website: "https://www.operationspark.org/",
			Geometry: greenlight.Geometry{
				Lat: 29.96325999999999,
				Lng: -90.052138,
			},
		},
	}
}
