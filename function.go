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
	// Set up services/tasks to run when someone signs up for an Info Session.
	mgDomain := os.Getenv("MAIL_DOMAIN")
	mgAPIKey := os.Getenv("MAILGUN_API_KEY")
	mgSvc := NewMailgunService(mgDomain, mgAPIKey, "")

	glWebhookURL := os.Getenv("GREENLIGHT_WEBHOOK_URL")
	glAPIkey := os.Getenv("GREENLIGHT_API_KEY")
	glSvc := NewGreenlightService(glWebhookURL, glAPIkey)

	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	slackSvc := NewSlackService(slackWebhookURL)

	zoomAccountID := os.Getenv("ZOOM_ACCOUNT_ID")
	zoomClientID := os.Getenv("ZOOM_CLIENT_ID")
	zoomClientSecret := os.Getenv("ZOOM_CLIENT_SECRET")
	zoomMeeting12 := os.Getenv("ZOOM_MEETING_12")
	zoomMeeting17 := os.Getenv("ZOOM_MEETING_17")

	zoomSvc := NewZoomService(ZoomOptions{
		clientID:     zoomClientID,
		clientSecret: zoomClientSecret,
		accountID:    zoomAccountID,
	})

	registrationService := newSignupService(
		signupServiceOptions{
			meetings: map[int]string{
				12: zoomMeeting12,
				17: zoomMeeting17,
			},
			// registering the user for the Zoom meeting,
			zoomService: zoomSvc,
			// Registration tasks:
			// (executed serially)
			tasks: []task{
				// posting a WebHook to Greenlight,
				glSvc,
				// sending a "Welcome Email",
				mgSvc,
				// sending a Slack message to #signups channel,
				slackSvc,

				// TODO:
				// sending an SMS confirmation message to the user.
			},
		},
	)

	server := newSignupServer(registrationService)
	return server
}
