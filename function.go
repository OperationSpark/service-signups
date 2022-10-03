package signup

import (
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	// Register an HTTP function with the Functions Framework
	// This handler name maps to the entry point name in the Google Cloud Function platform.
	// https://cloud.google.com/functions/docs/writing/write-http-functions
	functions.HTTP("HandleSignUp", NewServer().HandleSignUp)
}

func NewServer() *signupServer {
	mgDomain := os.Getenv("MAIL_DOMAIN")
	mgAPIKey := os.Getenv("MAILGUN_API_KEY")
	glWebhookURL := os.Getenv("GREENLIGHT_WEBHOOK_URL")
	glAPIkey := os.Getenv("GREENLIGHT_API_KEY")
	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	// Set up services/tasks to run when someone signs up for an Info Session.
	mgSvc := NewMailgunService(mgDomain, mgAPIKey, "")
	glSvc := NewGreenlightService(glWebhookURL, glAPIkey)
	slackSvc := NewSlackService(slackWebhookURL)

	// These registration tasks include:
	registrationService := newSignupService(
		// posting a WebHook to Greenlight,
		glSvc,
		// sending a "Welcome Email",
		mgSvc,
		// sending a Slack message to #signups channel,
		slackSvc,
		// TODO:
		// registering the user for the Zoom meeting,
		// sending an SMS confirmation message to the user.
	)

	server := newSignupServer(registrationService)
	return server
}
