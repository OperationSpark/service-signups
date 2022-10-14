package signup

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendSMS(t *testing.T) {
	t.Run("sends an SMS message to the signed up user", func(t *testing.T) {
		accountSID := "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		authToken := "YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"
		fromPhoneNum := "+15041234567"

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

		sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
		su := Signup{
			NameFirst:        "Bob",
			NameLast:         "Ross",
			Email:            "bross@pbs.org",
			Cell:             "+19197654321",
			Referrer:         "instagram",
			ReferrerResponse: "",
			StartDateTime:    sessionStartDate,
			Cohort:           "is-mar-14-22-12pm",
			SessionId:        "X5TsABhN94yesyMEi",
		}

		err := tSvc.sendSMS(su)
		if err != nil {
			t.Fatalf("twilio service: sendSMS: %v", err)
		}
	})
}
