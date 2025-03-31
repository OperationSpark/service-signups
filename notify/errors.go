package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
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

// logError logs the error to the server's logger and Sentry if it's enabled.
func (s *Server) logError(ctx context.Context, err error) {
	s.logger.ErrorContext(ctx, err.Error())

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
	}
}

// logRequestError logs the error and sends a generic 500 Internal Server Error response to the client.
// It also logs the error to Sentry if it's enabled.
func (s *Server) logRequestError(ctx context.Context, r *http.Request, err error) {
	method := r.Method
	url := r.URL.String()

	s.logger.ErrorContext(
		ctx,
		err.Error(),
		slog.String("method", method),
		slog.String("url", url))

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
	}
}

// errorResponse writes an error response to the client. The msg is only logged to the server's logger and not sent back to the client.
func (s *Server) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	respData := errorResponse{Error: msg}
	err := s.writeJSON(w, status, respData)
	if err != nil {
		s.logRequestError(r.Context(), r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse logs the error and sends a generic 500 Internal Server Error response to the client.
func (s *Server) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	s.logRequestError(r.Context(), r, err)
	msg := "internal server error"
	s.errorResponse(w, r, http.StatusInternalServerError, msg)
}

// badRequestResponse logs the error and sends a 400 Bad Request response to the client.
// The msg is sent as the error message in the response body, so it should be a human-readable message and not leak any sensitive information.
func (s *Server) badRequestResponse(w http.ResponseWriter, r *http.Request, msg string) {
	s.logRequestError(r.Context(), r, fmt.Errorf("bad request: %s", msg))
	s.errorResponse(w, r, http.StatusBadRequest, msg)
}

// notFoundResponse logs the error and sends a 404 Not Found response to the client.
// The msg is sent as the error message in the response body, so it should be a human-readable message and not leak any sensitive information.
func (s *Server) notFoundResponse(w http.ResponseWriter, r *http.Request, msg string) {
	s.logRequestError(r.Context(), r, fmt.Errorf("not found: %s", msg))
	s.errorResponse(w, r, http.StatusNotFound, msg)
}
