package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/schema"
)

type registerer interface {
	register(ctx context.Context, signup Signup) error
}

type signupServer struct {
	service registerer
}

func (ss *signupServer) HandleSignUp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var su Signup

	// Parse JSON or URL Encoded Signup Form
	switch r.Header.Get("Content-Type") {
	case "application/json":
		err := handleJson(&su, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			panic(err)
		}

	case "application/x-www-form-urlencoded":
		err := handleForm(&su, r)
		if err != nil {
			http.Error(w, "Error reading Form Body", http.StatusBadRequest)
			panic(err)
		}

	default:
		http.Error(w, "Unacceptable Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	err := ss.service.register(r.Context(), su)
	// depending on what we get back, respond accordingly
	if err != nil {
		// TODO: handle different kinds of errors differently
		// handle invalid phone number error
		if strings.Contains(err.Error(), "invalid number") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, bytes.NewBufferString(`{
				"message": "Phone Number Invalid",
				"field": "phone number"
				}`))
			fmt.Fprint(w, http.StatusBadRequest)
			http.Error(w, "Invalid Phone Number", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(os.Stderr, "\nproblem signing user up: %v\n\n", err)
		fmt.Printf("Signup:\n%s\n", prettyPrint(su))
		http.Error(w, "problem signing user up\n", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
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
