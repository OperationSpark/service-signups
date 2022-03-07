// Package signups implements utilities for passing Info Session signups
// from operationspark.org to relevant services.
package signups

import (
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/operationspark/slack-session-signups/email"
)

type Signup struct {
	ProgramId        string    `json:"programId" schema:"programId"`
	NameFirst        string    `json:"nameFirst" schema:"nameFirst"`
	NameLast         string    `json:"nameLast" schema:"nameLast"`
	Email            string    `json:"email" schema:"email"`
	Cell             string    `json:"cell" schema:"cell"`
	Referrer         string    `json:"referrer" schema:"referrer"`
	ReferrerResponse string    `json:"referrerResponse" schema:"referrerResponse"`
	StartDateTime    time.Time `json:"startDateTime" schema:"startDateTime"`
	Cohort           string    `json:"cohort" schema:"cohort"`
	SessionId        string    `json:"sessionId" schema:"sessionId"`
	Token            string    `json:"token" schema:"token"`
}

// Summary creates a string, summarizing a signup event.
func (s *Signup) Summary() string {
	msg := strings.Join([]string{
		fmt.Sprintf("%s %s has signed up for %s", s.NameFirst, s.NameLast, s.Cohort),
		fmt.Sprintf("Ph: %s)", s.Cell),
		fmt.Sprintf("email: %s)", s.Email),
	}, "\n")
	return msg
}

func (s *Signup) html(w io.Writer) error {
	t, err := template.New("welcome").Parse(email.InfoSessionHtml)
	if err != nil {
		return err
	}
	data := email.WelcomeValues{
		DisplayName: s.NameFirst,
		SessionDate: s.StartDateTime.Format("Monday, Jan 02"),
		SessionTime: s.StartDateTime.Format("3:00 PM MST"),
	}
	return t.Execute(w, data)
}
