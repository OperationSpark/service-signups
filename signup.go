package signup

import (
	"fmt"
	"strings"
)

type Session struct {
	Id string `json:"id"`
}

type Referral struct {
	Value          string `json:"value"`
	AdditionalInfo string `json:"additionalInfo"`
}

type SignUp struct {
	Session      Session  `json:"session"`
	Email        string   `json:"email"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	Phone        string   `json:"phone"`
	ReferencedBy Referral `json:"referencedBy"`
}

func (s *SignUp) Summary() string {
	msg := strings.Join([]string{
		fmt.Sprintf("%s %s has signed up for %s", s.FirstName, s.LastName, s.Session.Id),
		fmt.Sprintf("Ph: %s)", s.Phone),
		fmt.Sprintf("email: %s)", s.Email),
	}, "\n")
	return msg
}
