package signup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/schema"
)

type registerer interface {
	register(ctx context.Context, signup Signup, logger *slog.Logger) (Signup, error)
}

type signupServer struct {
	service registerer
	logger  *slog.Logger
}

// badReqBodyResp is the response body for a bad request. This is used for an invalid SignUp request.
type badReqBodyResp struct {
	Message string `json:"message"` // Message is the error message.
	Field   string `json:"field"`   // Field is the field that caused the error.
}

type response struct {
	URL string `json:"url"`
}

func (ss *signupServer) HandleSignUp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var su Signup

	// Parse JSON or URL Encoded Signup Form
	switch r.Header.Get("Content-Type") {
	case "application/json":
		err := handleJson(&su, r.Body)
		if err != nil {
			ss.badRequestResponse(w, r, fmt.Errorf("invalid JSON body: %w", err).Error())
			return
		}

	case "application/x-www-form-urlencoded":
		err := handleForm(&su, r)
		if err != nil {
			ss.badRequestResponse(w, r, fmt.Errorf("invalid form body: %w", err).Error())
			return
		}

	default:
		ss.errorResponse(w, r, http.StatusUnsupportedMediaType, "Unacceptable Content-Type")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var signupID string
	var conversationID string
	if su.id != nil {
		signupID = *su.id
	}
	if su.conversationID != nil {
		conversationID = *su.conversationID
	}

	signupLogger := ss.logger.With(
		slog.Group("signup",
			slog.String("nameFirst", su.NameFirst),
			slog.String("nameLast", su.NameLast),
			slog.String("cell", su.Cell),
			slog.String("email", su.Email),
			slog.String("startDateTime", su.StartDateTime.Format(time.RFC3339)),
			slog.String("cohort", su.Cohort),
			slog.String("id", signupID),
			slog.String("conversationID", conversationID),
			slog.String("sessionID", su.SessionID),
			slog.Bool("smsOptIn", su.SMSOptIn),
		),
	)

	postRegistration, err := ss.service.register(r.Context(), su, signupLogger)
	// depending on what we get back, respond accordingly
	if err != nil {
		// handle invalid phone number error
		if strings.Contains(err.Error(), "invalid number") {
			// marshall error response
			errResp := badReqBodyResp{
				Message: "Invalid Phone Number",
				Field:   "phone",
			}

			if err := ss.writeJSON(w, http.StatusBadRequest, errResp); err != nil {
				ss.serverErrorResponse(w, r, fmt.Errorf("write 'bad request' response: %w", err))
			}
			return
		}

		signupLogger.ErrorContext(r.Context(), "signup failed")
		ss.serverErrorResponse(w, r, fmt.Errorf("user registration: %w", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response{URL: postRegistration.ShortLink}); err != nil {
		ss.serverErrorResponse(w, r, fmt.Errorf("write 'created' response: %w", err))
		return
	}

}

// handleJson unmarshalls a JSON payload from a signUp request into a Signup.
func handleJson(s *Signup, body io.Reader) error {
	var timeParseError *time.ParseError

	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, s)
	if errors.As(err, &timeParseError) {
		return &InvalidFieldError{Field: "startDateTime"}
	}
	if err != nil {
		return err
	}

	return nil
}

// handleForm unmarshalls a FormData payload from a signUp request into a Signup
func handleForm(s *Signup, r *http.Request) error {
	decoder := schema.NewDecoder()

	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = decoder.Decode(s, r.PostForm)
	if err != nil {
		return err
	}

	return nil
}

func (ss *signupServer) writeJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
