package signing

import (
	"crypto"
	"crypto/hmac"
	"fmt"
	"strings"
)

// Sign takes a payload and a secret and returns a signature. The signature is hex encoded and prefixed with the hash function used.
func Sign(payload []byte, secret []byte, algo crypto.Hash) ([]byte, error) {
	mac := hmac.New(algo.New, secret)
	_, err := mac.Write(payload)
	if err != nil {
		return nil, fmt.Errorf("mac.Write: %w", err)

	}
	calculatedMAC := mac.Sum(nil)
	algoLabel := strings.Replace(strings.ToLower(algo.String()), "-", "", 1)
	signature := []byte(fmt.Sprintf("%s=%x", algoLabel, calculatedMAC))
	return signature, nil
}
