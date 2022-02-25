// Package signups implements utilities for passing Info Session signups
// from operationspark.org to relevant services.
package signups

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
