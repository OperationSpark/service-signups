package signup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type zoomService struct {
	baseURL      string // Base API Endpoint. Ex: "https://api.zoom.us/v2"
	meetings     map[string]string
	client       http.Client
	accessToken  string
	accountID    string
	clientID     string
	clientSecret string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

func NewZoomService(baseURL, clientID, clientSecret string) *zoomService {
	apiURL := "https://api.zoom.us/v2"
	if len(baseURL) > 0 {
		apiURL = baseURL
	}

	return &zoomService{
		baseURL:      apiURL,
		client:       *http.DefaultClient,
		clientID:     clientID,
		clientSecret: clientSecret,
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

	// Make a HTTP req to authenticate the client
	req, err := http.NewRequestWithContext(
		context.TODO(),
		http.MethodPost,
		fmt.Sprintf("https://zoom.us/oauth/token?grant_type=account_credentials&account_id=%s", z.accountID),
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
	fmt.Println(body)
	return nil
}

func (z *zoomService) encodeCredentials() string {
	creds := fmt.Sprintf("%s:%s", z.clientID, z.clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(creds))
}
