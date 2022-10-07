package signup

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type (
	Signup struct {
		ProgramId        string    `json:"programId" schema:"programId"`
		NameFirst        string    `json:"nameFirst" schema:"nameFirst"`
		NameLast         string    `json:"nameLast" schema:"nameLast"`
		Email            string    `json:"email" schema:"email"`
		Cell             string    `json:"cell" schema:"cell"`
		Referrer         string    `json:"referrer" schema:"referrer"`
		ReferrerResponse string    `json:"referrerResponse" schema:"referrerResponse"`
		StartDateTime    time.Time `json:"startDateTime,omitempty" schema:"startDateTime"`
		Cohort           string    `json:"cohort" schema:"cohort"`
		SessionId        string    `json:"sessionId" schema:"sessionId"`
		Token            string    `json:"token" schema:"token"`
		zoomMeetingID    int64
	}

	welcomeVariables struct {
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		SessionTime string `json:"sessionTime"`
		SessionDate string `json:"sessionDate"`
		ZoomURL     string `json:"zoomURL"`
	}

	SignupService struct {
		// Key-value map with the Central Time meeting start hour (int) as the keys, and Zoom Meeting ID as the values.
		// Ex: {17: "86935241734"} denotes meeting with ID, "86935241734", starts at 5pm central.
		meetings map[int]string
		tasks    []task
	}

	task interface {
		// Run takes a signup form struct and executes some action.
		// Ex.: Send an email, post a Slack message.
		run(signup Signup) error
		// Name Returns the name of the task.
		name() string
	}

	signupServiceOptions struct {
		// Key-value map with the Central Time meeting start hour (int) as the keys, and Zoom Meeting ID as the values.
		// Ex: {17: "86935241734"} denotes meeting with ID, "86935241734", starts at 5pm central.
		meetings map[int]string
		tasks    []task
	}
)

// welcomeData takes a Signup and prepares data for use in the Welcome email template
func (s *Signup) welcomeData() (welcomeVariables, error) {
	if s.StartDateTime.IsZero() {
		return welcomeVariables{
			FirstName: s.NameFirst,
		}, nil
	}
	ctz, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return welcomeVariables{}, err
	}
	return welcomeVariables{
		FirstName:   s.NameFirst,
		LastName:    s.NameLast,
		SessionDate: s.StartDateTime.Format("Monday, Jan 02"),
		SessionTime: s.StartDateTime.In(ctz).Format("3:04 PM MST"),
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

func (su Signup) ZoomMeetingID() int64 {
	// Set in SignupService.attachZoomMeetingID()
	return su.zoomMeetingID
}

func (su Signup) ZoomMeetingURL() string {
	return fmt.Sprintf("https://us06web.zoom.us/s/%d", su.zoomMeetingID)
}

func newSignupService(o signupServiceOptions) *SignupService {
	return &SignupService{
		meetings: o.meetings,
		tasks:    o.tasks,
	}
}

// Register executes a series of tasks in order. If one fails, the remaining tasks are cancelled.
func (sc *SignupService) register(su Signup) error {
	// TODO: Create specific errors for each handler
	// TODO: Use context.Context to cancel subsequent requests on any failures
	for _, task := range sc.tasks {
		err := task.run(su)
		if err != nil {
			return fmt.Errorf("task failed: %q: %v", task.name(), err)
		}
	}
	return nil
}

// AttachZoomMeetingID sets the Zoom meeting ID on the Signup based on the Signup's StartDateTime and the SignService's Zoom sessions.
func (sc *SignupService) attachZoomMeetingID(su *Signup) error {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return fmt.Errorf("loadLocation: %v", err)
	}
	sessionStart := su.StartDateTime
	centralStart := sessionStart.In(loc)

	if _, ok := sc.meetings[centralStart.Hour()]; !ok {
		return fmt.Errorf("no zoom meeting found with start hour: %d", centralStart.Hour())
	}
	id, err := strconv.Atoi(sc.meetings[centralStart.Hour()])
	if err != nil {
		return fmt.Errorf("convert string to intL %v", err)
	}
	su.SetZoomMeetingID(int64(id))
	return nil
}
