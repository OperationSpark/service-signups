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
	t.Run("prints human-readable message for HTTP error status responses", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Respond with an error
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
		}))

		resp, err := http.DefaultClient.Get(mockServer.URL)
		if err != nil {
			t.Fatal(err)
		}
		err = HandleHTTPError(resp)
		assertEqual(t, err.Error(), fmt.Sprintf(`HTTP Error:
GET: %s
/

Response:
401 Unauthorized
Invalid API key`+"\n", mockServer.URL))
	})

	t.Run("does not print HTML bodies", func(t *testing.T) {
		respBody := "<html><body>{A giant chunk of HTML we don't want to see in the logs}</body></html>"
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Respond with HTML not found page
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, respBody)
		}))

		resp, err := http.DefaultClient.Get(mockServer.URL)
		if err != nil {
			t.Fatal(err)
		}

		err = HandleHTTPError(resp)
		assertEqual(t, err.Error(), fmt.Sprintf(`HTTP Error:
GET: %s
/

Response:
404 Not Found
[HTML response removed]`, mockServer.URL))
	})

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
		gotErr := HandleHTTPError(resp)
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
	if err := json.NewEncoder(w).Encode(err); err != nil {
		panic(err)
	}
}
