package signup

import (
	"crypto"
	"crypto/hmac"
	"fmt"
)

// CreateSignature take a request body and a secret and returns a signature. The signature is hex encoded and prefixed with "sha256=".
func createSignature(body []byte, secret []byte) ([]byte, error) {
	mac := hmac.New(crypto.SHA256.New, secret)
	_, err := mac.Write(body)
	if err != nil {
		return nil, fmt.Errorf("mac.Write: %w", err)

	}
	calculatedMAC := mac.Sum(nil)
	signature := []byte(fmt.Sprintf("sha256=%x", calculatedMAC))
	return signature, nil
}
