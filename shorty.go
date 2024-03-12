package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type (
	Shortener struct {
		baseApiEndpoint string
		client          http.Client
		apiKey          string
	}

	ShortLink struct {
		// Shortened URL result. Ex: https://ospk.org/bas12d21dc.
		ShortURL string `json:"shortUrl"`
		// Short Code used as the path of the short URL. Ex: bas12d21dc.
		Code string `json:"code"`
		// Optional custom short code passed when creating or updating the short URL.
		CustomCode string `json:"customCode"`
		// The URL where the short URL redirects.
		OriginalUrl string `json:"originalUrl"`
		// Count of times the short URL has been used.
		TotalClicks int `json:"totalClicks"`
		// Identifier of the entity that created the short URL.
		CreatedBy string `json:"createdBy"`
		// DateTime the URL was created.
		CreatedAt time.Time `json:"createdAt"`
		// DateTime the URL was last updated.
		UpdatedAt time.Time `json:"updatedAt"`
	}

	ShortenerOpts struct {
		// Overrides the API endpoint. Defaults to https://ospk.org/api/urls.
		apiOverride string
		// API key needed to use the Shortener service.
		apiKey string
	}
)

// NewURLShortener creates a new Shortener service.
func NewURLShortener(o ShortenerOpts) *Shortener {
	baseApiEndpoint := "https://ospk.org/api/urls"
	if len(o.apiOverride) > 0 {
		baseApiEndpoint = o.apiOverride
	}

	return &Shortener{
		baseApiEndpoint: baseApiEndpoint,
		client:          *http.DefaultClient,
		apiKey:          o.apiKey,
	}
}

// ShortenURL POSTs a URL to Operation Spark's URL shortener service and returns the shortened URL result.
// Ex:
//
//		"https://www.google.com/search?q=url+shortener&source=hp&ei=lwFXY7CTOOmVwbkPx6GZEA&oq=url+shortener"
//	 -> "https://ospk.org/bas12d21dc"
//
// If there is an error, the original URL is returned along with the error.
func (s Shortener) ShortenURL(ctx context.Context, url string) (string, error) {
	body, err := json.Marshal(ShortLink{OriginalUrl: url})
	if err != nil {
		return url, fmt.Errorf("marshall: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseApiEndpoint, bytes.NewReader(body))
	if err != nil {
		return url, fmt.Errorf("newRequestWithContext: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("key", s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return url, fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return url, fmt.Errorf("post: %w", handleHTTPError(resp))
	}

	d := json.NewDecoder(resp.Body)
	var link ShortLink
	err = d.Decode(&link)
	if err != nil {
		return url, fmt.Errorf("decode: %w", err)
	}

	return link.ShortURL, nil
}
