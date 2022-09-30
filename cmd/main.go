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
	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	mgSvc := signup.NewMailgunService(mgDomain, mgAPIKey, "")
	glSvc := signup.NewGreenlightService(glWebhookURL)
	slackSvc := signup.NewSlackService(slackWebhookURL)
	registrationService := signup.NewSignupService(mgSvc, glSvc, slackSvc)

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
