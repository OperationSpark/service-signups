package main

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	signup "github.com/operationspark/service-signup"
)

func main() {
	mgDomain := os.Getenv("MAIL_DOMAIN")
	mgAPIKey := os.Getenv("MAIL_GUN_PRIVATE_API_KEY")
	glWebhookURL := os.Getenv("GREENLIGHT_WEBHOOK_URL")
	glAPIkey := os.Getenv("GREENLIGHT_API_KEY")
	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	// Set up services/tasks to run when someone signs up for an Info Session.
	mgSvc := signup.NewMailgunService(mgDomain, mgAPIKey, "")
	glSvc := signup.NewGreenlightService(glWebhookURL, glAPIkey)
	slackSvc := signup.NewSlackService(slackWebhookURL)

	// These registration tasks include:
	registrationService := signup.NewSignupService(
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

	server := signup.NewSignupServer(registrationService)

	ctx := context.Background()
	if err := funcframework.RegisterHTTPFunctionContext(ctx, "/", server.HandleSignUp); err != nil {
		log.Fatalf("funcframework.RegisterHTTPFunctionContext: %v\n", err)
	}
	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
