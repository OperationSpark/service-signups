package signup

import (
	"testing"
	"time"
)

func TestSendSMS(t *testing.T) {
	t.Run("sends an SMS message to the signed up user", func(t *testing.T) {
		accountSid := "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		authToken := "YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"
		fromPhoneNum := "+15041234567"
		tSvc := NewTwilioService(accountSid, authToken, fromPhoneNum)

		sessionStartDate, _ := time.Parse(time.RFC822, "14 Mar 22 17:00 UTC")
		su := Signup{
			NameFirst:        "Bob",
			NameLast:         "Ross",
			Email:            "bross@pbs.org",
			Cell:             `json:"cell" schema:"cell"`,
			Referrer:         "instagram",
			ReferrerResponse: "",
			StartDateTime:    sessionStartDate,
			Cohort:           "is-mar-14-22-12pm",
			SessionId:        "X5TsABhN94yesyMEi",
		}

		err := tSvc.sendSMS(su)
		if err != nil {
			t.Fatalf("twilio service: sendSMS: %v", err)
		}

	})
}
