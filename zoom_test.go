package signup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/operationspark/service-signup/zoom/meeting"
)

func TestRun(t *testing.T) {
	t.Run("does nothing if there is no session start time", func(t *testing.T) {
		mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("the Zoom API should not be called.")
		}))
		su := Signup{}
		zsvc := NewZoomService(ZoomOptions{
			baseAPIOverride: mockAPIServer.URL,
		})

		err := zsvc.run(su)
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})
}

func TestAuthHeader(t *testing.T) {
	t.Run("base64 encodes the client ID and secret", func(t *testing.T) {
		// These are NOT real credentials
		// Do NOT put real credentials here
		clientID := "jhasdbnca7843SHndd9324"
		clientSecret := "jdas87238hVSVDD9b2fe9nf2r2n8HJHV"

		want := "amhhc2RibmNhNzg0M1NIbmRkOTMyNDpqZGFzODcyMzhoVlNWREQ5YjJmZTluZjJyMm44SEpIVg=="

		zsvc := NewZoomService(ZoomOptions{
			clientID:     clientID,
			clientSecret: clientSecret,
		})
		got := zsvc.encodeCredentials()

		assertEqual(t, got, want)
	})
}

func TestAuthenticate(t *testing.T) {
	// These are NOT real credentials
	// Do NOT put real credentials here
	fakeClientID := "jhasdbnca7843SHndd9324"
	fakeClientSecret := "jdas87238hVSVDD9b2fe9nf2r2n8HJHV"
	fakeAccountID := "test-asfdd35345sger"
	fakeAccessToken := "nasdnadajdnkasd"

	// Pre calculated value
	// base64Encode(fakeClientID + ":" + fakeClientSecret)
	encodedCreds := "amhhc2RibmNhNzg0M1NIbmRkOTMyNDpqZGFzODcyMzhoVlNWREQ5YjJmZTluZjJyMm44SEpIVg=="

	expiresIn := 3599

	t.Run("authenticates the client", func(t *testing.T) {
		authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assertEqual(t, r.URL.Path, "/oauth/token")
			// Check the account ID is in the URL params
			id := r.URL.Query().Get("account_id")
			assertEqual(t, id, fakeAccountID)
			// Check the Authorization Header contains the client ID and secret
			authHeader := r.Header.Get("Authorization")
			assertEqual(t, authHeader, "Basic "+encodedCreds)

			w.WriteHeader(http.StatusOK)
			e := json.NewEncoder(w)
			body := tokenResponse{
				AccessToken: fakeAccessToken,
				ExpiresIn:   expiresIn,
				TokenType:   "bearer",
				Scope:       "meeting:master meeting:read:admin meeting:write:admin",
			}
			e.Encode(&body)
		}))

		zsvc := NewZoomService(ZoomOptions{
			baseOAuthOverride: authServer.URL + "/oauth",
			clientID:          fakeClientID,
			clientSecret:      fakeClientSecret,
			accountID:         fakeAccountID,
		})
		err := zsvc.authenticate()
		if err != nil {
			t.Fatalf("authenticate: %v", err)
		}

		assertEqual(t, zsvc.accessToken, fakeAccessToken)
		// token expiration date should be now() + expiresIn
		wantExpiry := time.Now().
			Add(time.Second * time.Duration(expiresIn)).
			// Round down to the nearest minute
			Truncate(time.Minute)

		gotExpiry := zsvc.tokenExpiresAt.Truncate(time.Minute)
		assertEqual(t, gotExpiry, wantExpiry)
	})
}

func TestIsAuthenticated(t *testing.T) {
	t.Run("returns false if the client has no token", func(t *testing.T) {
		zsvc := NewZoomService(ZoomOptions{})
		assertEqual(t, zsvc.isAuthenticated(), false)
	})

	t.Run("returns false if the client's token is expired", func(t *testing.T) {
		zsvc := NewZoomService(ZoomOptions{})
		zsvc.tokenExpiresAt = time.Now().Add(-time.Minute)

		assertEqual(t, zsvc.isAuthenticated(), false)
	})

	t.Run("returns true if the client has an unexpired token", func(t *testing.T) {
		zsvc := NewZoomService(ZoomOptions{})
		zsvc.accessToken = "an-access-token"
		zsvc.tokenExpiresAt = time.Now().Add(time.Minute * 1)

		assertEqual(t, zsvc.isAuthenticated(), true)
	})
}

func TestRegisterForMeeting(t *testing.T) {
	sessionStartDate, _ := time.Parse(time.RFC822, "17 Oct 22 22:30 UTC")
	su := Signup{
		NameFirst:     "Tamari",
		NameLast:      "Quanka",
		Email:         "t.quan@aol.com",
		StartDateTime: sessionStartDate,
	}

	mockMeetingID := int64(87582741258)
	su.SetZoomMeetingID(mockMeetingID)
	mockZoomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		assertEqual(t, r.Method, http.MethodPost)
		// Check auth token
		authHeader := r.Header.Get("Authorization")
		assertEqual(t, authHeader, "Bearer fake_access_token")

		assertEqual(t, r.URL.Path, fmt.Sprintf("/meetings/%d/registrants", mockMeetingID))
		// Meeting Occurrence ID. Provide this field to view meeting details of a particular occurrence of the recurring meeting.
		assertEqual(t, r.URL.Query().Get("occurrence_id"), "1666045800000")

		var reqBody meeting.RegistrantRequest

		d := json.NewDecoder(r.Body)
		d.Decode(&reqBody)

		assertEqual(t, reqBody.Email, su.Email)
		assertEqual(t, reqBody.FirstName, su.NameFirst)
		assertEqual(t, reqBody.LastName, su.NameLast)

		w.WriteHeader(http.StatusOK)
		e := json.NewEncoder(w)
		e.Encode(meeting.RegistrationResponse{
			JoinURL: fmt.Sprintf("https://us06web.zoom.us/j/%d", mockMeetingID),
		})
	}))

	zsvc := NewZoomService(ZoomOptions{
		baseAPIOverride: mockZoomServer.URL,
	})

	// Fake authentication
	zsvc.accessToken = "fake_access_token"
	zsvc.tokenExpiresAt = time.Now().Add(time.Minute * 10)

	// Method under test
	err := zsvc.registerUser(su)
	if err != nil {
		t.Fatalf("register for meeting: %v", err)
	}
}