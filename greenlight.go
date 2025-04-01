package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/operationspark/service-signup/greenlight"
)

type greenlightService struct {
	url    string // URL to POST webhooks.
	apiKey string // Token to make Greenlight API requests.
}

func NewGreenlightService(url, apiKey string) *greenlightService {
	return &greenlightService{
		url:    url,
		apiKey: apiKey,
	}
}

func (g greenlightService) run(ctx context.Context, su *Signup, logger *slog.Logger) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return g.postWebhook(ctx, su)
}

// IsRequired return true because the signup record in Greenlight is needed by staff.
func (g greenlightService) isRequired() bool {
	return true
}

func (g greenlightService) name() string {
	return "greenlight service"
}

type signupResp struct {
	Status   string `json:"status"`
	SignupID string `json:"signupId"`
}

// signupReq is the request body for the Greenlight signup webhook.
type signupReq struct {
	// Wether the person is attending "IN_PERSON" | "VIRTUAL"ly.
	// This field is selected by the user on the website Sign Up form.
	AttendingLocation string `json:"attendingLocation" schema:"attendingLocation"`
	// The person's phone number.
	Cell string `json:"cell" schema:"cell"`
	// The session cohort the person is signing up for. Ex: "is-feb-28-22-12pm".
	Cohort string `json:"cohort" schema:"cohort"`
	// The person's email address.
	Email string `json:"email" schema:"email"`
	// The session's location's Google Place details.
	GooglePlace greenlight.GooglePlace `json:"googlePlace" schema:"googlePlace"`
	// Session's set location type. One of "IN_PERSON" | "VIRTUAL" | "IN_PERSON". If the session's location type is "HYBRID", a student can attend "IN_PERSON" or "VIRTUAL"ly.
	LocationType string `json:"locationType" schema:"locationType"`
	// A legacy 4-character join code for a Greenlight session.
	JoinCode         string `json:"joinCode,omitempty"`
	NameFirst        string `json:"nameFirst" schema:"nameFirst"`
	NameLast         string `json:"nameLast" schema:"nameLast"`
	ProgramID        string `json:"programId" schema:"programId"`
	Referrer         string `json:"referrer" schema:"referrer"`
	ReferrerResponse string `json:"referrerResponse" schema:"referrerResponse"`
	SessionID        string `json:"sessionId" schema:"sessionId"`
	// If the user has opted-out of receiving text messages.
	SMSOptOut     bool      `json:"smsOptOut"`
	StartDateTime time.Time `json:"startDateTime,omitempty" schema:"startDateTime"`
	Token         string    `json:"token" schema:"token"`
	// State or country where the person resides.
	UserLocation string `json:"userLocation" schema:"userLocation"`
}

// PostWebhook sends a webhook to Greenlight (POST /signup).
// The webhook creates a Info Session Signup record in the Greenlight database.
func (g greenlightService) postWebhook(ctx context.Context, su *Signup) error {
	signupReq := signupReq{
		AttendingLocation: su.AttendingLocation,
		Cell:              su.Cell,
		Cohort:            su.Cohort,
		Email:             su.Email,
		GooglePlace:       su.GooglePlace,
		LocationType:      su.LocationType,
		JoinCode:          su.userJoinCode,
		NameFirst:         su.NameFirst,
		NameLast:          su.NameLast,
		ProgramID:         su.ProgramID,
		Referrer:          su.Referrer,
		ReferrerResponse:  su.ReferrerResponse,
		SessionID:         su.SessionID,
		SMSOptOut:         !su.SMSOptIn,
		StartDateTime:     su.StartDateTime,
		Token:             su.Token,
		UserLocation:      su.UserLocation,
	}

	reqBody, err := json.Marshal(signupReq)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.url,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("newRequest: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Greenlight-Signup-Api-Key", g.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}

	suResp := &signupResp{}
	if err := json.NewDecoder(resp.Body).Decode(suResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	su.id = &suResp.SignupID
	return nil
}
