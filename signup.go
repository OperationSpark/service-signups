package signup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/operationspark/service-signup/greenlight"
	"github.com/operationspark/service-signup/notify"
	"golang.org/x/sync/errgroup"
)

type (
	Signup struct {
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
		// If the user has opted-in to receiving text messages.
		SMSOptIn      bool      `json:"smsOptIn"`
		StartDateTime time.Time `json:"startDateTime,omitempty" schema:"startDateTime"`
		Token         string    `json:"token" schema:"token"`
		// State or country where the person resides.
		UserLocation string `json:"userLocation" schema:"userLocation"`

		// URL linking the user to an post-signup information page.
		ShortLink string
		// A user specific join code for a Greenlight session.
		userJoinCode   string
		zoomMeetingID  int64
		zoomMeetingURL string
	}

	SignupAlias Signup

	SignupJSON struct {
		SignupAlias
		ZoomJoinURL string `json:"zoomJoinUrl"`
	}

	welcomeVariables struct {
		FirstName            string `json:"firstName"`            // Person's first name.
		LastName             string `json:"lastName"`             // Person's last name.
		SessionTime          string `json:"sessionTime"`          // Greenlight session start time. Ex: "12:00 PM CDT"
		SessionDate          string `json:"sessionDate"`          // Greenlight session start Date. Ex: "Monday, Mar 14"
		ZoomURL              string `json:"zoomURL"`              // Zoom meeting URL.
		LocationLine1        string `json:"locationLine1"`        // Greenlight session location address line.
		LocationCityStateZip string `json:"locationCityStateZip"` // Greenlight session location city, state, and postal code.
		LocationMapURL       string `json:"locationMapUrl"`       // Google Maps location URL.
		JoinCode             string `json:"joinCode,omitempty"`   // Greenlight session join code.
		IsGmail              bool   `json:"isGmail,omitempty"`    // True if the person used a Gmail email address.
		GreenlightEnrollURL  string `json:"greenlightEnrollUrl"`  // Greenlight auto-enrollment URL.
	}

	SignupService struct {
		// Key-value map with the Central Time meeting start hour (int) as the keys, and Zoom Meeting ID as the values.
		// Ex: {17: "86935241734"} denotes meeting with ID, "86935241734", starts at 5pm central.
		meetings    map[int]string
		tasks       []Task
		zoomService mutationTask
		gldbService codeCreator
	}

	// codeCreator creates a Session join code for a user.
	codeCreator interface {
		CreateUserJoinCode(ctx context.Context, sessionID string) (string, string, error)
	}

	Task interface {
		// Run takes a signup form struct and executes some action.
		// Ex.: Send an email, post a Slack message.
		run(ctx context.Context, signup Signup) error
		// Name Returns the name of the task.
		name() string
		// IsRequired determines if the signup request fails when this task fails. If the task is not required and fails, the signup can still succeed.
		isRequired() bool
	}

	mutationTask interface {
		run(ctx context.Context, signup *Signup) error
		name() string
	}

	signupServiceOptions struct {
		// Key-value map with the Central Time meeting start hour (int) as the keys, and Zoom Meeting ID as the values.
		// Ex: {17: "86935241734"} denotes meeting with ID, "86935241734", starts at 5pm central.
		meetings map[int]string
		tasks    []Task
		// The Zoom Service needs to mutate the Signup struct with a meeting join URL. Due to this mutation, we need to pull the zoom service out of the task flow and use it before running the tasks.
		zoomService mutationTask
		gldbService codeCreator
	}

	Location struct {
		Name         string `json:"name"`
		Line1        string `json:"line1"`
		CityStateZip string `json:"cityStateZip"`
		MapURL       string `json:"mapUrl"`
	}

	// Request params for the Operation Spark Message Template Renderer service.
	rendererReqParams struct {
		Template      osRendererTemplate `json:"template"`
		ZoomLink      string             `json:"zoomLink"`
		Date          time.Time          `json:"date"`
		Name          string             `json:"name"`
		LocationType  string             `json:"locationType"`
		Location      Location           `json:"location"`
		JoinCode      string             `json:"joinCode,omitempty"`
		IsGmail       bool               `json:"isGmail"`
		GreenlightURL string             `json:"greenlightUrl"`
	}

	osRenderer struct {
		// OpSpark Message Template Renderer Service base URL.
		// Defaults to https://sms.operationspark.org
		baseURL string
	}

	osRendererTemplate string
)

const (
	INFO_SESSION_TEMPLATE osRendererTemplate = "InfoSession"
)

// StructToBase64 marshals a struct to JSON then encodes the string to base64.
func (m *rendererReqParams) toBase64() (string, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshall: %w", err)
	}

	return base64.URLEncoding.EncodeToString(j), nil
}

// FromBase64 decodes a base64 string into a messagingReqParams struct.
func (m *rendererReqParams) fromBase64(encoded string) error {
	jsonBytes, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, m)
}

func (s Signup) MarshalJSON() ([]byte, error) {
	return json.Marshal(SignupJSON{
		SignupAlias(s),
		s.ZoomMeetingURL(),
	})
}

// WelcomeData takes a Signup and prepares template variables for use in the Welcome email template.
func (s *Signup) welcomeData() (welcomeVariables, error) {
	if s.StartDateTime.IsZero() {
		return welcomeVariables{
			FirstName: s.NameFirst,
			LastName:  s.NameLast,
		}, nil
	}
	ctz, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return welcomeVariables{}, err
	}

	line1, cityStateZip := greenlight.ParseAddress(s.GooglePlace.Address)
	return welcomeVariables{
		FirstName:            s.NameFirst,
		LastName:             s.NameLast,
		SessionTime:          s.StartDateTime.In(ctz).Format("3:04 PM MST"),
		SessionDate:          s.StartDateTime.In(ctz).Format("Monday, Jan 02"),
		ZoomURL:              s.ZoomMeetingURL(),
		LocationLine1:        line1,
		LocationCityStateZip: cityStateZip,
		LocationMapURL:       greenlight.GoogleLocationLink(s.GooglePlace.Address),
		JoinCode:             s.JoinCode,
		IsGmail:              s.isGmail(),
		GreenlightEnrollURL:  s.greenlightAutoEnrollURL("https://greenlight.operationspark.org"),
	}, nil
}

// Summary creates a string, summarizing a signup event.
func (s *Signup) Summary() string {
	sessionNote := fmt.Sprintf("%s %s has signed up for %s.", s.NameFirst, s.NameLast, s.Cohort)
	if s.StartDateTime.IsZero() {
		sessionNote = fmt.Sprintf("%s %s requested information on upcoming session times.", s.NameFirst, s.NameLast)
	}
	msg := strings.Join([]string{
		sessionNote,
		fmt.Sprintf("Ph: %s", s.Cell),
		fmt.Sprintf("email: %s", s.Email),
	}, "\n")
	return msg
}

func (su *Signup) SetZoomMeetingID(id int64) {
	su.zoomMeetingID = id
}

func (su *Signup) SetZoomJoinURL(url string) {
	su.zoomMeetingURL = url
}

func (su Signup) ZoomMeetingID() int64 {
	// Set in SignupService.attachZoomMeetingID()
	return su.zoomMeetingID
}

func (su Signup) ZoomMeetingURL() string {
	return su.zoomMeetingURL
}

// ShortMessage creates a signup confirmation message in 160 characters or less.
func (su Signup) shortMessage(infoURL string) (string, error) {
	// Handle "None of these fit my schedule"
	if su.StartDateTime.IsZero() {
		return fmt.Sprintf("Hello from Operation Spark!\nView this link for details:\n%s", infoURL), nil
	}

	// Set times to Central time
	ctz, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return "", fmt.Errorf("loadLocation: %w", err)
	}
	infoTime := su.StartDateTime.In(ctz).Format("3:04p MST")
	infoDate := su.StartDateTime.In(ctz).Format("Mon Jan 02")

	msg := fmt.Sprintf(
		"You've signed up for an info session with Operation Spark!\nThe session is %s @ %s.",
		infoDate,
		infoTime,
	)

	// Refer to email if the Information Link is not set for some reason.
	if len(infoURL) == 0 {
		return msg + "\nCheck your email for confirmation.", nil
	}
	// Append the Information Short Link
	return msg + fmt.Sprintf("\nView this link for details:\n%s", infoURL), nil

}

// GreenlightAutoEnrollURL returns a URL that auto-enrolls a user into a Greenlight session.
func (su Signup) greenlightAutoEnrollURL(greenlightHost string) string {
	if len(su.SessionID) == 0 {
		return ""
	}
	return fmt.Sprintf("%s/sessions/%s/?subview=overview&userJoinCode=%s&joinCode=%s", greenlightHost, su.SessionID, su.userJoinCode, su.JoinCode)
}

// IsGmail returns true if the user's email is a Gmail address.
func (su Signup) isGmail() bool {
	return strings.HasSuffix(su.Email, "gmail.com")
}

// ShortMessagingURL produces a custom URL for use on Operation Spark's SMS Messaging Preview service.
// https://github.com/OperationSpark/sms.opspark.org
func (su Signup) shortMessagingURL(greenlightHost, baseURL string) (string, error) {
	line1, cityStateZip := greenlight.ParseAddress(su.GooglePlace.Address)

	p := rendererReqParams{
		Template:      INFO_SESSION_TEMPLATE,
		ZoomLink:      su.zoomMeetingURL,
		Date:          su.StartDateTime,
		Name:          su.NameFirst,
		LocationType:  su.LocationType,
		JoinCode:      su.JoinCode,
		IsGmail:       su.isGmail(),
		GreenlightURL: su.greenlightAutoEnrollURL(greenlightHost),
		Location: Location{
			Name:         su.GooglePlace.Name,
			Line1:        line1,
			CityStateZip: cityStateZip,
			MapURL:       greenlight.GoogleLocationLink(su.GooglePlace.Address),
		},
	}

	encoded, err := p.toBase64()
	if err != nil {
		return "", fmt.Errorf("structToBase64: %w", err)
	}

	return fmt.Sprintf("%s/m/%s", baseURL, encoded), nil
}

// String creates a human-readable Signup for debugging purposes.
func (su Signup) String() string {
	ctz, _ := time.LoadLocation("America/Chicago")
	return fmt.Sprintf("%q\n%q\n%q\n%q\n%q\n%q\n",
		su.NameFirst,
		su.NameLast,
		su.Email,
		su.Cell,
		su.StartDateTime.In(ctz).Format(time.RFC822),
		su.SessionID,
	)
}

func newSignupService(o signupServiceOptions) *SignupService {
	return &SignupService{
		meetings:    o.meetings,
		tasks:       o.tasks,
		zoomService: o.zoomService,
		gldbService: o.gldbService,
	}
}

// Register concurrently executes a list of tasks. Completion of tasks are not dependent on each other.
func (sc *SignupService) register(ctx context.Context, su Signup) (Signup, error) {
	// TODO: Create specific errors for each handler
	err := sc.attachZoomMeetingID(&su)
	if err != nil {
		return su, fmt.Errorf("attachZoomMeetingID: %w", err)
	}
	err = sc.zoomService.run(ctx, &su)
	if err != nil {
		return su, fmt.Errorf("zoomService.run: %w", err)
	}

	if su.SessionID != "" {
		joinCodeID, sessionJoinCode, err := sc.gldbService.CreateUserJoinCode(ctx, su.SessionID)
		if err != nil {
			return su, fmt.Errorf("userJoinCode Create: %w", err)
		}

		su.userJoinCode = joinCodeID
		su.JoinCode = sessionJoinCode
	}

	// create user-specific info session details URL
	msgngURL, err := su.shortMessagingURL(os.Getenv("GREENLIGHT_HOST"), os.Getenv("OS_RENDERING_SERVICE_URL"))
	if err != nil {
		return su, fmt.Errorf("shortMessagingURL: %w", err)
	}

	shorty := NewURLShortener(ShortenerOpts{apiKey: os.Getenv("URL_SHORTENER_API_KEY")})
	shortLink, err := shorty.ShortenURL(ctx, msgngURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shortenURL ERROR: %v", err)
		// Don't early return. ShortenURL returns the original URL if there is a failure
		// Fallback to long URL if shortener fails
	}

	su.ShortLink = shortLink

	// Run each task in a go routine for concurrent execution
	g, ctx := errgroup.WithContext(ctx)
	for _, task := range sc.tasks {
		func(t Task) {
			g.Go(func() error {
				err := t.run(ctx, su)
				if err != nil {
					if t.isRequired() {
						return fmt.Errorf("task failed: %q: %w", t.name(), err)
					}
					fmt.Printf("task failed: %q: %v", t.name(), err)
				}
				return nil
			})
		}(task)
	}
	if err := g.Wait(); err != nil {
		return su, err
	}
	return su, nil
}

// AttachZoomMeetingID sets the Zoom meeting ID on the Signup based on the Signup's StartDateTime and the SignService's Zoom sessions.
func (sc *SignupService) attachZoomMeetingID(su *Signup) error {
	// Do nothing if the user has not signed up for a specific session
	if su.StartDateTime.IsZero() {
		return nil
	}
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return fmt.Errorf("loadLocation: %w", err)
	}
	sessionStart := su.StartDateTime
	centralStart := sessionStart.In(loc)

	if _, ok := sc.meetings[centralStart.Hour()]; !ok {
		return fmt.Errorf("no zoom meeting found with start hour: %d", centralStart.Hour())
	}
	id, err := strconv.Atoi(sc.meetings[centralStart.Hour()])
	if err != nil {
		return fmt.Errorf("convert string to int: %w", err)
	}
	su.SetZoomMeetingID(int64(id))
	return nil
}

// CreateMessageURL creates a custom URL for use on Operation Spark's SMS Messaging Preview service.
func (osm *osRenderer) CreateMessageURL(p notify.Participant) (string, error) {
	params := rendererReqParams{
		Template:     INFO_SESSION_TEMPLATE,
		ZoomLink:     p.ZoomJoinURL,
		Name:         p.NameFirst,
		Date:         p.SessionDate,
		LocationType: p.SessionLocationType,
		Location:     Location(p.SessionLocation),
	}
	encoded, err := params.toBase64()
	if err != nil {
		return "", fmt.Errorf("structToBase64: %w", err)
	}
	return fmt.Sprintf("%s/m/%s", osm.baseURL, encoded), nil
}
