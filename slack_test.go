package signup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendWebhook(t *testing.T) {
	t.Run("send a webhook containing a message to the given URL", func(t *testing.T) {
		type body map[string]string

		msg := message{Text: "Arceele signed up for an Info Session"}

		slackAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload body
			d := json.NewDecoder(r.Body)
			d.Decode(&payload)

			assertEqual(t, payload["text"], msg.Text)
		}))

		err := sendWebhook(slackAPI.URL, msg)
		if err != nil {
			t.Fatalf("sendWebhook: %v", err)
		}
	})
}
