package signups

import (
	"testing"
	"time"
)

func TestWelcomeData(t *testing.T) {
	sessionStart, _ := time.Parse(time.RFC3339, "2022-02-28T18:00:00.000Z")
	tests := []struct {
		name   string
		signup Signup
		want   WelcomeValues
	}{
		{
			name: "UTC 18 converted to noon central",
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
				SessionTime: "6:00 PM CDT",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.signup.WelcomeData()
			if got.SessionTime != test.want.SessionTime {
				t.Errorf("s.WelcomeData():\ns.StartDateTime:%v\nwant:%s\ngot:%s", test.signup.StartDateTime, test.want.SessionTime, got.SessionTime)
			}
		})
	}
}
