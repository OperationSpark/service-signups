package signup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterForMeeting(t *testing.T) {

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

	t.Run("authorizes the client", func(t *testing.T) {
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
				ExpiresIn:   3599,
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
	})
}
