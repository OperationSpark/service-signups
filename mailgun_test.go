package signup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

			// Check template version is not set
			assertEqual(t, r.FormValue("t:version"), "")

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

		mgSvc := NewMailgunService(domain, apiKey, mockMailgunAPI.URL+"/v4")

		err := mgSvc.sendWelcome(context.Background(), form)

		if err != nil {
			t.Fatalf("send welcome: %v", err)
		}
	})

	t.Run("uses 'dev' info-session-signup template 'APP_ENV' == 'staging' ", func(t *testing.T) {
		os.Setenv("APP_ENV", "staging")

		mockMailgunAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(128)

			assertEqual(t, r.FormValue("t:version"), "dev")

			w.Write([]byte("{}"))
		}))

		mgSvc := NewMailgunService(
			"mail.example.com",
			"api-key",
			mockMailgunAPI.URL+"/v4",
		)

		err := mgSvc.sendWelcome(context.Background(), Signup{})
		if err != nil {
			t.Fatal(err)
		}

	})

	t.Run("uses the 'info-session-signup-hybrid' template when 'hybrid' is true", func(t *testing.T) {
		mockMailgunAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(128)

			assertEqual(t, r.FormValue("template"), "info-session-signup-hybrid")

			w.Write([]byte("{}"))
		}))

		mgSvc := NewMailgunService(
			"mail.example.com",
			"api-key",
			mockMailgunAPI.URL+"/v4",
		)

		err := mgSvc.sendWelcome(context.Background(), Signup{LocationType: "HYBRID"})
		if err != nil {
			t.Fatal(err)
		}
	})

}
