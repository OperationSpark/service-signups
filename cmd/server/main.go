package main

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	signup "github.com/operationspark/service-signup"
)

func main() {
	ctx := context.Background()
	err := funcframework.RegisterHTTPFunctionContext(ctx, "/", signup.NewServer().ServeHTTP)
	if err != nil {
		log.Fatalf("funcframework.RegisterHTTPFunctionContext: %v\n", err)
	}

	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	log.Printf("server starting on port: %s\n", port)
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
