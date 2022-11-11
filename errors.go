package signup

import (
	"fmt"
	"io"
	"net/http"
)

type InvalidFieldError struct {
	Field string
}

func (e *InvalidFieldError) Error() string {
	return fmt.Sprintf("invalid value for field: '%s'", e.Field)
}

// HandleHTTPError parses and prints the response body.
func handleHTTPError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("response Body: %s", resp.Status)
	}

	reqLabel := fmt.Sprintf(
		"%s: %s://%s\n%s\n",
		resp.Request.Method,
		resp.Request.URL.Scheme,
		resp.Request.URL.Host,
		resp.Request.URL.RequestURI(),
	)
	return fmt.Errorf("HTTP Error:\n%s\nResponse:\n%s\n%s", reqLabel, resp.Status, string(body))
}
