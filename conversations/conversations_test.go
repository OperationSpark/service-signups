package conversations_test

import (
	"context"
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
			svc := convos.NewService(
				convos.WithMessengerAPIBase("http://localhost:3200/api/v0"),
				convos.WithSigningSecret("a-super-secret-key"),
			)

			err := svc.Run(context.Background(), tt.conversationID, tt.signupID)
			require.NoError(t, err)

			// TODO: Mock the HTTP server and assert the request body is signed
		})
	}
}
