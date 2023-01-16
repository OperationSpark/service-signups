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
		NameFirst:         "Halle",
		NameLast:          "Bot",
		ProgramID:         s.selectedSession.ProgramID,
		Referrer:          "verbal",
		ReferrerResponse:  "Automated Smoke Test",
		SessionID:         s.selectedSession.ID,
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

	// Get expected SMS message from Twilio API
	sms, err := fetchSMSmessage(s.toNum, s.fromNum)
	require.NoError(t, err)
	require.NotEmptyf(t, sms, "no message found sent from %q -> %q", s.fromNum, s.toNum)

	// Parse info link in SMS
	link := parseSMSShortLink(sms)
	// Intentionally using assert to continue running tests even if shortener fails
	assert.NotEmpty(t, link, "URL Shortener service failed\nSMS: %q", sms)

	if link == "" {
		// Shortener service failed, get the long link
		link = parseSMSOriginalLink(sms)
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
		// TODO: HTML only contains the next props, so "Hello Halle," not rendered yet.
		// Zoom link
		"https://us06web.zoom.us/w/8", //...
	}
	// conditionally check for location infomation
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
	t.Run("returns no error if all the target strigns are found", func(t *testing.T) {
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
	if err != nil {
		fmt.Println("could not parse 'SMOKE_LIVE' env var")
		return false
	}
	return isSmokeLive
}
