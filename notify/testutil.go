package notify

import (
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func MustFakeZoomURL(t *testing.T) string {
	rand.NewSource(time.Now().Unix())
	id := rand.Intn(int(math.Pow10(11)))
	return fmt.Sprintf("https://us06web.zoom.us/w/%d?tk=%s.%s", id, mustRandHex(t, 43), mustRandHex(t, 20))
}

func mustRandHex(t *testing.T, n int) string {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	require.NoError(t, err)
	return hex.EncodeToString(bytes)
}

// RandID generates a random 17-character string to simulate Meteor's Mongo ID generation.
// Meteor did not originally use Mongo's ObjectID() for document IDs.
func randID() string {
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	length := 17
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func mustMakeReq(t *testing.T, body io.Reader) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/notify", body)
	require.NoError(t, err)
	return req
}
