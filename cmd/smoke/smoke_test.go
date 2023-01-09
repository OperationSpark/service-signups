package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmokeSignup(t *testing.T) {
	if !checkTestsEnabled() {
		workflowName := os.Getenv("GITHUB_WORKFLOW")
		t.Skip("Smoke Test disable for workflow: %w", workflowName)
	}

	s := smoke{
		glAPIurl:     "https://greenlight.operationspark.org/api",
		signupAPIurl: "https://us-central1-operationspark-org.cloudfunctions.net/session-signups",
		toNum:        os.Getenv("TEST_TO_NUM"),
	}

	//  Fetch Info Session ID from Greenlight endpoint
	err := s.fetchInfoSessions()
	require.NoError(t, err, "fetching open sessions")

	// TODO: POST Signup to production endpoint
	// TODO: Get expected SMS message from Twilio API
	// TODO: Parse info link in SMS and visit it
	// TODO: Ensure rendered page has expected content
}

// CheckTestsEnabled checks we're on the "CI" workflow. If so, returns false, otherwise returns true. We only want these tests to run after a successful deployment.
func checkTestsEnabled() bool {
	workflowName := os.Getenv("GITHUB_WORKFLOW")
	return strings.ToUpper(workflowName) != "CI"
}
