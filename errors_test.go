package signup

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestHandleHTTPError(t *testing.T) {
	t.Run("prints human-readable message for HTTP error status responses", func(t *testing.T) {
		respBody := "Invalid API key"
		resp := &http.Response{
			Status:     "401 Unauthorized",
			StatusCode: http.StatusUnauthorized,
			Body:       ioutil.NopCloser(bytes.NewBufferString(respBody)),
		}

		err := handleHTTPError(resp)
		assertEqual(t, err.Error(), "HTTP: 401 Unauthorized\nresponse body: Invalid API key")
	})

	t.Run("does not print HTML bodies", func(t *testing.T) {
		respBody := "<html><body>{A giant chunk of HTML we don't want to see in the logs}</body></html>"
		headers := make(http.Header, 1)
		headers.Add("Content-Type", "text/html; charset=UTF-8")
		resp := &http.Response{
			Status:     "404 Not Found",
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewBufferString(respBody)),
			Header:     headers,
		}

		err := handleHTTPError(resp)
		assertEqual(t, err.Error(), "HTTP: 404 Not Found\n[HTML response removed]")
	})

}
