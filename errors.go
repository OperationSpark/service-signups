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
	return fmt.Errorf("HTTP: %s\nresponse body: %s", resp.Status, string(body))
}