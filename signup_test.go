package signup

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

		signupService := newSignupService(signupServiceOptions{
			tasks: []task{mailService},
		})

		err := signupService.register(signup)
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
		if diff := cmp.Diff(test.want, got, cmpopts.IgnoreUnexported(test.want, got)); diff != "" {
			t.Errorf("handleJSON() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestWelcomeData(t *testing.T) {
	sessionStart, _ := time.Parse(time.RFC3339, "2022-02-28T23:30:00.000Z")
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
				LastName:    "Testaroni",
				SessionDate: "Monday, Feb 28",
				SessionTime: "5:30 PM CST",
				ZoomURL:     "https://us06web.zoom.us/s/17171717171",
			},
		},
		{
			name: "handle empty startDateTime",
			signup: Signup{
				ProgramId:     "",
				NameFirst:     "Cordell",
				NameLast:      "Kinavan",
				Email:         "henri@email.com",
				Cell:          "555-123-4567",
				Referrer:      "Word of mouth",
				StartDateTime: time.Time{}, //  Empty value
			},
			want: welcomeVariables{
				FirstName:   "Cordell",
				LastName:    "Kinavan",
				SessionDate: "",
				SessionTime: "",
			},
		},
		{
			name: "contains the correct Zoom URL",
			signup: Signup{
				ProgramId:        "",
				NameFirst:        "Delcina",
				NameLast:         "Hallward",
				Email:            "henri@email.com",
				Cell:             "555-123-4567",
				Referrer:         "Word of mouth",
				ReferrerResponse: "Jane Smith",
				StartDateTime:    sessionStart,
				Cohort:           "is-feb-28-22-12pm",
			},
			want: welcomeVariables{
				FirstName:   "Delcina",
				LastName:    "Hallward",
				SessionDate: "Monday, Feb 28",
				SessionTime: "5:30 PM CST",
				ZoomURL:     "https://us06web.zoom.us/s/17171717171",
			},
		},
		{
			name: "contains the correct Zoom URL",
			signup: Signup{
				ProgramId:        "",
				NameFirst:        "Miquela",
				NameLast:         "Carmo",
				Email:            "mcarmo2@opensource.org",
				Cell:             "832-546-7105",
				Referrer:         "Word of mouth",
				ReferrerResponse: "Jane Smith",
				StartDateTime:    time.Date(2022, time.Month(10), 10, 17, 0, 0, 0, time.UTC),
				Cohort:           "is-oct-10-22-12pm",
			},
			want: welcomeVariables{
				FirstName:   "Miquela",
				LastName:    "Carmo",
				SessionDate: "Monday, Oct 10",
				SessionTime: "12:00 PM CDT",
				ZoomURL:     "https://us06web.zoom.us/s/12121212121",
			},
		},
	}

	suSvc := newSignupService(signupServiceOptions{
		meetings: map[int]string{
			12: "12121212121",
			17: "17171717171",
		},
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			suSvc.attachZoomMeetingID(&test.signup)
			got, err := test.signup.welcomeData()
			if err != nil {
				t.Errorf("Unexpected error for input date %v.\n%v", test.signup.StartDateTime, err)
			}

			assertDeepEqual(t, got, test.want)
		})
	}
}

func TestAttachZoomMeetingID(t *testing.T) {
	t.Run("generates the correct Zoom URL for a given session start time", func(t *testing.T) {
		suSvc := newSignupService(signupServiceOptions{
			meetings: map[int]string{
				// Noon Central
				12: "12123456789",
				// 5p Central
				17: "17123456789",
			},
		})
		sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
		su := Signup{
			StartDateTime: sessionStartDate,
		}

		suSvc.attachZoomMeetingID(&su)

		gotID := su.ZoomMeetingID()
		wantID := int64(12123456789) // Meeting for 12p Central

		assertEqual(t, gotID, wantID)

		gotURL := su.ZoomMeetingURL()
		wantURL := "https://us06web.zoom.us/s/12123456789" // Meeting for 12p Central
		assertEqual(t, gotURL, wantURL)
	})
}

func TestSummary(t *testing.T) {
	sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
	tests := []struct {
		s    Signup
		want []string
	}{
		{
			s: Signup{
				NameFirst:     "Yasiin",
				NameLast:      "Bey",
				Cell:          "555-555-5555",
				StartDateTime: sessionStartDate,
				Cohort:        "is-mar-14-22-12pm",
			},
			want: []string{"Yasiin Bey has signed up for is-mar-14-22-12pm", "555-555-5555"},
		},
		{
			s:    Signup{NameFirst: "Solána", NameLast: "Rowe", StartDateTime: time.Time{}},
			want: []string{"Solána Rowe requested information on upcoming session times."},
		},
	}

	for _, test := range tests {
		got := test.s.Summary()
		for _, want := range test.want {
			if !strings.Contains(got, want) {
				t.Fatalf("string missing in s.Summary()\n\ngot:\n \"%s\"\n\nwant: \"%s\"\n\nSignup:\n%+v", got, want, test.s)
			}
		}
	}
}