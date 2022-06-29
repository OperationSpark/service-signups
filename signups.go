// Package signups implements utilities for passing Info Session signups
// from operationspark.org to relevant services.
package signups

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

// WelcomeData takes a Signup and prepares data for use in the Welcome email template
func (s *Signup) WelcomeData() (WelcomeValues, error) {
	if s.StartDateTime.IsZero() {
		return WelcomeValues{
			DisplayName: s.NameFirst,
		}, nil
	}
	ctz, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return WelcomeValues{}, err
	}
	return WelcomeValues{
		DisplayName: s.NameFirst,
		SessionDate: s.StartDateTime.Format("Monday, Jan 02"),
		SessionTime: s.StartDateTime.In(ctz).Format("3:04 PM MST"),
	}, nil
}

// html populates the Info Session Welcome email template with values from the Signup. It then writes the result to the io.Writer, w.
func (s *Signup) Html(w io.Writer) error {

	templatePath, err := filepath.Abs(filepath.Join(".", "email", "templates", "signup_template.html"))
	if err != nil {
		fmt.Println("Error: 'templatePath\nPath: ", templatePath, err)
		return err
	}
	signupTemplate, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	t, err := template.New("welcome").Parse(string(signupTemplate))
	if err != nil {
		return err
	}

	data, err := s.WelcomeData()
	if err != nil {
		return err
	}

	err = t.Execute(w, data)

	if err != nil {
		return err
	}
	return nil
}
