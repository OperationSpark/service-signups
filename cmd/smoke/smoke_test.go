package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	signup "github.com/operationspark/service-signup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmokeSignup(t *testing.T) {
	if !checkTestsEnabled() {
		workflowName := os.Getenv("GITHUB_WORKFLOW")
		t.Skip("Smoke Test disable for workflow: %w", workflowName)
	}

	s := newSmokeTest()

	// Fetch Info Session ID from Greenlight endpoint
	err := s.fetchInfoSessions()
	require.NoError(t, err, "fetching open sessions")

	// POST Signup to production endpoint
	su := signup.Signup{
		AttendingLocation: "IN_PERSON",
		Cell:              s.toNum,
		Cohort:            s.selectedSession.Cohort,
		Email:             s.toEmail,
		GooglePlace:       s.selectedSession.GooglePlace,
		LocationType:      s.selectedSession.LocationType,
		JoinCode:          s.selectedSession.Code,
		NameFirst:         "Halle",
		NameLast:          "Bot",
		ProgramID:         s.selectedSession.ProgramID,
		Referrer:          "verbal",
		ReferrerResponse:  "Automated Smoke Test",
		SessionID:         s.selectedSession.ID,
		SMSOptIn:          true,
		StartDateTime:     s.selectedSession.Times.Start.DateTime,
		UserLocation:      "South Dakota",
	}

	// Conditionally set AttendingLocation based on LocationType
	if su.LocationType == "HYBRID" || su.LocationType == "VIRTUAL" {
		su.AttendingLocation = "VIRTUAL"
	}

	// TODO: Also test "None of these fit my schedule" option
	err = s.postSignup(su)
	require.NoError(t, err, "POST Signup to Cloud Function")

	// TODO: ** Fetch last two messages. As of this commit, Twilio is blocking our opt-in message with error 30034. They will block some number of messages until our A2P 10DLC campaign is approved. In meantime, assume the opt-in confirmation message will be blocked and the last delivered message will be the Info Session confirmation. ***
	// Get the last 2 text messages from Twilio API
	// The first should be the Opt-in confirmation
	// The second should be the Info Session confirmation with the short link.
	expectedMessageAmt := 1
	msgs, err := fetchLastTextMessages(s.toNum, s.fromNum, expectedMessageAmt)
	require.NoError(t, err)
	require.NotEmptyf(t, msgs, "no messages found sent from %q -> %q", s.fromNum, s.toNum)
	require.Len(t, msgs, expectedMessageAmt, "Expected %d text messages sent", expectedMessageAmt)

	// TODO: Check opt-in message once campaign approved. See above comment.
	// textOptInMsg := msgs[0]
	// require.Containsf(t, textOptInMsg, "opt", "expected an opt-in confirmation containing the word 'opt'\ngot:\n%q", textOptInMsg)

	infoSessionConfirmation := msgs[0]
	// Parse info link in SMS
	link := parseSMSShortLink(infoSessionConfirmation)
	// Intentionally using assert to continue running tests even if shortener fails
	assert.NotEmpty(t, link, "URL Shortener service failed\nSMS: %q", infoSessionConfirmation)

	if link == "" {
		// Shortener service failed, get the long link
		link = parseSMSOriginalLink(infoSessionConfirmation)
	}

	// Visit link
	body, err := fetchLinkBody(link)
	require.NoError(t, err)

	// Ensure rendered page has expected content
	ct, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	infoHTMLtargets := []string{
		// Session Date
		s.selectedSession.Times.Start.DateTime.In(ct).Format("Monday, January 2, 2006"),
		// Session Time
		s.selectedSession.Times.Start.DateTime.In(ct).Format("3:00pm (MST)"),
		// Name
		su.NameFirst,
		//xJoin Code
		s.selectedSession.Code,
		// TODO: HTML only contains the next props, so "Hello Halle," not rendered yet.
		// Zoom link
		"https://us06web.zoom.us/w/8", //...
	}

	// conditionally check for location information
	if s.selectedSession.LocationType == "HYBRID" || s.selectedSession.LocationType == "IN_PERSON" {
		// Google Map link
		infoHTMLtargets = append(infoHTMLtargets,
			"https://www.google.com/maps/place/514+Franklin+Ave%2CNew+Orleans%2C+LA+70117",
		)
	}

	err = checkInfoPageContent(body,
		infoHTMLtargets...,
	)
	require.NoErrorf(t, err, "Info URL: %s", link)
}

func TestLinkExtractors(t *testing.T) {
	t.Run("extracts short link from SMS", func(t *testing.T) {
		sms := `You've signed up for an info session with Operation Spark! The session is Mon Jan 09 @ 12:00p CST. View this link for details: https://ospk.org/lztvCQIvUK`

		want := "https://ospk.org/lztvCQIvUK"
		got := parseSMSShortLink(sms)
		require.Equal(t, want, got)
	})

	t.Run("extracts long link when shortener service fails", func(t *testing.T) {
		sms := `You've signed up for an info session with Operation Spark! The session is Mon Jan 09 @ 12:00p CST. View this link for details: https://sms.operationspark.org/m/eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy93L0FBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBIiwiZGF0ZSI6IjIwMjMtMDEtMDJUMTg6MDA6MDBaIiwibmFtZSI6IkhhbGxlIiwibG9jYXRpb25UeXBlIjoiSFlCUklEIiwibG9jYXRpb24iOnsibmFtZSI6Ik9wZXJhdGlvbiBTcGFyayIsImxpbmUxIjoiNTE0IEZyYW5rbGluIEF2ZSIsImNpdHlTdGF0ZVppcCI6Ik5ldyBPcmxlYW5zLCBMQSA3MDExNyIsIm1hcFVybCI6Imh0dHBzOi8vd3d3Lmdvb2dsZS5jb20vbWFwcy9wbGFjZS81MTQrRnJhbmtsaW4rQXZlJTJDTmV3K09ybGVhbnMlMkMrTEErNzAxMTcifX0=`

		want := "https://sms.operationspark.org/m/eyJ0ZW1wbGF0ZSI6IkluZm9TZXNzaW9uIiwiem9vbUxpbmsiOiJodHRwczovL3VzMDZ3ZWIuem9vbS51cy93L0FBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBIiwiZGF0ZSI6IjIwMjMtMDEtMDJUMTg6MDA6MDBaIiwibmFtZSI6IkhhbGxlIiwibG9jYXRpb25UeXBlIjoiSFlCUklEIiwibG9jYXRpb24iOnsibmFtZSI6Ik9wZXJhdGlvbiBTcGFyayIsImxpbmUxIjoiNTE0IEZyYW5rbGluIEF2ZSIsImNpdHlTdGF0ZVppcCI6Ik5ldyBPcmxlYW5zLCBMQSA3MDExNyIsIm1hcFVybCI6Imh0dHBzOi8vd3d3Lmdvb2dsZS5jb20vbWFwcy9wbGFjZS81MTQrRnJhbmtsaW4rQXZlJTJDTmV3K09ybGVhbnMlMkMrTEErNzAxMTcifX0="
		got := parseSMSOriginalLink(sms)
		require.Equal(t, want, got)
	})
}

func TestCheckInfoPageContent(t *testing.T) {
	t.Run("returns no error if all the target strings are found", func(t *testing.T) {
		html := `<html><body>Henri</body></html>`
		err := checkInfoPageContent(strings.NewReader(html), "Henri")
		require.NoError(t, err)
	})

	t.Run("returns an error if any of the targets are missing from the HTML body", func(t *testing.T) {
		html := `<html><body>Henri</body></html>`
		err := checkInfoPageContent(strings.NewReader(html),
			"Henri",
			// Should be missing
			"Today",
		)
		require.Errorf(t, err, `target "Today" is missing from body and should produce an error`)
	})
}

// CheckTestsEnabled checks if the 'SMOKE_LIVE' env var is explicitly set to "true". If so, returns true, otherwise returns false. We only want these tests to run after a successful deployment.
func checkTestsEnabled() bool {
	isSmokeLive, err := strconv.ParseBool(os.Getenv("SMOKE_LIVE"))
	if err != nil || !isSmokeLive {
		if err != nil {
			fmt.Print("could not parse 'SMOKE_LIVE' env var")
		}
		fmt.Println("smoke test skipped.\nSet `SMOKE_LIVE=true` to run smoke test")
		return false
	}
	return isSmokeLive
}
