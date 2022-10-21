package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Shortener struct {
	baseApiEndpoint string
	client          http.Client
	apiKey          string
}

func NewURLShortener(baseApiEndpoint, apiKey string) *Shortener {
	return &Shortener{
		baseApiEndpoint: baseApiEndpoint,
		client:          *http.DefaultClient,
		apiKey:          apiKey,
	}
}
func (s Shortener) ShortenURL(ctx context.Context, url string) (string, error) {
	type shortLink struct {
		URL string `json:"url"`
	}

	body, err := json.Marshal(shortLink{URL: url})
	if err != nil {
		return "", fmt.Errorf("marshall: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseApiEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("newRequestWithContext: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("key", s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("post: %v", handleHTTPError(resp))
	}

	d := json.NewDecoder(resp.Body)
	var shortURL shortLink
	err = d.Decode(&shortURL)
	if err != nil {
		return "", fmt.Errorf("decode: %v", err)
	}

	return shortURL.URL, nil
}
