package signup

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/operationspark/service-signup/conversations"
	"github.com/operationspark/service-signup/mongodb"
	"github.com/operationspark/service-signup/notify"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	// GCP Cloud Function requires the dreaded init() call to register the handler with the functions-framework. Init() is also unnecessarily called in test mode. Init() in turns calls NewServer() which needs to connect to MongoDB. To avoid that, I have this StubStore that implements the Store interface, but does nothing. It is only used in test mode to prevent the MongoDB connection error.
	StubStore struct{}
)

// Implement the Store interface
func (s *StubStore) GetUpcomingSessions(context.Context, time.Duration) ([]*notify.UpcomingSession, error) {
	return []*notify.UpcomingSession{}, nil
}

func init() {
	// Register an HTTP function with the Functions Framework
	// This handler name maps to the entry point name in the Google Cloud Function platform.
	// https://cloud.google.com/functions/docs/writing/write-http-functions
	functions.HTTP("HandleSignUp", NewServer().ServeHTTP)
}

func NewServer() *http.ServeMux {
	sentryDSN := os.Getenv("SENTRY_DSN")
	err := sentry.Init(sentry.ClientOptions{
		Dsn:           sentryDSN,
		EnableTracing: false,
	})
	if err != nil {
		log.Fatalf("sentry init: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	// Check env vars only in GCP context
	// K_REVISION is set in GCP environment, so if it's not set, we're not running in GCP and we can skip the check
	// https://cloud.google.com/functions/docs/configuring/env-var#newer_runtimes
	skipEnvVarCheck := os.Getenv("K_REVISION") == ""
	err = checkEnvVars(skipEnvVarCheck)
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger = logger.With("git_hash", getGitRev())

	mux := http.NewServeMux()
	sentryHandler := sentryhttp.New(sentryhttp.Options{})
	mux.HandleFunc("/", sentryHandler.HandleFunc(NewSignupServer(logger).HandleSignUp))
	mux.HandleFunc("/notify", sentryHandler.HandleFunc(NewNotifyServer().ServeHTTP))
	return mux
}

func checkEnvVars(skip bool) error {
	if skip {
		return nil
	}
	requiredEnvVars := []string{
		"GREENLIGHT_API_KEY",
		"GREENLIGHT_WEBHOOK_URL",
		"MAIL_DOMAIN",
		"MAILGUN_API_KEY",
		"MONGO_URI",
		"OS_MESSAGING_SERVICE_URL",
		"OS_MESSAGING_SIGNING_SECRET",
		"OS_RENDERING_SERVICE_URL",
		"SLACK_WEBHOOK_URL",
		"TWILIO_ACCOUNT_SID",
		"TWILIO_AUTH_TOKEN",
		"TWILIO_CONVERSATIONS_SID",
		"TWILIO_PHONE_NUMBER",
		"URL_SHORTENER_API_KEY",
		"ZOOM_ACCOUNT_ID",
		"ZOOM_CLIENT_ID",
		"ZOOM_CLIENT_SECRET",
		"ZOOM_MEETING_12",
		"ZOOM_MEETING_17",
	}

	for _, ev := range requiredEnvVars {
		if os.Getenv(ev) == "" {
			return fmt.Errorf("env var %q is required", ev)
		}
	}
	return nil
}

func getMongoClient() (*mongo.Client, string, error) {
	mongoURI := os.Getenv("MONGO_URI")
	isCI := os.Getenv("CI") == "true"
	parsed, err := url.Parse(mongoURI)
	if isCI || (mongoURI == "" || err != nil) {
		return nil, "", fmt.Errorf("invalid 'MONGO_URI' environmental variable: %q", mongoURI)
	}
	dbName := strings.TrimPrefix(parsed.Path, "/")
	m, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	return m, dbName, err
}

func NewNotifyServer() *notify.Server {
	mongoURI := os.Getenv("MONGO_URI")
	isCI := os.Getenv("CI") == "true"
	parsed, err := url.Parse(mongoURI)
	if isCI || (mongoURI == "" || err != nil) {
		fmt.Printf("Invalid 'MONGO_URI' environmental variable: %q\n", mongoURI)
		fmt.Printf("If you're running tests, you can ignore this message.\n\n")
		// See StubStore comment above
		// **  This server is never used ** //
		return notify.NewServer(notify.ServerOpts{
			Store: &StubStore{},
		})
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %q", mongoURI)
	}
	mongoService := notify.NewMongoService(mongoClient, dbName)

	// TODO: Should we just use the once instance of a Twilio service?
	twilioSvc := NewTwilioService(twilioServiceOptions{
		accountSID:                 os.Getenv("TWILIO_ACCOUNT_SID"),
		authToken:                  os.Getenv("TWILIO_AUTH_TOKEN"),
		fromPhoneNum:               os.Getenv("TWILIO_PHONE_NUMBER"),
		conversationsSid:           os.Getenv("TWILIO_CONVERSATIONS_SID"),
		opSparkMessagingSvcBaseURL: os.Getenv("OS_MESSAGING_SERVICE_URL"),
	})

	return notify.NewServer(notify.ServerOpts{
		OSRendererService: &osRenderer{baseURL: os.Getenv("OS_RENDERING_SERVICE_URL")},
		Store:             mongoService,
		SMSService:        twilioSvc,
		ShortLinkService:  NewURLShortener(ShortenerOpts{apiKey: os.Getenv("URL_SHORTENER_API_KEY")}),
	})
}

func NewSignupServer(logger *slog.Logger) *signupServer {
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
	osMessagingSigningSecret := os.Getenv("OS_MESSAGING_SIGNING_SECRET")

	twilioSvc := NewTwilioService(twilioServiceOptions{
		accountSID:                 twilioAcctSID,
		authToken:                  twilioAuthToken,
		fromPhoneNum:               twilioPhoneNum,
		opSparkMessagingSvcBaseURL: osMessagingSvcURL,
		conversationsSid:           twilioConversationsSid,
	})

	mongoClient, dbName, err := getMongoClient()
	if err != nil && os.Getenv("CI") != "true" {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	}

	gldbService := mongodb.New(dbName, mongoClient)
	snapMailURL := os.Getenv("SNAP_MAIL_URL")
	snapMailSvc := NewSnapMail(snapMailURL, WithSigningSecret(os.Getenv("SIGNING_SECRET")))

	convoLinkSvc := conversations.NewService(
		conversations.WithMessengerAPIBase(osMessagingSvcURL+"/api/v0"),
		conversations.WithSigningSecret(osMessagingSigningSecret),
	)

	logger = logger.With("service", "signup")

	registrationService := newSignupService(
		signupServiceOptions{
			meetings: map[int]string{
				12: zoomMeeting12,
				17: zoomMeeting17,
			},
			// registering the user for the Zoom meeting,
			zoomService: zoomSvc,
			gldbService: gldbService,
			// Registration tasks:
			// (executed concurrently)
			tasks: []mutationTask{
				// posting a WebHook to Greenlight,
				glSvc,
				// sending a "Welcome Email",
				mgSvc,
				// sending a Slack message to #signups channel,
				slackSvc,
				// sending an SMS confirmation message to the user.
				twilioSvc,
				// sending Signup message to SNAP mail application
				snapMailSvc,
			},
			postSignupTasks: []Runner{convoLinkSvc},
			logger:          logger,
		},
	)

	return &signupServer{
		service: registrationService,
		logger:  logger,
	}
}

func getGitRev() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if !ok {
			return ""
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}
