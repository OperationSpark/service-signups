package conversations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	convos "github.com/operationspark/service-signup/conversations"
	"github.com/stretchr/testify/require"
)

func TestLinkConversation(t *testing.T) {
	// test cases
	tests := []struct {
		desc           string
		signupID       string
		conversationID string
		wantErr        bool
	}{
		{
			desc:           "success",
			signupID:       "123",
			conversationID: "123",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Mock the Messenger API
			mockMux := http.NewServeMux()
			mockMux.HandleFunc("/conversations/{convoID}/link", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					require.Equal(t, http.MethodPost, r.Method, "HTTP method")
					return
				}

				// test conversation ID path
				gotConvoID := r.PathValue("convoID")
				if gotConvoID != tt.conversationID {
					w.WriteHeader(http.StatusBadRequest)
					require.Equal(t, tt.conversationID, gotConvoID, "conversation ID in path")
					return
				}

				if r.Header.Get("Content-Type") != "application/json" {
					w.WriteHeader(http.StatusBadRequest)
					require.Equal(t, "application/json", r.Header.Get("Content-Type"), "Content-Type header")
					return
				}

				if r.Header.Get("x-auth-signature-512") == "" {
					w.WriteHeader(http.StatusUnauthorized)
					require.NotEmpty(t, r.Header.Get("x-auth-signature-512"), "x-auth-signature-512 header")
					return
				}

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`
{ "id": "a-user-id", "image": "DEFAULT_PROFILE_IMAGE", "signupId": "123", "name": "Jean Deaux" }
`[1:]))
			})

			mockMessengerSrv := httptest.NewServer(mockMux)
			defer mockMessengerSrv.Close()

			svc := convos.NewService(
				convos.WithMessengerAPIBase(mockMessengerSrv.URL),
				convos.WithSigningSecret("a-super-secret-key"),
			)

			err := svc.Run(context.Background(), tt.conversationID, tt.signupID)
			require.NoError(t, err)
		})
	}
}
