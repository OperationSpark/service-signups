package signup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShortenURL(t *testing.T) {
	t.Run("calls shortening service", func(t *testing.T) {
		apiKey := "TEST_API_KEY"
		wantURL := "https://ospk.org/ahd2dh1xg2j"

		mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("key") != apiKey {
				fmt.Fprint(w, http.StatusUnauthorized)
				return
			}
			fmt.Fprintf(w, `{"url": %q }`, wantURL)
		}))

		fmt.Println(mockSrv.URL)
		shorty := NewURLShortener(mockSrv.URL, apiKey)

		got, err := shorty.ShortenURL(context.Background(), "http://thisisalongurl.gov/q?x=1&morestuff=everything")

		if err != nil {
			t.Fatal(err)
		}

		if got != wantURL {
			t.Fatalf("want %q, but got %q", wantURL, got)
		}
	})
}
