package signup

import (
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
			Cell:             `json:"cell" schema:"cell"`,
			Referrer:         "instagram",
			ReferrerResponse: "",
			StartDateTime:    sessionStartDate,
			Cohort:           "is-mar-14-22-12pm",
			SessionId:        "X5TsABhN94yesyMEi",
		}
		mockGreenlightSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			// Sends API Key header
			assertEqual(t, r.Header.Get("X-Greenlight-Signup-Api-Key"), apiKey)

			// POSTs correct JSON body
			var resp Signup
			d := json.NewDecoder(r.Body)
			d.Decode(&resp)

			assertDeepEqual(t, resp, su)
		}))

		glSvc := NewGreenlightService(mockGreenlightSvr.URL, apiKey)

		glSvc.postWebhook(su)
	})
}
