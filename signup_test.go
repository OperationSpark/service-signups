package signup

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type MockMailgunService struct {
	WelcomeFunc func(Signup) error
	called      bool
}

func (ms *MockMailgunService) run(su Signup) error {
	ms.called = true
	return ms.WelcomeFunc(su)
}

func (ms *MockMailgunService) name() string {
	return "mock mailgun service"
}

func TestRegisterUser(t *testing.T) {
	t.Run("triggers an 'Welcome Email'", func(t *testing.T) {
		signup := Signup{
			NameFirst:        "Henri",
			NameLast:         "Testaroni",
			Email:            "henri@email.com",
			Cell:             "555-123-4567",
			Referrer:         "instagram",
			ReferrerResponse: "",
		}

		mailService := &MockMailgunService{
			WelcomeFunc: func(s Signup) error {
				return nil
			},
		}

		signupService := NewSignupService(mailService)

		err := signupService.Register(signup)
		if err != nil {
			t.Fatalf("register: %v", err)
		}

		if !mailService.called {
			t.Fatalf("mailService.SendWelcome should have been called")
		}
	})
}

func TestHandleJson(t *testing.T) {
	tests := []struct {
		json []byte
		want Signup
		err  error
	}{
		{[]byte(`{"startDateTime": null}`), Signup{}, nil},
		{
			[]byte(`{"startDateTime": ""}`),
			Signup{},
			&InvalidFieldError{Field: "startDateTime"},
		},
		{
			[]byte(`{
			"nameFirst": "Henri",
			"nameLast": "Testaroni",
			"email": "henri@email.com",
			"cell": "555-123-4567",
			"referrer": "instagram",
			"referrerResponse": ""
		}
		`),
			Signup{
				NameFirst:        "Henri",
				NameLast:         "Testaroni",
				Email:            "henri@email.com",
				Cell:             "555-123-4567",
				Referrer:         "instagram",
				ReferrerResponse: "",
			},
			nil},
	}

	for _, test := range tests {
		got := Signup{}
		err := handleJson(&got, bytes.NewReader(test.json))
		if err != nil && test.err == nil {
			t.Errorf("Unexpected error for \n%s\nerror: %s", string(test.json), err)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("handleJSON() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestWelcomeData(t *testing.T) {
	sessionStart, _ := time.Parse(time.RFC3339, "2022-03-21T22:30:00.000Z")
	tests := []struct {
		name   string
		signup Signup
		want   welcomeVariables
	}{
		{
			name: "18:00 UTC is converted to noon central standard time",
			signup: Signup{
				ProgramId:        "",
				NameFirst:        "Henri",
				NameLast:         "Testaroni",
				Email:            "henri@email.com",
				Cell:             "555-123-4567",
				Referrer:         "Word of mouth",
				ReferrerResponse: "Jane Smith",
				StartDateTime:    sessionStart,
				Cohort:           "is-feb-28-22-12pm",
			},
			want: welcomeVariables{
				FirstName:   "Henri",
				SessionDate: "Monday, Feb 28",
				SessionTime: "5:30 PM CDT",
			},
		},
		{
			name: "handle empty startDateTime",
			signup: Signup{
				ProgramId:     "",
				NameFirst:     "Henri",
				NameLast:      "Testaroni",
				Email:         "henri@email.com",
				Cell:          "555-123-4567",
				Referrer:      "Word of mouth",
				StartDateTime: time.Time{}, //  Empty value
				Cohort:        "is-feb-28-22-12pm",
			},
			want: welcomeVariables{
				FirstName:   "Henri",
				SessionDate: "",
				SessionTime: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.signup.welcomeData()
			if err != nil {
				t.Errorf("Unexpected error for input date %v.\n%v", test.signup.StartDateTime, err)
			}
			if got.SessionTime != test.want.SessionTime {
				t.Errorf("s.WelcomeData():\ns.StartDateTime:%v\nwant:%s\ngot:%s", test.signup.StartDateTime, test.want.SessionTime, got.SessionTime)
			}
		})
	}
}
