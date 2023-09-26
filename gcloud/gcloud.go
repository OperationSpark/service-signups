package gcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"google.golang.org/api/idtoken"
)

// MakeAuthenticatedReq makes an HTTP request using Google Service Account credentials.
// https://cloud.google.com/run/docs/authenticating/service-to-service#acquire-token
func MakeAuthenticatedReq(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	audience := url
	creds := os.Getenv("GCP_SA_CREDS_JSON")
	opts := idtoken.WithCredentialsJSON([]byte(creds))

	if creds == "" {
		opts = idtoken.WithCredentialsFile("../creds.json")
	}
	ts, err := idtoken.NewTokenSource(ctx, audience, opts)
	if err != nil {
		return nil, fmt.Errorf("newTokenSource: %w", err)
	}
	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, audience, body)
	token.SetAuthHeader(req)
	return req, err
}
