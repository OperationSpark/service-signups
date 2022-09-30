package signup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendWelcome(t *testing.T) {
	t.Run("sends a 'Welcome Email' with the correct template variables", func(t *testing.T) {
		sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
		form := Signup{
			NameFirst:        "Henri",
			NameLast:         "Testaroni",
			Email:            "henri@email.com",
			Cell:             "555-123-4567",
			Referrer:         "instagram",
			ReferrerResponse: "",
			StartDateTime:    sessionStartDate,
			Cohort:           "is-mar-14-22-12pm",
		}

		domain := "test.notarealdomain.org"
		apiKey := "test-key"

		expectedFormFields := map[string]string{
			"to": form.Email,
			// mgSrv.defaultTemplate
			"template": "info-session-signup",
			// mgSrv.defaultSender
			"from":    "Operation Spark <admissions@test.notarealdomain.org>",
			"subject": "Welcome from Operation Spark!",
		}

		mockMailgunAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(128)

			// Check request has the correct fields
			for key, want := range expectedFormFields {
				got := r.FormValue(key)
				if got != want {
					w.WriteHeader(http.StatusExpectationFailed)
					t.Fatalf("expected Mailgun POST /messages form field %q:%q\nGot value: %q\n", key, want, got)
				}
			}
			// Check the template variables are correct
			jsonVars := r.Form.Get("h:X-Mailgun-Variables")
			var gotVars welcomeVariables
			json.Unmarshal([]byte(jsonVars), &gotVars)

			assertEqual(t, gotVars.FirstName, form.NameFirst)
			assertEqual(t, gotVars.LastName, form.NameLast)
			assertEqual(t, gotVars.SessionDate, "Monday, Mar 14")
			assertEqual(t, gotVars.SessionTime, "12:00 PM CDT")
			// TODO: ZoomURL
			// assertEqual(t, gotVars.ZoomURL, "TODO")

			w.Write([]byte("{}"))
		}))
		defer mockMailgunAPI.Close()

		mgSvc := NewMailgunService(domain, apiKey, mockMailgunAPI.URL+
			"/v4")

		err := mgSvc.sendWelcome(form)

		if err != nil {
			t.Fatalf("send welcome: %v", err)
		}
	})
}

func assertEqual(t *testing.T, got, want string) {
	if got != want {
		t.Fatalf("Want: %q, but got: %q", want, got)
	}
}
