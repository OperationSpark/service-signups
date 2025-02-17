package signup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckEnvVars(t *testing.T) {
	t.Run("fails if any require env vars are missing or empty", func(t *testing.T) {
		err := checkEnvVars(false)
		require.ErrorContains(t, err, "is required")
	})

	t.Run("returns nil if all vars have non-empty values", func(t *testing.T) {
		requiredVars := []string{"GREENLIGHT_API_KEY",
			"GREENLIGHT_WEBHOOK_URL",
			"MAIL_DOMAIN",
			"MAILGUN_API_KEY",
			"MONGO_URI",
			"OS_MESSAGING_SERVICE_URL",
			"OS_MESSAGING_SIGNING_SECRET",
			"OS_RENDERING_SERVICE_URL",
			"SLACK_WEBHOOK_URL",
			"TWILIO_ACCOUNT_SID",
			"TWILIO_AUTH_TOKEN",
			"TWILIO_CONVERSATIONS_SID",
			"TWILIO_PHONE_NUMBER",
			"URL_SHORTENER_API_KEY",
			"ZOOM_ACCOUNT_ID",
			"ZOOM_CLIENT_ID",
			"ZOOM_CLIENT_SECRET",
			"ZOOM_MEETING_12",
			"ZOOM_MEETING_17"}
		for _, envVar := range requiredVars {
			os.Setenv(envVar, "a_value")
		}

		err := checkEnvVars(false)
		require.NoError(t, err)
		for _, envVar := range requiredVars {
			os.Setenv(envVar, "")
		}
	})
}
