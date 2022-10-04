package signup

import "testing"

func TestRegisterForMeeting(t *testing.T) {

}

func TestAuthHeader(t *testing.T) {
	t.Run("base64 encodes the client ID and secret", func(t *testing.T) {
		// These are NOT real credentials
		// Do NOT put real credentials here
		clientID := "jhasdbnca7843SHndd9324"
		clientSecret := "jdas87238hVSVDD9b2fe9nf2r2n8HJHV"

		want := "amhhc2RibmNhNzg0M1NIbmRkOTMyNDpqZGFzODcyMzhoVlNWREQ5YjJmZTluZjJyMm44SEpIVg=="

		zsvc := NewZoomService("", clientID, clientSecret)
		got := zsvc.encodeCredentials()

		assertEqual(t, got, want)
	})
}

func TestAuthenticate(t *testing.T) {
	t.Run("authorizes the client", func(t *testing.T) {
		zsvc := NewZoomService("", "", "")
		err := zsvc.authenticate()
		if err != nil {
			t.Fatalf("authenticate: %v", err)
		}

	})
}
