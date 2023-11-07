package signup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateSignature(t *testing.T) {
	t.Run("returns a signature", func(t *testing.T) {
		secret := []byte("It's a Secret to Everybody")
		payload := []byte("Hello, World!")
		want := []byte("sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")

		got, err := createSignature(payload, secret)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}
