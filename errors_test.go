package signup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type JSONErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestHandleHTTPError(t *testing.T) {
	t.Run("prints the request context and HTTP Error response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tooManyErr := JSONErr{
				Code:    http.StatusTooManyRequests,
				Message: "You have exceeded the daily rate limit of (3) for Add meeting registrant API requests for the registrant (ahlerjia+test@example.com). You can resume these API requests at GMT 00:00:00.",
			}
			JSONError(w, tooManyErr, http.StatusTooManyRequests)
		}))

		resp, err := http.DefaultClient.Get(srv.URL + "/v2/meetings/89012345678/registrants?occurrence_id=1665594000000")
		if err != nil {
			t.Fatal(err)
		}
		gotErr := handleHTTPError(resp)
		if gotErr == nil {
			t.Fatal("unexpected nil err")
		}

		want := fmt.Sprintf(`HTTP Error:
GET: %s
/v2/meetings/89012345678/registrants?occurrence_id=1665594000000

Response:
429 Too Many Requests
{"code":429,"message":"You have exceeded the daily rate limit of (3) for Add meeting registrant API requests for the registrant (ahlerjia+test@example.com). You can resume these API requests at GMT 00:00:00."}
`, srv.URL)

		assertEqual(t, gotErr.Error(), want)

	})
}

// JSONError writes an HTTP error code and a JSON structured error body to an HTTP response.
func JSONError(w http.ResponseWriter, err interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err)
}
