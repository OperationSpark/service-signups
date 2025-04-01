package signup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPostWebhook(t *testing.T) {
	t.Run("POSTs a webhook to Greenlight with the signup payload", func(t *testing.T) {
		apiKey := "test-api-key"

		sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
		su := Signup{
			NameFirst:         "Bob",
			NameLast:          "Ross",
			Email:             "bross@pbs.org",
			Cell:              "555-123-4567",
			Referrer:          "instagram",
			ReferrerResponse:  "",
			StartDateTime:     sessionStartDate,
			Cohort:            "is-mar-14-22-12pm",
			SessionID:         "X5TsABhN94yesyMEi",
			UserLocation:      "Louisiana",
			AttendingLocation: "IN_PERSON",
			SMSOptIn:          true,
		}
		mockGreenlightSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sends API Key header
			require.Equal(t, apiKey, r.Header.Get("X-Greenlight-Signup-Api-Key"))

			// POSTs correct JSON body
			var glReq signupReq
			d := json.NewDecoder(r.Body)
			err := d.Decode(&glReq)
			require.NoError(t, err)

			require.Equal(t, "Bob", glReq.NameFirst)
			require.Equal(t, "Ross", glReq.NameLast)
			require.Equal(t, "bross@pbs.org", glReq.Email)
			require.Equal(t, "555-123-4567", glReq.Cell)
			require.Equal(t, "instagram", glReq.Referrer)
			require.Equal(t, "", glReq.ReferrerResponse)
			require.Equal(t, "is-mar-14-22-12pm", glReq.Cohort)
			require.Equal(t, "X5TsABhN94yesyMEi", glReq.SessionID)
			require.Equal(t, "Louisiana", glReq.UserLocation)
			require.Equal(t, "IN_PERSON", glReq.AttendingLocation)
			require.Equal(t, false, glReq.SMSOptOut)
			require.Equal(t, "2022-03-14T17:00:00Z", glReq.StartDateTime.Format(time.RFC3339))

			// Responds with a signup ID
			resp := signupResp{
				Status:   "success",
				SignupID: "a-new-signup-id-from-greenlight",
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))

		glSvc := NewGreenlightService(mockGreenlightSvr.URL, apiKey)

		err := glSvc.postWebhook(context.Background(), &su)
		require.NoError(t, err)

		require.Equal(t, "a-new-signup-id-from-greenlight", *su.id)
	})
}
