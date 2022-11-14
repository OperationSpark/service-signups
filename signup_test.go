package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type MockMailgunService struct {
	WelcomeFunc func(context.Context, Signup) error
	called      bool
}

func (ms *MockMailgunService) run(ctx context.Context, su Signup) error {
	ms.called = true
	return ms.WelcomeFunc(ctx, su)
}

func (ms *MockMailgunService) name() string {
	return "mock mailgun service"
}

type MockZoomService struct{}

func (*MockZoomService) run(ctx context.Context, su *Signup) error {
	mockZoomJoinURL := "https://us06web.zoom.us/w/fakemeetingid?tk=faketoken"
	su.SetZoomJoinURL(mockZoomJoinURL)
	return nil
}

func (*MockZoomService) name() string {
	return "mock zoom service"
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
			WelcomeFunc: func(ctx context.Context, s Signup) error {
				return nil
			},
		}

		zoomService := &MockZoomService{}

		signupService := newSignupService(signupServiceOptions{
			tasks:       []task{mailService},
			zoomService: zoomService,
		})

		err := signupService.register(context.Background(), signup)
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
				ZoomURL:     "https://us06web.zoom.us/w/fakemeetingid?tk=faketoken",
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
				ZoomURL:     "https://us06web.zoom.us/w/fakemeetingid?tk=faketoken",
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
				ZoomURL:     "https://us06web.zoom.us/w/fakemeetingid?tk=faketoken",
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
			test.signup.SetZoomJoinURL("https://us06web.zoom.us/w/fakemeetingid?tk=faketoken")
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

		err := suSvc.attachZoomMeetingID(&su)
		assertNilError(t, err)

		gotID := su.ZoomMeetingID()
		wantID := int64(12123456789) // Meeting for 12p Central

		assertEqual(t, gotID, wantID)

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

func TestMarshalJSON(t *testing.T) {
	t.Run("should contain the 'zoomJoinUrl' field", func(t *testing.T) {
		su := Signup{
			zoomMeetingURL: "http://jointhiszoom.com",
		}

		j, err := json.Marshal(su)
		if err != nil {
			t.Fatalf("marshall: %v", err)
		}

		hasJoinField := bytes.Contains(j, []byte(`"zoomJoinUrl":"http://jointhiszoom.com"`))

		assertEqual(t, hasJoinField, true)
	})
}

func TestShortMessage(t *testing.T) {
	// URLs will be shortened by Twilio in this format
	mockShortLink := "https://oprk.org/kRds5MKvKI"
	maxCopyLen := 160

	t.Run(fmt.Sprintf("creates a message of %v characters or less", maxCopyLen), func(t *testing.T) {
		su := Signup{
			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-10-31T17:00:00.000Z"),
		}

		msg, err := su.shortMessage(mockShortLink)
		if err != nil {
			t.Fatal(err)
		}
		if len(msg) > maxCopyLen {
			t.Fatalf("\nMessage is over 160 characters.\nLength: %v\n\nMessage:\n%s", len(msg), msg)
		}
	})

	t.Run("creates a a message for SMS", func(t *testing.T) {
		su := Signup{
			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-10-31T17:00:00.000Z"),
		}

		got, err := su.shortMessage(mockShortLink)
		assertNilError(t, err)

		want := `You've signed up for an info session with Operation Spark!
The session is Mon Oct 31 @ 12:00p CDT.
View this link for details:
https://oprk.org/kRds5MKvKI`

		assertEqual(t, got, want)
	})

	t.Run("send proper message when someone select 'None of these [sessions] fit my schedule'", func(t *testing.T) {

		su := Signup{
			NameFirst: "Jamey",
			NameLast:  "Ramet",
			Email:     "jramet0@narod.ru",
		}

		got, err := su.shortMessage(mockShortLink)
		assertNilError(t, err)
		want := "Hello from Operation Spark!\nView this link for details:\nhttps://oprk.org/kRds5MKvKI"
		assertEqual(t, got, want)

	})
}

func TestStructToBase64(t *testing.T) {
	t.Run("serializes a struct to base 64 encoding", func(t *testing.T) {
		params := messagingReqParams{
			Template: "InfoSession",
			ZoomLink: "https://us06web.zoom.us/j/12345678910",
			Date:     mustMakeTime(t, time.RFC3339, "2022-10-05T17:00:00.000Z"),
			Name:     "FirstName",
		}

		want := "eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy9qLzEyMzQ1Njc4OTEwIiwiZGF0ZSI6IjIwMjItMTAtMDVUMTc6MDA6MDBaIiwibmFtZSI6IkZpcnN0TmFtZSJ9"

		got, err := structToBase64(params)
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, got, want)
	})
}

func TestFromBase64(t *testing.T) {
	t.Run("decodes", func(t *testing.T) {
		wantParams := messagingReqParams{
			Template: "InfoSession",
			ZoomLink: "https://us06web.zoom.us/j/12345678910",
			Date:     mustMakeTime(t, "January 02, 2006 3pm MST", "December 25, 2022 1pm CST"),
			Name:     "Halle Bot",
		}

		encoded, err := structToBase64(wantParams)
		if err != nil {
			t.Fatal(err)
		}

		var gotParams messagingReqParams

		err = gotParams.fromBase64(encoded)
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, gotParams.Name, wantParams.Name)
		assertEqual(t, gotParams.Date.Format("January 02, 2006 3pm MST"), "December 25, 2022 1pm CST")
		assertEqual(t, gotParams.ZoomLink, wantParams.ZoomLink)
		assertEqual(t, gotParams.Template, wantParams.Template)

	})

	t.Run("decodes pre-encoded info session details link", func(t *testing.T) {
		var params messagingReqParams

		err := params.fromBase64("eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy9qLzEyMzQ1Njc4OTEwIiwiZGF0ZSI6IjIwMjItMTAtMDVUMTc6MDA6MDAuMDAwWiIsIm5hbWUiOiJGaXJzdE5hbWUiLCJsb2NhdGlvblR5cGUiOiJIWUJSSUQiLCJsb2NhdGlvbiI6eyJuYW1lIjoiU29tZSBQbGFjZSIsImxpbmUxIjoiMTIzIE1haW4gU3QiLCJjaXR5U3RhdGVaaXAiOiJDaXR5LCBTdGF0ZSAxMjM0NSIsIm1hcFVybCI6Imh0dHBzOi8vd3d3Lmdvb2dsZS5jb20vbWFwcy9wbGFjZS8xMjMrTWFpbitTdCwrQ2l0eSwrU3RhdGUrMTIzNDUifX0=")

		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, params.Name, "FirstName")
		assertEqual(t, params.Template, "InfoSession")
		assertEqual(t, params.Date.Format(time.RFC3339), "2022-10-05T17:00:00Z")
		assertEqual(t, params.ZoomLink, "https://us06web.zoom.us/j/12345678910")
	})

}

func TestString(t *testing.T) {
	t.Run("returns a human readable string", func(t *testing.T) {
		s := Signup{
			NameFirst:     "Yasiin",
			NameLast:      "Bey",
			Cell:          "555-555-5555",
			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-03-14T17:00:00.000Z"),
			Cohort:        "is-mar-14-22-12pm",
			Email:         "yasiin@blackstar.net",
			SessionId:     "WpkB3jcw6gCw2uEMf",
		}

		got := s.String()
		want := `"Yasiin"
"Bey"
"yasiin@blackstar.net"
"555-555-5555"
"14 Mar 22 12:00 CDT"
"WpkB3jcw6gCw2uEMf"`

		if !strings.Contains(got, want) {
			t.Fatal(got)
		}
	})
}

func TestParseAddress(t *testing.T) {
	t.Run("parses an address string into street address string and cityStateZip string", func(t *testing.T) {
		address := "514 Franklin Ave, New Orleans, LA 70117, USA"

		line1, cityStateZip := parseAddress(address)
		assertEqual(t, line1, "514 Franklin Ave")
		assertEqual(t, cityStateZip, "New Orleans, LA 70117")

	})

	t.Run("handles empty string", func(t *testing.T) {
		address := ""

		line1, cityStateZip := parseAddress(address)
		assertEqual(t, line1, "")
		assertEqual(t, cityStateZip, "")

	})

	t.Run("handles addresses with a street address only", func(t *testing.T) {
		address := "514 Franklin Ave"

		line1, cityStateZip := parseAddress(address)
		assertEqual(t, line1, "514 Franklin Ave")
		assertEqual(t, cityStateZip, "")

	})
}

func TestGoogleLocationLink(t *testing.T) {
	t.Run("returns the google maps link to address", func(t *testing.T) {
		address := "514 Franklin Ave, New Orleans, LA 70117, USA"

		assertEqual(t, googleLocationLink(address), "https://www.google.com/maps/place/514+Franklin+Ave%2CNew+Orleans%2C+LA+70117")
	})

	t.Run("handles empty string", func(t *testing.T) {
		address := ""

		assertEqual(t, googleLocationLink((address)), "")

	})

	t.Run("handles addresses with a street address only", func(t *testing.T) {
		address := "514 Franklin Ave"

		assertEqual(t, googleLocationLink(address), "")

	})
}

func TestShortMessagingURL(t *testing.T) {
	t.Skip()
	t.Run("creates a user specific info session details URL", func(t *testing.T) {
		s := Signup{
			NameFirst:     "Yasiin",
			NameLast:      "Bey",
			Cell:          "555-555-5555",
			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-03-14T17:00:00.000Z"),
			Cohort:        "is-mar-14-22-12pm",
			Email:         "yasiin@blackstar.net",
			SessionId:     "WpkB3jcw6gCw2uEMf",
			LocationType:  "HYBRID",
			GooglePlace: GooglePlace{
				Name:    "Some Place",
				Address: "123 Main St, City, State 12345",
			},
		}

		baseUrl := "https://sms.opspark.org"

		// wantMessagingServiceParams := messagingReqParams{
		// 	Template: "InfoSession",
		// 	ZoomLink: s.zoomMeetingURL,
		// 	Date:     s.StartDateTime,
		// 	Name:     s.NameFirst,
		// }

		want := "https://sms.opspark.org/m/eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy9qLzEyMzQ1Njc4OTEwIiwiZGF0ZSI6IjIwMjItMTAtMDVUMTc6MDA6MDAuMDAwWiIsIm5hbWUiOiJGaXJzdE5hbWUiLCJsb2NhdGlvblR5cGUiOiJIWUJSSUQiLCJsb2NhdGlvbiI6eyJuYW1lIjoiU29tZSBQbGFjZSIsImxpbmUxIjoiMTIzIE1haW4gU3QiLCJjaXR5U3RhdGVaaXAiOiJDaXR5LCBTdGF0ZSAxMjM0NSIsIm1hcFVybCI6Imh0dHBzOi8vd3d3Lmdvb2dsZS5jb20vbWFwcy9wbGFjZS8xMjMrTWFpbitTdCwrQ2l0eSwrU3RhdGUrMTIzNDUifX0="

		got, err := s.shortMessagingURL(baseUrl)
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, got, want)
	})

}
