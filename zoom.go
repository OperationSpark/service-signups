package signup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/operationspark/service-signup/zoom/meeting"
)

type (
	zoomService struct {
		// Base API endpoint. Default: "https://api.zoom.us/v2"
		baseURL string
		// Base OAuth endpoint. Default: "https://zoom.us/oauth"
		oauthURL string
		// HTTP client for making Zoom API requests.
		// https://marketplace.zoom.us/docs/api-reference/zoom-api/methods/#overview
		client         http.Client
		accessToken    string
		tokenExpiresAt time.Time
		accountID      string
		clientID       string
		clientSecret   string
	}

	tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}

	ZoomOptions struct {
		// Overrides Zoom API base URL for testing. Default: "https://api.zoom.us/v2"
		baseAPIOverride string
		// Overrides Zoom OAuth base URL for testing. Default: "https://zoom.us/oauth"
		baseOAuthOverride string
		clientID          string
		clientSecret      string
		accountID         string
	}
)

func NewZoomService(o ZoomOptions) *zoomService {
	apiURL := "https://api.zoom.us/v2"

	if len(o.baseAPIOverride) > 0 {
		apiURL = o.baseAPIOverride
	}

	oauthURL := "https://zoom.us/oauth"
	if len(o.baseOAuthOverride) > 0 {
		oauthURL = o.baseOAuthOverride
	}

	return &zoomService{
		baseURL:      apiURL,
		oauthURL:     oauthURL,
		client:       *http.DefaultClient,
		clientID:     o.clientID,
		clientSecret: o.clientSecret,
		accountID:    o.accountID,
	}
}

func (z *zoomService) run(su Signup) error {
	// Do nothing if the user has not signed up for a specific session
	if su.StartDateTime.IsZero() {
		return nil
	}
	return z.registerUser(su)
}

func (z *zoomService) name() string {
	return "zoom service"
}

// RegisterUser creates and submits a user's registration to a meeting. The specific meeting is decided from the Signup's startDateTime.
func (z *zoomService) registerUser(su Signup) error {
	// Authenticate client
	if !z.isAuthenticated() {
		if err := z.authenticate(); err != nil {
			return fmt.Errorf("authenticate: %v", err)
		}
	}

	// Send Zoom API req to register user to meeting
	reqBody := meeting.RegistrantRequest{
		FirstName: su.NameFirst,
		LastName:  su.NameLast,
		Email:     su.Email,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshall: %v", err)
	}

	// Register for a specific occurrence for the recurring meeting
	url := fmt.Sprintf(
		"%s/meetings/%d/registrants?occurrence_id=%d",
		z.baseURL,
		su.ZoomMeetingID(),
		su.StartDateTime.Unix()*int64(time.Microsecond),
	)

	req, err := http.NewRequestWithContext(
		context.TODO(),
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("newRequestWithContext: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", z.accessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}

	var respBody meeting.RegistrationResponse
	d := json.NewDecoder(resp.Body)
	d.Decode(&respBody)
	return nil
}

// Authenticate requests an access token and sets it along with the expiration date on the service.
func (z *zoomService) authenticate() error {
	url := fmt.Sprintf("%s/token?grant_type=account_credentials&account_id=%s", z.oauthURL, z.accountID)
	// Make a HTTP req to authenticate the client
	req, err := http.NewRequestWithContext(
		context.TODO(),
		http.MethodPost,
		url,
		nil,
	)
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+z.encodeCredentials())

	resp, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}

	var body tokenResponse
	d := json.NewDecoder(resp.Body)
	d.Decode(&body)

	z.accessToken = body.AccessToken
	z.tokenExpiresAt = time.Now().Add(time.Second * time.Duration(body.ExpiresIn))
	return nil
}

func (z *zoomService) isAuthenticated() bool {
	return len(z.accessToken) > 0 &&
		time.Now().Before(z.tokenExpiresAt.Truncate(time.Minute))
}

// EncodeCredentials base64 encodes the client ID and secret, separated by a colon.
// ie: Base64Encode([clientID]:[clientSecret])
func (z *zoomService) encodeCredentials() string {
	creds := fmt.Sprintf("%s:%s", z.clientID, z.clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(creds))
}