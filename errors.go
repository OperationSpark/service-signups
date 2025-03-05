package signup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type (
	errorResponse struct {
		Error any `json:"error"`
	}

	InvalidFieldError struct {
		Field string
	}
)

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

func (ss *signupServer) logError(ctx context.Context, r *http.Request, err error) {
	method := r.Method
	url := r.URL.String()

	ss.logger.ErrorContext(
		ctx,
		err.Error(),
		slog.String("method", method),
		slog.String("url", url))
}

// errorResponse writes an error response to the client. The msg is sent as the error message in the response body,
// so it should be a human-readable message and not leak any sensitive information.
func (ss *signupServer) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	respData := errorResponse{Error: msg}
	err := ss.writeJSON(w, status, respData)
	if err != nil {
		ss.logError(r.Context(), r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse logs the error and sends a generic 500 Internal Server Error response to the client.
func (ss *signupServer) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	ss.logError(r.Context(), r, err)
	msg := "internal server error"
	ss.errorResponse(w, r, http.StatusInternalServerError, msg)
}

func (ss *signupServer) badRequestResponse(w http.ResponseWriter, r *http.Request, msg string) {
	ss.logError(r.Context(), r, fmt.Errorf("bad request: %s", msg))
	ss.errorResponse(w, r, http.StatusBadRequest, msg)
}
