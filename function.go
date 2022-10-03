package signup

import (
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	// Register an HTTP function with the Functions Framework
	functions.HTTP("HandleSignup", makeEntryPoint())
}

func makeEntryPoint() http.HandlerFunc {
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
	registrationService := NewSignupService(
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

	server := NewSignupServer(registrationService)

	return server.HandleSignUp
}
