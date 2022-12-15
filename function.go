package signup

import (
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/operationspark/service-signup/notify"
)

func init() {
	// Register an HTTP function with the Functions Framework
	// This handler name maps to the entry point name in the Google Cloud Function platform.
	// https://cloud.google.com/functions/docs/writing/write-http-functions
	functions.HTTP("HandleSignUp", NewServer().ServeHTTP)
}

func NewServer() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", NewSignupServer().HandleSignUp)
	mux.HandleFunc("/notify", NewNotifyServer().ServeHTTP)
	return mux
}

func NewNotifyServer() *notify.Server {
	mongoURI := os.Getenv("MONGO_URI")
	twilioAcctSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioPhoneNum := os.Getenv("TWILIO_PHONE_NUMBER")
	twilioConversationsSid := os.Getenv("TWILIO_CONVERSATIONS_SID")

	// TODO: Should we just use the once instance of a Twilio service?
	twilioSvc := NewTwilioService(twilioServiceOptions{
		accountSID:       twilioAcctSID,
		authToken:        twilioAuthToken,
		fromPhoneNum:     twilioPhoneNum,
		conversationsSid: twilioConversationsSid,
	})

	return notify.NewServer(notify.ServerOpts{
		MongoURI:      mongoURI,
		TwilioService: twilioSvc,
	})
}

func NewSignupServer() *signupServer {
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

	twilioAcctSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioPhoneNum := os.Getenv("TWILIO_PHONE_NUMBER")
	twilioConversationsSid := os.Getenv("TWILIO_CONVERSATIONS_SID")

	osMessagingSvcURL := os.Getenv("OS_MESSAGING_SERVICE_URL")

	twilioSvc := NewTwilioService(twilioServiceOptions{
		accountSID:                 twilioAcctSID,
		authToken:                  twilioAuthToken,
		fromPhoneNum:               twilioPhoneNum,
		opSparkMessagingSvcBaseURL: osMessagingSvcURL,
		conversationsSid:           twilioConversationsSid,
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
			// (executed concurrently)
			tasks: []Task{
				// posting a WebHook to Greenlight,
				glSvc,
				// sending a "Welcome Email",
				mgSvc,
				// sending a Slack message to #signups channel,
				slackSvc,
				// sending an SMS confirmation message to the user.
				twilioSvc,
			},
		},
	)

	return &signupServer{registrationService}
}
