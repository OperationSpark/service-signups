package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockSignupService struct {
	RegisterFunc func(context.Context, Signup) (Signup, error)
}

func (m *MockSignupService) register(ctx context.Context, signup Signup) (Signup, error) {
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
			RegisterFunc: func(ctx context.Context, su Signup) (Signup, error) {
				return su, nil
			},
		}

		server := &signupServer{
			service: service,
			logger:  nil,
		}

		req := httptest.NewRequest(http.MethodPost, "/", signupToJson(t, signup))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		server.HandleSignUp(res, req)

		require.Equal(t, http.StatusCreated, res.Code)
	})

	t.Run("responds with the  generated info URL", func(t *testing.T) {
		signup := Signup{
			NameFirst:        "Henri",
			NameLast:         "Testaroni",
			Email:            "henri@email.com",
			Cell:             "555-123-4567",
			Referrer:         "tiktok",
			ReferrerResponse: "",
		}

		service := &MockSignupService{
			RegisterFunc: func(ctx context.Context, su Signup) (Signup, error) {
				su.ShortLink = "https://ospk.org/abcd1234"
				return su, nil
			},
		}

		server := &signupServer{service: service, logger: nil}

		req := httptest.NewRequest(http.MethodPost, "/", signupToJson(t, signup))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		server.HandleSignUp(res, req)

		require.Equal(t, http.StatusCreated, res.Code)
		require.JSONEq(t, `{"url":"https://ospk.org/abcd1234"}`, res.Body.String())
	})
}

func signupToJson(t *testing.T, signup Signup) io.Reader {
	b, err := json.Marshal(signup)
	if err != nil {
		t.Fatalf("marshall json: %v", err)
	}
	return bytes.NewReader(b)
}
