package signup

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/operationspark/service-signup/zoom/meeting"
	"github.com/stretchr/testify/require"
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

		err := zsvc.run(context.Background(), &su, slog.Default())
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

		require.Equal(t, want, got)
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
			require.Equal(t, "/oauth/token", r.URL.Path)
			// Check the account ID is in the URL params
			id := r.URL.Query().Get("account_id")
			require.Equal(t, fakeAccountID, id)
			// Check the Authorization Header contains the client ID and secret
			authHeader := r.Header.Get("Authorization")
			require.Equal(t, "Basic "+encodedCreds, authHeader)

			w.WriteHeader(http.StatusOK)
			e := json.NewEncoder(w)
			body := tokenResponse{
				AccessToken: fakeAccessToken,
				ExpiresIn:   expiresIn,
				TokenType:   "bearer",
				Scope:       "meeting:master meeting:read:admin meeting:write:admin",
			}
			err := e.Encode(&body)
			require.NoError(t, err)
		}))

		zsvc := NewZoomService(ZoomOptions{
			baseOAuthOverride: authServer.URL + "/oauth",
			clientID:          fakeClientID,
			clientSecret:      fakeClientSecret,
			accountID:         fakeAccountID,
		})

		token, err := zsvc.authenticate(context.Background())
		if err != nil {
			t.Fatalf("authenticate: %v", err)
		}

		require.Equal(t, fakeAccessToken, token.AccessToken)
		// token expiration date should be now() + expiresIn
		wantExpiry := time.Now().
			Add(time.Second * time.Duration(expiresIn)).
			// Round down to the nearest minute
			Truncate(time.Minute)

		gotExpiry := token.ExpiresAt.Truncate(time.Minute)
		require.Equal(t, wantExpiry, gotExpiry)
	})
}

func TestIsAuthenticated(t *testing.T) {
	t.Run("returns false if the client has no token", func(t *testing.T) {
		zsvc := NewZoomService(ZoomOptions{})
		require.Equal(t, false, zsvc.isAuthenticated(tokenResponse{}))
	})

	t.Run("returns false if the client's token is expired", func(t *testing.T) {
		zsvc := NewZoomService(ZoomOptions{})
		tokenExpiresAt := time.Now().Add(-time.Minute)

		require.Equal(t, false, zsvc.isAuthenticated(tokenResponse{ExpiresAt: tokenExpiresAt}))
	})

	t.Run("returns true if the client has an unexpired token", func(t *testing.T) {
		zsvc := NewZoomService(ZoomOptions{})

		require.Equal(t, true, zsvc.isAuthenticated(tokenResponse{
			AccessToken: "an-access-token",
			ExpiresAt:   time.Now().Add(time.Minute * 10),
		}))
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
	// Simulate the registrant-specific Join URL
	// Regular Zoom links, (Ex: https://us06web.zoom.us/j/87582741258), will redirect an unauthorized user to the registration page, defeating the purpose of auto-registration.
	mockJoinURL := fmt.Sprintf("https://us06web.zoom.us/w/%d?tk=6ySWiEckpHMI15UYaou_2dkNdDxTHbx7LM8l73iT7rM.DQMAAAAUeoDxnxZ5HSAGdi4newfHJJB#NBDETFhraE1BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", mockMeetingID)
	accessToken := "fake_access_token"

	mockZoomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			e := json.NewEncoder(w)
			err := e.Encode(tokenResponse{
				AccessToken: accessToken,
				ExpiresIn:   3600, // one hour in secs
			})
			assertNilError(t, err)
			return
		}

		if strings.Contains(r.URL.Path, "/meetings") {
			require.Equal(t, http.MethodPost, r.Method)
			// Check auth token
			authHeader := r.Header.Get("Authorization")
			require.Equal(t, "Bearer fake_access_token", authHeader)

			require.Equal(t, fmt.Sprintf("/meetings/%d/registrants", mockMeetingID), r.URL.Path)
			// Meeting Occurrence ID. Provide this field to view meeting details of a particular occurrence of the recurring meeting.
			require.Equal(t, "1666045800000", r.URL.Query().Get("occurrence_id"))

			var reqBody meeting.RegistrantRequest

			d := json.NewDecoder(r.Body)
			err := d.Decode(&reqBody)
			require.NoError(t, err)

			require.Equal(t, su.Email, reqBody.Email)
			require.Equal(t, su.NameFirst, reqBody.FirstName)
			require.Equal(t, su.NameLast, reqBody.LastName)

			w.WriteHeader(http.StatusOK)
			e := json.NewEncoder(w)
			// Respond with the Join URL
			err = e.Encode(meeting.RegistrationResponse{
				JoinURL: mockJoinURL,
			})
			require.NoError(t, err)
			return
		}

		http.Error(w, fmt.Sprintf("invalid URL:\n%q", r.URL.Path), http.StatusMethodNotAllowed)
	}))

	zsvc := NewZoomService(ZoomOptions{
		baseAPIOverride:   mockZoomServer.URL,
		baseOAuthOverride: mockZoomServer.URL,
	})

	// Method under test
	err := zsvc.registerUser(context.Background(), &su)
	if err != nil {
		t.Fatalf("register for meeting: %v", err)
	}

	// Check for custom Join URL from Zoom
	require.Equal(t, mockJoinURL, su.ZoomMeetingURL())
}

func TestAuthRefresh(t *testing.T) {
	t.Skip("This test is for the auto refresh token implementation. Currently we're just fetching a new token on every Zoom request.")

	ogToken := tokenResponse{
		AccessToken: "original-invalid-token",
		ExpiresIn:   3600,
	}
	refreshedToken := tokenResponse{
		AccessToken: "this-is-a-new-token",
		ExpiresIn:   3600,
		TokenType:   "bearer",
	}

	meetingEndpointCalls := 0
	authEndpointCalls := 0

	mockZoomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Meeting Registration handler
		if strings.Contains(r.URL.Path, "/meetings") {
			meetingEndpointCalls++
			// First /meeting call
			if meetingEndpointCalls == 1 {
				gotToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				require.Equal(t, ogToken.AccessToken, gotToken)
				// Send Invalid access token error
				type zoomErrorResp struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}
				w.WriteHeader(http.StatusUnauthorized)
				e := json.NewEncoder(w)
				err := e.Encode(zoomErrorResp{
					Code:    124,
					Message: "Invalid access token.",
				})
				require.NoError(t, err)
				return
			}

			// Subsequent /meeting call(s) should have a new token
			gotToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			require.Equal(t, refreshedToken, gotToken)
			return
		}

		// Auth Handler
		if strings.Contains(r.URL.Path, "/token") {
			authEndpointCalls++
			e := json.NewEncoder(w)
			// First call -> respond with original token
			if authEndpointCalls == 1 {
				err := e.Encode(ogToken)
				require.NoError(t, err)
				return
			}
			// Respond with refreshed token on subsequent calls
			err := e.Encode(refreshedToken)
			require.NoError(t, err)
			return
		}
	}))

	zsvc := NewZoomService(ZoomOptions{
		baseAPIOverride:   mockZoomServer.URL,
		baseOAuthOverride: mockZoomServer.URL,
	})

	err := zsvc.registerUser(context.Background(), &Signup{})
	require.NoError(t, err)

}
