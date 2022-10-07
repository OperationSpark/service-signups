package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockSignupService struct {
	RegisterFunc func(context.Context, Signup) error
}

func (m *MockSignupService) register(ctx context.Context, signup Signup) error {
	return m.RegisterFunc(ctx, signup)
}

func TestHandleSignup(t *testing.T) {
	t.Run("can register valid users", func(t *testing.T) {
		signup := Signup{
			NameFirst:        "Henri",
			NameLast:         "Testaroni",
			Email:            "henri@email.com",
			Cell:             "555-123-4567",
			Referrer:         "instagram",
			ReferrerResponse: "",
		}

		service := &MockSignupService{
			RegisterFunc: func(context.Context, Signup) error {
				return nil
			},
		}

		server := newSignupServer(service)

		req := httptest.NewRequest(http.MethodPost, "/", signupToJson(t, signup))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		server.HandleSignUp(res, req)

		assertStatus(t, res.Code, http.StatusCreated)
	})
}

func signupToJson(t *testing.T, signup Signup) io.Reader {
	b, err := json.Marshal(signup)
	if err != nil {
		t.Fatalf("marshall json: %v", err)
	}
	return bytes.NewReader(b)
}

func assertStatus(t *testing.T, want, got int) {
	if want != got {
		t.Fatalf("expected status %d, but got %d", want, got)
	}
}
