package signups

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWelcomeData(t *testing.T) {
	sessionStart, _ := time.Parse(time.RFC3339, "2022-03-21T22:30:00.000Z")
	tests := []struct {
		name   string
		signup Signup
		want   WelcomeValues
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
			want: WelcomeValues{
				DisplayName: "Henri",
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
			want: WelcomeValues{
				DisplayName: "Henri",
				SessionDate: "",
				SessionTime: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.signup.WelcomeData()
			if err != nil {
				t.Errorf("Unexpected error for input date %v.\n%v", test.signup.StartDateTime, err)
			}
			if got.SessionTime != test.want.SessionTime {
				t.Errorf("s.WelcomeData():\ns.StartDateTime:%v\nwant:%s\ngot:%s", test.signup.StartDateTime, test.want.SessionTime, got.SessionTime)
			}
		})
	}
}

func TestHTML(t *testing.T) {
	sessionStartDate, _ := time.Parse(time.RFC822, "02 Feb 22 15:00 UTC")
	tests := []struct {
		s    Signup
		want []string
	}{
		{
			s: Signup{
				NameFirst:     "Tariq",
				NameLast:      "Trotter",
				StartDateTime: sessionStartDate,
			},
			want: []string{"Tariq", "Wednesday, Feb 02", "9:00 AM CST"},
		},
		{
			s:    Signup{NameFirst: "Amir", NameLast: "Thompson", StartDateTime: time.Time{}},
			want: []string{"Amir", "we don't have any info session times to fit your"},
		},
	}

	for _, test := range tests {
		var b bytes.Buffer
		err := test.s.Html(&b)

		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		hiIndex := strings.Index(b.String(), "Hi ")
		got := b.Bytes()[hiIndex : hiIndex+500]
		for _, expected := range test.want {
			if !strings.Contains(b.String(), expected) {
				t.Fatalf("string missing from rendered HTML\nwant: \"...%s...\"\ngot:\n %s\nSignup:%+v\n", expected, got, test.s)
			}

		}
	}
}

func TestSummary(t *testing.T) {
	sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 18:00 UTC")
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

func TestRenderedHtml(t *testing.T) {
	sessionStart, _ := time.Parse(time.RFC3339, "2022-03-21T22:30:00.000Z")
	test := Signup{
		ProgramId:        "",
		NameFirst:        "Peter",
		NameLast:         "Barnum",
		Email:            "the-tester@email.com",
		Cell:             "555-123-4567",
		Referrer:         "Word of mouth",
		ReferrerResponse: "JaneDoe Smith",
		StartDateTime:    sessionStart,
		Cohort:           "is-feb-28-22-12pm",
	}

	var b bytes.Buffer
	err := test.Html(&b)

	if err != nil {
		fmt.Println("Bytes", b)
	}

	renderedPath, err := filepath.Abs(filepath.Join(".", "signup_template.ignore.html"))

	f, err := os.Create(renderedPath)
	if err != nil {
		t.Fatalf("Should create file 'signup_template.ignore.html'\n  > %v", renderedPath)
	}

	n, err := f.WriteString(b.String())
	if err != nil || n == 0 {
		t.Fatal("Should write to file 'signup_template.ignore.html'")
	}

}
