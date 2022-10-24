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
		ShortURL    string    `json:"shortUrl"`
		Code        string    `json:"code"`
		CustomCode  string    `json:"customCode"`
		OriginalUrl string    `json:"originalUrl"`
		TotalClicks int       `json:"totalClicks"`
		CreatedBy   string    `json:"createdBy"`
		CreatedAt   time.Time `json:"createdAt"`
		UpdatedAt   time.Time `json:"updatedAt"`
	}
)

func NewURLShortener(baseApiEndpoint, apiKey string) *Shortener {
	return &Shortener{
		baseApiEndpoint: baseApiEndpoint,
		client:          *http.DefaultClient,
		apiKey:          apiKey,
	}
}
func (s Shortener) ShortenURL(ctx context.Context, url string) (string, error) {
	body, err := json.Marshal(ShortLink{OriginalUrl: url})
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
	var link ShortLink
	err = d.Decode(&link)
	if err != nil {
		return "", fmt.Errorf("decode: %v", err)
	}

	return link.ShortURL, nil
}
