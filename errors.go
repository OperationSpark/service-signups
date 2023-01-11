package signup

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type InvalidFieldError struct {
	Field string
}

func (e *InvalidFieldError) Error() string {
	return fmt.Sprintf("invalid value for field: '%s'", e.Field)
}

// handleHTTPError parses and prints the response body. It expects the response body to contain JSON and returns an error if the body in HTML.
func handleHTTPError(resp *http.Response) error {
	reqLabel := fmt.Sprintf(
		"%s: %s://%s\n%s\n",
		resp.Request.Method,
		resp.Request.URL.Scheme,
		resp.Request.URL.Host,
		resp.Request.URL.RequestURI(),
	)

	errMsg := fmt.Sprintf("HTTP Error:\n%s\nResponse:\n%s", reqLabel, resp.Status)

	// Ignore response body if it's HTML to avoid flooding the logs
	isHTML := strings.Contains(resp.Header.Get("Content-Type"), "text/html")
	if isHTML {
		return fmt.Errorf("%s\n[HTML response removed]", errMsg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("response code: %s", resp.Status)
	}

	return fmt.Errorf("%s\n%s", errMsg, body)
}
