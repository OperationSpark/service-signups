package signup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShortenURL(t *testing.T) {
	t.Run("calls shortening service", func(t *testing.T) {
		apiKey := "TEST_API_KEY"
		shortCode := "ahd2dh1xg2j"
		wantURL := "https://ospk.org/" + shortCode
		originalUrl := "http://thisisalongurl.gov/q?x=1&morestuff=everything"

		mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("key") != apiKey {
				fmt.Fprint(w, http.StatusUnauthorized)
				return
			}

			var reqBody ShortLink
			d := json.NewDecoder(r.Body)
			err := d.Decode(&reqBody)
			if err != nil {
				t.Fatal(err)
			}

			assertEqual(t, reqBody.OriginalUrl, originalUrl)

			resp := ShortLink{ShortURL: wantURL, Code: shortCode, OriginalUrl: reqBody.OriginalUrl}
			e := json.NewEncoder(w)
			e.Encode(resp)
		}))

		shorty := NewURLShortener(ShortenerOpts{mockSrv.URL, apiKey})

		got, err := shorty.ShortenURL(context.Background(), originalUrl)

		if err != nil {
			t.Fatal(err)
		}

		if got != wantURL {
			t.Fatalf("want %q, but got %q", wantURL, got)
		}
	})
}