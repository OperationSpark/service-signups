package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	signup "github.com/operationspark/service-signup"
	"github.com/operationspark/service-signup/greenlight"
	"github.com/twilio/twilio-go"
	twiAPI "github.com/twilio/twilio-go/rest/api/v2010"
	"google.golang.org/api/idtoken"
)

type (
	smoke struct {
		// Greenlight API base. Used to fetch open Info Sessions.
		glAPIurl string
		// Selected Info Session to use for the sign up smoke test.
		selectedSession openSession
		// This services HTTP trigger URL.
		signupAPIurl string
		// Email address to used for the test signup.
		toEmail string
		// Twilio number that accepts test SMS messages.
		toNum string
		// Twilio number the signup service uses to sends SMS messages.
		fromNum string
	}

	openSession struct {
		ID           string                 `json:"_id"`
		Cohort       string                 `json:"cohort"`
		LocationType string                 `json:"locationType"`
		GooglePlace  greenlight.GooglePlace `json:"googlePlace"`
		Private      bool                   `json:"private"`
		ProgramID    string                 `json:"programId"`
		Times        struct {
			Start struct {
				DateTime time.Time `json:"dateTime"`
			} `json:"start"`
		} `json:"times"`
	}
)

func main() {}

func newSmokeTest() *smoke {
	return &smoke{
		glAPIurl:     "https://greenlight.operationspark.org/api",
		signupAPIurl: "https://us-central1-operationspark-org.cloudfunctions.net/session-signups",
		// TODO: Add "+randomString" suffix to avoid Zoom 3 registrants rate-limit
		toEmail: os.Getenv("TEST_TO_EMAIL"),
		toNum:   strings.TrimPrefix(os.Getenv("TEST_TO_NUM"), "+1"),
		fromNum: os.Getenv("TWILIO_PHONE_NUMBER"),
	}
}

func (s *smoke) fetchInfoSessions() error {
	type response struct {
		Sessions []openSession `json:"sessions"`
	}
	resp, err := http.Get(s.glAPIurl + "/sessions/open?programId=5sTmB97DzcqCwEZFR&limit=4")
	if err != nil {
		return fmt.Errorf("GET: %w", err)
	}

	d := json.NewDecoder(resp.Body)
	var respBody response
	err = d.Decode(&respBody)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	err = checkHTTPError(resp)
	if err != nil {
		return err
	}

	for _, session := range respBody.Sessions {
		// Select next upcoming session that isn't 'private' (test)
		if session.Private {
			continue
		}
		s.selectedSession = session
		return nil
	}
	return errors.New("no open Info Sessions provided from Greenlight to select")
}

func (s *smoke) postSignup(su signup.Signup) error {
	if os.Getenv("DRY_RUN") != "" {
		fmt.Printf("Signup: %+v", su)
		fmt.Printf("** Dry Run **\nSkipping Signup POST\n")
		return nil
	}

	var body bytes.Buffer
	e := json.NewEncoder(&body)
	err := e.Encode(&su)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// Use Google Auth to trigger cloud function
	req, err := makeAuthenticatedReq(http.MethodPost, s.signupAPIurl, &body)
	if err != nil {
		return fmt.Errorf("auth'd req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http POST: %w", err)
	}

	return checkHTTPError(resp)
}

// MakeAuthenticatedReq makes an HTTP request using Google Service Account credentials.
func makeAuthenticatedReq(method string, url string, body io.Reader) (*http.Request, error) {
	audience := url
	creds := os.Getenv("GCP_SA_CREDS_JSON")
	opts := idtoken.WithCredentialsJSON([]byte(creds))

	if creds == "" {
		opts = idtoken.WithCredentialsFile("../../creds.json")
	}
	ts, err := idtoken.NewTokenSource(context.Background(), audience, opts)
	if err != nil {
		return nil, fmt.Errorf("newTokenSource: %w", err)
	}
	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("token: %w", err)
	}
	req, err := http.NewRequest(method, audience, body)
	token.SetAuthHeader(req)
	return req, err
}

func fetchSMSmessage(toNum, fromNum string) (string, error) {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: os.Getenv("TWILIO_AUTH_TOKEN"),
	})

	params := &twiAPI.ListMessageParams{}
	params.SetPathAccountSid(accountSID)
	params.SetTo(toNum)
	params.SetFrom(fromNum)
	params.SetLimit(1)

	messages, err := client.Api.ListMessage(params)
	if err != nil {
		return "", fmt.Errorf("fetchMessage: %w", err)
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages found sent from %q -> %q", fromNum, toNum)
	}

	return *messages[0].Body, nil
}

// ParseSMSShortLink pulls a "ospk.org" short link out of a string.
func parseSMSShortLink(sms string) string {
	re := regexp.MustCompile(`https://ospk\.org/\w{10}`)
	match := re.FindString(sms)
	return match
}

// ParseSMSOriginal pulls an Info Session info link from an SMS body.
// This func is used to check if the URL shortener service has failed/
func parseSMSOriginalLink(sms string) string {
	// https://sms.operationspark.org/
	re := regexp.MustCompile(`https://sms\.operationspark\.org/m/\w+={1,2}`)
	match := re.FindString(sms)
	return match
}

// FetchLinkBody GETs a link and returns the body or error if HTTP response is 400+.
func fetchLinkBody(link string) (io.ReadCloser, error) {
	resp, err := http.Get(link)
	if err != nil {
		return resp.Body, fmt.Errorf("GET: %w", err)
	}

	err = checkHTTPError(resp)
	return resp.Body, err
}

func checkInfoPageContent(body io.Reader, wantStrings ...string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("readAll: %w", err)
	}

	for _, targetVal := range wantStrings {
		found := bytes.Contains(b, []byte(targetVal))
		if !found {
			htmlBody := extractHTMLBody(b)
			return fmt.Errorf("target value %q not found in Info Page body:\n%s", targetVal, string(htmlBody))
		}
	}
	return nil
}

// checkHTTPError checks if HTTP status abd returns an error if code is >= 400.
func checkHTTPError(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	reqLabel := fmt.Sprintf(
		"%s: %s://%s\n%s\n",
		resp.Request.Method,
		resp.Request.URL.Scheme,
		resp.Request.URL.Host,
		resp.Request.URL.RequestURI(),
	)

	errMsg := fmt.Sprintf("HTTP Error:\n%s\nResponse:\n%s", reqLabel, resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readAll: %w", err)
	}

	isHTML := strings.Contains(resp.Header.Get("Content-Type"), "text/html")
	if isHTML {
		body := extractHTMLBody(body)
		return fmt.Errorf("%s\n\n%s", errMsg, body)
	}

	return fmt.Errorf("%s\n%s", errMsg, body)
}

func extractHTMLBody(html []byte) []byte {
	re := regexp.MustCompile(`<body>(.+)</body>`)
	return re.Find(html)
}
