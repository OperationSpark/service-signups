// Package signing provides a service for signing and verifying messages.
package signing

import (
	"crypto"
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"strings"
)

type Encoding int

const (
	EncodingUnknown Encoding = iota // Unknown encoding
	EncodingHex                     // Hex encoding
	EncodingBase64                  // Base64 encoding
)

// Sign takes a payload and a secret and returns a signature. The signature is hex encoded and prefixed with the hash function used.
func Sign(payload []byte, secret []byte, algo crypto.Hash, enc Encoding) ([]byte, error) {
	mac := hmac.New(algo.New, secret)
	_, err := mac.Write(payload)
	if err != nil {
		return nil, fmt.Errorf("mac.Write: %w", err)
	}
	calculatedMAC := mac.Sum(nil)
	algoLabel := strings.Replace(strings.ToLower(algo.String()), "-", "", 1)

	switch enc {
	case EncodingHex:
		signature := []byte(fmt.Sprintf("%s=%x", algoLabel, calculatedMAC))
		return signature, nil
	case EncodingBase64:
		signature := []byte(fmt.Sprintf("%s=%s", algoLabel, base64.StdEncoding.EncodeToString(calculatedMAC)))
		return signature, nil
	}
	return nil, fmt.Errorf("unsupported encoding")
}
