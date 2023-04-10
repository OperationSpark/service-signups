package signup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/operationspark/service-signup/greenlight"
	"github.com/operationspark/service-signup/notify"
	"github.com/stretchr/testify/require"
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

type MockGreenlightDBService struct{}

func (m *MockGreenlightDBService) CreateUserJoinCode(ctx context.Context, sessionID string) (string, string, error) {
	return "", "", nil
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
			StartDateTime:    mustMakeTime(t, time.RFC822, "16 Nov 22 18:00 UTC"), // 12 central
		}

		mailService := &MockMailgunService{
			WelcomeFunc: func(ctx context.Context, s Signup) error {
				return nil
			},
		}

		signupService := newSignupService(signupServiceOptions{
			tasks:       []Task{mailService},
			zoomService: &MockZoomService{},
			// zoom meeting id for 12 central
			meetings:    map[int]string{12: "983782"},
			gldbService: &MockGreenlightDBService{},
		})

		err := signupService.register(context.Background(), signup)
		if err != nil {
			t.Fatalf("register: %v", err)
		}

		if !mailService.called {
			t.Fatalf("mailService.SendWelcome should have been called")
		}
	})

	t.Run("sends an email even when 'None of these fit my schedule' selected.", func(t *testing.T) {
		signup := Signup{
			NameFirst:        "Henri",
			NameLast:         "Testaroni",
			Email:            "henri@email.com",
			Cell:             "555-123-4567",
			Referrer:         "instagram",
			ReferrerResponse: "",
			StartDateTime:    time.Time{}, // Empty session start time
		}

		mailService := &MockMailgunService{
			WelcomeFunc: func(ctx context.Context, s Signup) error {
				return nil
			},
		}

		zoomService := &MockZoomService{}

		signupService := newSignupService(signupServiceOptions{
			tasks:       []Task{mailService},
			zoomService: zoomService,
			gldbService: &MockGreenlightDBService{},
		})

		err := signupService.register(context.Background(), signup)
		if err != nil {
			t.Fatalf("register: %v", err)
		}

		if !mailService.called {
			t.Fatal("mailService.SendWelcome should have been called")
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
				ProgramID:        "",
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
				ProgramID:     "",
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
				ProgramID:        "",
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
				ProgramID:        "",
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
			err := suSvc.attachZoomMeetingID(&test.signup)
			assertNilError(t, err)

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
		params := rendererReqParams{
			Template:      "InfoSession",
			ZoomLink:      "https://us06web.zoom.us/j/12345678910",
			Date:          mustMakeTime(t, time.RFC3339, "2022-10-05T17:00:00.000Z"),
			Name:          "FirstName",
			IsGmail:       false,
			LocationType:  "Hybrid",
			GreenlightURL: "https://greenlight.operationspark.org/sessions/kyvYitLoFfTickbP2/?userJoinCode=123415&joinCode=12314",
			Location: Location{
				Name:         "Some Place",
				Line1:        "123 Main St",
				CityStateZip: "City, State 12345",
				MapURL:       "https://www.google.com/maps/place/123+Main+St,+City,+State+12345",
			},
		}

		want := "eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy9qLzEyMzQ1Njc4OTEwIiwiZGF0ZSI6IjIwMjItMTAtMDVUMTc6MDA6MDBaIiwibmFtZSI6IkZpcnN0TmFtZSIsImxvY2F0aW9uVHlwZSI6Ikh5YnJpZCIsImxvY2F0aW9uIjp7Im5hbWUiOiJTb21lIFBsYWNlIiwibGluZTEiOiIxMjMgTWFpbiBTdCIsImNpdHlTdGF0ZVppcCI6IkNpdHksIFN0YXRlIDEyMzQ1IiwibWFwVXJsIjoiaHR0cHM6Ly93d3cuZ29vZ2xlLmNvbS9tYXBzL3BsYWNlLzEyMytNYWluK1N0LCtDaXR5LCtTdGF0ZSsxMjM0NSJ9LCJpc0dtYWlsIjpmYWxzZSwiZ3JlZW5saWdodFVybCI6Imh0dHBzOi8vZ3JlZW5saWdodC5vcGVyYXRpb25zcGFyay5vcmcvc2Vzc2lvbnMva3l2WWl0TG9GZlRpY2tiUDIvP3VzZXJKb2luQ29kZT0xMjM0MTVcdTAwMjZqb2luQ29kZT0xMjMxNCJ9"

		got, err := params.toBase64()
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, got, want)
	})
}

func TestFromBase64(t *testing.T) {
	t.Run("decodes", func(t *testing.T) {
		wantParams := rendererReqParams{
			Template:     "InfoSession",
			ZoomLink:     "https://us06web.zoom.us/j/12345678910",
			Date:         mustMakeTime(t, "January 02, 2006 3pm MST", "December 25, 2022 1pm CST"),
			Name:         "Halle Bot",
			LocationType: "Hybrid",
			Location: Location{
				Name:         "Some Place",
				Line1:        "123 Main St",
				CityStateZip: "City, State 12345",
				MapURL:       "https://www.google.com/maps/place/123+Main+St,+City,+State+12345",
			},
		}

		encoded, err := wantParams.toBase64()
		if err != nil {
			t.Fatal(err)
		}

		var gotParams rendererReqParams

		err = gotParams.fromBase64(encoded)
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, gotParams.Name, wantParams.Name)
		assertEqual(t, gotParams.Date.Equal(wantParams.Date), true)
		assertEqual(t, gotParams.ZoomLink, wantParams.ZoomLink)
		assertEqual(t, gotParams.Template, wantParams.Template)
		assertEqual(t, gotParams.LocationType, wantParams.LocationType)
		assertEqual(t, gotParams.Location.Name, wantParams.Location.Name)
		assertEqual(t, gotParams.Location.Line1, wantParams.Location.Line1)
		assertEqual(t, gotParams.Location.CityStateZip, wantParams.Location.CityStateZip)
		assertEqual(t, gotParams.Location.MapURL, wantParams.Location.MapURL)

	})

	t.Run("decodes pre-encoded info session details link", func(t *testing.T) {
		var params rendererReqParams

		err := params.fromBase64("eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy9qLzEyMzQ1Njc4OTEwIiwiZGF0ZSI6IjIwMjItMTAtMDVUMTc6MDA6MDAuMDAwWiIsIm5hbWUiOiJGaXJzdE5hbWUiLCJsb2NhdGlvblR5cGUiOiJIWUJSSUQiLCJsb2NhdGlvbiI6eyJuYW1lIjoiU29tZSBQbGFjZSIsImxpbmUxIjoiMTIzIE1haW4gU3QiLCJjaXR5U3RhdGVaaXAiOiJDaXR5LCBTdGF0ZSAxMjM0NSIsIm1hcFVybCI6Imh0dHBzOi8vd3d3Lmdvb2dsZS5jb20vbWFwcy9wbGFjZS8xMjMrTWFpbitTdCwrQ2l0eSwrU3RhdGUrMTIzNDUifX0=")

		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, params.Name, "FirstName")
		assertEqual(t, params.Template, INFO_SESSION_TEMPLATE)
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
			SessionID:     "WpkB3jcw6gCw2uEMf",
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

		line1, cityStateZip := greenlight.ParseAddress(address)
		assertEqual(t, line1, "514 Franklin Ave")
		assertEqual(t, cityStateZip, "New Orleans, LA 70117")

	})

	t.Run("handles empty string", func(t *testing.T) {
		address := ""

		line1, cityStateZip := greenlight.ParseAddress(address)
		assertEqual(t, line1, "")
		assertEqual(t, cityStateZip, "")

	})

	t.Run("handles addresses with a street address only", func(t *testing.T) {
		address := "514 Franklin Ave"

		line1, cityStateZip := greenlight.ParseAddress(address)
		assertEqual(t, line1, "514 Franklin Ave")
		assertEqual(t, cityStateZip, "")

	})
}

func TestGoogleLocationLink(t *testing.T) {
	t.Run("returns the google maps link to address", func(t *testing.T) {
		address := "514 Franklin Ave, New Orleans, LA 70117, USA"

		assertEqual(t, greenlight.GoogleLocationLink(address), "https://www.google.com/maps/place/514+Franklin+Ave%2CNew+Orleans%2C+LA+70117")
	})

	t.Run("handles empty string", func(t *testing.T) {
		address := ""

		assertEqual(t, greenlight.GoogleLocationLink((address)), "")

	})

	t.Run("handles addresses with a street address only", func(t *testing.T) {
		address := "514 Franklin Ave"

		assertEqual(t, greenlight.GoogleLocationLink(address), "")

	})
}

func TestShortMessagingURL(t *testing.T) {
	t.Run("creates a user specific info session details URL", func(t *testing.T) {
		s := Signup{
			NameFirst:     "Yasiin",
			NameLast:      "Bey",
			Cell:          "555-555-5555",
			StartDateTime: mustMakeTime(t, time.RFC3339, "2022-03-14T17:00:00.000Z"),
			Cohort:        "is-mar-14-22-12pm",
			Email:         "yasiin@gmail.com",
			SessionID:     "WpkB3jcw6gCw2uEMf",
			LocationType:  "HYBRID",
			GooglePlace: greenlight.GooglePlace{
				Name:    "Some Place",
				Address: "2723 Guess Rd, Durham, NC 27705",
			},
			userJoinCode: "6421ecaa903dc77763e51829",
		}

		wantURLPrefix := "https://sms.operationspark.org/m/"

		// method under test
		gotURL, err := s.shortMessagingURL(os.Getenv("GREENLIGHT_HOST"))
		if err != nil {
			t.Fatal(err)
		}

		// the messaging URL should be prefixed with the passed in base URL
		if !strings.HasPrefix(gotURL, wantURLPrefix) {
			t.Fatalf("URL: %q doesn't have prefix: %q", wantURLPrefix, gotURL)
		}

		// grab the encoded info session details from the URL
		encoded := strings.TrimPrefix(gotURL, wantURLPrefix)

		// decode the params
		var gotParams rendererReqParams
		err = gotParams.fromBase64(encoded)
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, gotParams.Name, s.NameFirst)
		assertEqual(t, gotParams.Date.Equal(s.StartDateTime), true)
		assertEqual(t, gotParams.ZoomLink, s.zoomMeetingURL)
		assertEqual(t, gotParams.Template, INFO_SESSION_TEMPLATE)
		assertEqual(t, gotParams.LocationType, "HYBRID")
		assertEqual(t, gotParams.Location.Name, "Some Place")
		assertEqual(t, gotParams.Location.Line1, "2723 Guess Rd")
		assertEqual(t, gotParams.Location.CityStateZip, "Durham, NC 27705")
		assertEqual(t, gotParams.Location.MapURL, "https://www.google.com/maps/place/2723+Guess+Rd%2CDurham%2C+NC+27705")
		// should be true because "gmail.com" should be the signup's email address domain
		assertEqual(t, gotParams.IsGmail, true)
		assertEqual(t, gotParams.GreenlightURL, "https://greenlight.operationspark.org/sessions/WpkB3jcw6gCw2uEMf/?subview=overview&userJoinCode=6421ecaa903dc77763e51829")

	})
}

func TestCreateMessageURL(t *testing.T) {
	t.Run("creates a message URL with base64 encoded details", func(t *testing.T) {
		mardiGras, err := time.Parse("Jan 02, 2006", "Feb 21, 2023") // Mardi Gras
		require.NoError(t, err)

		osLoc := Location{
			Name:         "Operation Spark",
			Line1:        "514 Franklin Av",
			CityStateZip: "New Orleans, LA 70117",
			MapURL:       "https://testmapurl.google.com",
		}

		r := osRenderer{}
		person := gofakeit.Person()
		p := notify.Participant{
			NameFirst:           person.FirstName,
			NameLast:            person.LastName,
			FullName:            person.FirstName + " " + person.LastName,
			Cell:                person.Contact.Phone,
			Email:               person.Contact.Email,
			ZoomJoinURL:         notify.MustFakeZoomURL(t),
			SessionDate:         mardiGras,
			SessionLocationType: "HYBRID",
			SessionLocation:     notify.Location(osLoc),
		}
		msgURL, err := r.CreateMessageURL(p)
		require.NoError(t, err)
		// Make sure location data is encoded in the URL
		u, err := url.Parse(msgURL)
		require.NoError(t, err)

		// Decode the base64 encoded data from the generated URL
		encoded := strings.TrimPrefix(u.Path, "/m/")
		d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
		decodedJson, err := io.ReadAll(d)
		require.NoError(t, err)

		// Unmarshal the decoded JSON into a messaging request params struct
		var params rendererReqParams
		jd := json.NewDecoder(bytes.NewReader(decodedJson))
		err = jd.Decode(&params)
		require.NoError(t, err)

		// Verify the location data matches the input from the Participant
		require.Equal(t, "HYBRID", params.LocationType)
		require.Equal(t, osLoc, params.Location)
	})

	t.Run("ok even without a Location", func(t *testing.T) {
		mardiGras, err := time.Parse("Jan 02, 2006", "Feb 21, 2023") // Mardi Gras
		require.NoError(t, err)

		person := gofakeit.Person()
		p := notify.Participant{
			NameFirst:           person.FirstName,
			NameLast:            person.LastName,
			FullName:            person.FirstName + " " + person.LastName,
			Cell:                person.Contact.Phone,
			Email:               person.Contact.Email,
			ZoomJoinURL:         notify.MustFakeZoomURL(t),
			SessionDate:         mardiGras,
			SessionLocationType: "VIRTUAL",
			// Empty location for VIRTUAL types
			SessionLocation: notify.Location{},
		}

		r := osRenderer{}
		msgURL, err := r.CreateMessageURL(p)
		require.NoError(t, err)
		// Make sure location data is encoded in the URL
		u, err := url.Parse(msgURL)
		require.NoError(t, err)

		// Decode the base64 encoded data from the generated URL
		encoded := strings.TrimPrefix(u.Path, "/m/")
		d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
		jsonBytes, err := io.ReadAll(d)
		require.NoError(t, err)

		require.True(t, bytes.Contains(jsonBytes, []byte(`"locationType":"VIRTUAL"`)), "decoded JSON should contain the VIRTUAL location type")
	})

}
