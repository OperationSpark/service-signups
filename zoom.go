package signup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type zoomService struct {
	baseURL      string // Base API endpoint. Default: "https://api.zoom.us/v2"
	oauthURL     string // Base OAuth endpoint. Default: "https://zoom.us/oauth"
	client       http.Client
	accessToken  string
	accountID    string
	clientID     string
	clientSecret string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type ZoomOptions struct {
	baseAPIOverride   string //
	baseOAuthOverride string
	clientID          string
	clientSecret      string
	accountID         string
}

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
	return z.registerUser()
}

func (z *zoomService) name() string {
	return "zoom service"
}

func (z *zoomService) registerUser() error {
	// Authenticate client
	// Get Meeting ID based on Info Session time
	// Send Zoom API req to register user to meeting
	return nil
}

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
		return fmt.Errorf("%s: authenticate: %v", z.name(), err)
	}

	req.Header.Add("Authorization", "Basic "+z.encodeCredentials())

	resp, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: Do: %v", z.name(), err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: authenticate: HTTP: %v", z.name(), resp.Status)
	}

	var body tokenResponse
	d := json.NewDecoder(resp.Body)
	d.Decode(&body)

	z.accessToken = body.AccessToken
	return nil
}

func (z *zoomService) encodeCredentials() string {
	creds := fmt.Sprintf("%s:%s", z.clientID, z.clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(creds))
}
