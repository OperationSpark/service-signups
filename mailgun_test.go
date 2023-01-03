package signup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/operationspark/service-signup/greenlight"
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
			err := r.ParseMultipartForm(128)
			assertNilError(t, err)
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
			err = json.Unmarshal([]byte(jsonVars), &gotVars)
			assertNilError(t, err)

			assertEqual(t, gotVars.FirstName, form.NameFirst)
			assertEqual(t, gotVars.LastName, form.NameLast)
			assertEqual(t, gotVars.SessionDate, "Monday, Mar 14")
			assertEqual(t, gotVars.SessionTime, "12:00 PM CDT")
			// TODO: ZoomURL
			// assertEqual(t, gotVars.ZoomURL, "TODO")

			_, err = w.Write([]byte("{}"))
			assertNilError(t, err)
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
			err := r.ParseMultipartForm(128)
			assertNilError(t, err)

			assertEqual(t, r.FormValue("t:version"), "dev")

			_, err = w.Write([]byte("{}"))
			assertNilError(t, err)

		}))

		mgSvc := NewMailgunService(
			"mail.example.com",
			"api-key",
			mockMailgunAPI.URL+"/v4",
		)

		err := mgSvc.sendWelcome(context.Background(), Signup{})
		assertNilError(t, err)
	})

	t.Run("uses the 'info-session-signup-hybrid' template when 'hybrid' is true", func(t *testing.T) {

		signUp := Signup{
			LocationType: "HYBRID",
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

			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-12-05T18:00:00.000Z"),
		}

		mockMailgunAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(128)
			assertNilError(t, err)

			assertEqual(t, r.FormValue("template"), "info-session-signup-hybrid")

			// Check the template variables are correct
			jsonVars := r.Form.Get("h:X-Mailgun-Variables")
			var gotVars welcomeVariables
			err = json.Unmarshal([]byte(jsonVars), &gotVars)
			assertNilError(t, err)

			assertEqual(t, gotVars.LocationLine1, "514 Franklin Ave")
			assertEqual(t, gotVars.LocationCityStateZip, "New Orleans, LA 70117")
			assertEqual(t, gotVars.LocationMapURL, "https://www.google.com/maps/place/514+Franklin+Ave%2CNew+Orleans%2C+LA+70117")

			_, err = w.Write([]byte("{}"))
			assertNilError(t, err)

		}))

		mgSvc := NewMailgunService(
			"mail.example.com",
			"api-key",
			mockMailgunAPI.URL+"/v4",
		)

		err := mgSvc.sendWelcome(context.Background(), signUp)
		if err != nil {
			t.Fatal(err)
		}
	})

}
