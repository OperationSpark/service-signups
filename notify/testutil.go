package notify

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func MustFakeZoomURL(t *testing.T) string {
	t.Helper()
	id, err := rand.Int(rand.Reader, big.NewInt(int64(math.Pow10(11))))
	require.NoError(t, err)
	return fmt.Sprintf("https://us06web.zoom.us/w/%d?tk=%s.%s", id, mustRandHex(t, 43), mustRandHex(t, 20))
}

func mustRandHex(t *testing.T, n int) string {
	t.Helper()

	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	require.NoError(t, err)
	return hex.EncodeToString(bytes)
}

// MustRandID generates a random 17-character string to simulate Meteor's Mongo ID generation.
// Meteor did not originally use Mongo's ObjectID() for document IDs.
func mustRandID(t *testing.T) string {
	t.Helper()

	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	length := 17
	b := make([]rune, length)
	for i := range b {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		require.NoError(t, err)
		b[i] = letters[randIndex.Int64()]
	}
	return string(b)
}

func mustMakeReq(t *testing.T, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/notify", body)
	require.NoError(t, err)
	return req
}
