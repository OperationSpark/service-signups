package signup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/operationspark/service-signup/zoom/meeting"
)

type (
	zoomService struct {
		baseURL        string // Base API endpoint. Default: "https://api.zoom.us/v2"
		oauthURL       string // Base OAuth endpoint. Default: "https://zoom.us/oauth"
		client         http.Client
		accessToken    string
		tokenExpiresAt time.Time
		accountID      string
		clientID       string
		clientSecret   string
		//
		meetings map[int]string
	}

	tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}

	ZoomOptions struct {
		baseAPIOverride   string //
		baseOAuthOverride string
		clientID          string
		clientSecret      string
		accountID         string
		meetings          map[int]string
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
		meetings:     o.meetings,
	}
}

func (z *zoomService) run(su Signup) error {
	return z.registerUser(su)
}

func (z *zoomService) name() string {
	return "zoom service"
}

func (z *zoomService) registerUser(su Signup) error {
	// Get Meeting ID based on Info Session time
	meetingID, err := z.getMeetingID(su)
	if err != nil {
		return fmt.Errorf("getMeetingID: %v", err)
	}

	// Authenticate client
	if !z.isAuthenticated() {
		if err = z.authenticate(); err != nil {
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

	url := fmt.Sprintf("%s/meetings/%d/registrants", z.baseURL, meetingID)

	req, err := http.NewRequestWithContext(
		context.TODO(),
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("newRequestWithContext: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer: %s", z.accessToken))
	resp, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		return fmt.Errorf("HTTP: %s", resp.Status)
	}
	var respBody meeting.RegistrationResponse
	d := json.NewDecoder(resp.Body)
	d.Decode(&respBody)

	return nil
}

func (z *zoomService) getMeetingID(su Signup) (int64, error) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return 0, fmt.Errorf("loadLocation: %v", err)
	}
	sessionStart := su.StartDateTime
	centralStart := sessionStart.In(loc)

	if _, ok := z.meetings[centralStart.Hour()]; !ok {
		return 0, fmt.Errorf("no zoom meeting found with start hour: %d", centralStart.Hour())
	}
	id, err := strconv.Atoi(z.meetings[centralStart.Hour()])
	if err != nil {
		return 0, fmt.Errorf("convert string to intL %v", err)
	}
	return int64(id), nil
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
		return fmt.Errorf("NewRequestWithContext: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+z.encodeCredentials())

	resp, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP: %v", resp.Status)
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

func (z *zoomService) encodeCredentials() string {
	creds := fmt.Sprintf("%s:%s", z.clientID, z.clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(creds))
}
