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

// HandleHTTPError parses and prints the response body.
func handleHTTPError(resp *http.Response) error {
	// Ignore response body if it's HTML to avoid flooding the logs
	isHTML := strings.Contains(resp.Header.Get("Content-Type"), "text/html")
	if isHTML {
		return fmt.Errorf("HTTP: %s\n[HTML response removed]", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("response code: %s", resp.Status)
	}
	return fmt.Errorf("HTTP: %s\nresponse body: %s", resp.Status, body)
}
