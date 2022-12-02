package signup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPostWebhook(t *testing.T) {

	t.Run("POSTs a webhook to Greenlight with the signup payload", func(t *testing.T) {
		apiKey := "test-api-key"

		sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
		su := Signup{
			NameFirst:        "Bob",
			NameLast:         "Ross",
			Email:            "bross@pbs.org",
			Cell:             "555-123-4567",
			Referrer:         "instagram",
			ReferrerResponse: "",
			StartDateTime:    sessionStartDate,
			Cohort:           "is-mar-14-22-12pm",
			SessionID:        "X5TsABhN94yesyMEi",
			UserLocation:     "Louisiana",
		}
		mockGreenlightSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sends API Key header
			assertEqual(t, r.Header.Get("X-Greenlight-Signup-Api-Key"), apiKey)

			// POSTs correct JSON body
			var glReq Signup
			d := json.NewDecoder(r.Body)
			d.Decode(&glReq)

			assertDeepEqual(t, glReq, su)
		}))

		glSvc := NewGreenlightService(mockGreenlightSvr.URL, apiKey)

		glSvc.postWebhook(context.Background(), su)
	})
}
