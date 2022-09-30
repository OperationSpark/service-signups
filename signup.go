package signup

import (
	"fmt"
	"strings"
	"time"
)

type Signup struct {
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
}

type welcomeVariables struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	SessionTime string `json:"sessionTime"`
	SessionDate string `json:"sessionDate"`
	ZoomURL     string `json:"zoomURL"`
}

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

type mailer interface {
	SendWelcome(signup Signup) error
}

type task interface {
	// Run takes a signup form struct and executes some action.
	// Ex.: Send an email, post a Slack message.
	run(signup Signup) error
	// Name Returns the name of the task.
	name() string
}
type SignupService struct {
	tasks []task
}

func NewSignupService(tasks ...task) *SignupService {
	return &SignupService{
		tasks: tasks,
	}
}

// Register executes a series of tasks in order. If one fails, the remaining tasks are cancelled.
func (sc *SignupService) Register(su Signup) error {
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
