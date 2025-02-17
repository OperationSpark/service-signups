package signing_test

import (
	"crypto"
	"testing"

	"github.com/operationspark/service-signup/signing"
	"github.com/stretchr/testify/require"
)

func TestCreateSignature(t *testing.T) {
	testCase := []struct {
		desc    string
		secret  []byte
		payload []byte
		want    string
		algo    crypto.Hash
	}{
		{
			desc:    "returns a SHA-256 signature",
			payload: []byte("Hello, World!"),
			want:    "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17",
			secret:  []byte("It's a Secret to Everybody"),
			algo:    crypto.SHA256,
		},
		{
			desc:    "returns a SHA-512 signature",
			payload: []byte("Hello, World!"),
			want:    "sha512=11ed355a617e98134e842012a7944ccf59c10256cb182357bd7e3a42013ff07c376f8c14cf5cc1923da20b51d64256b2fb8ebbf100aa67a61326f61fea8111bc",
			secret:  []byte("It's a Secret to Everybody"),
			algo:    crypto.SHA512,
		},
	}

	for _, tt := range testCase {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := signing.Sign(tt.payload, tt.secret, tt.algo)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}
