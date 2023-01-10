package greenlight

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGooglePlaceUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		inputJSON string
		want      GooglePlace
	}{
		{
			inputJSON: `{"googlePlace":""}`,
			want:      GooglePlace{},
		},
		{
			inputJSON: `{"googlePlace":{"placeId":"123456","name":"","address":"123 Some St","phone":"5551234567","website":"example.com","geometry":{"lat":0,  "lng":0}}}`,
			want:      GooglePlace{PlaceID: "123456", Name: "", Address: "123 Some St", Phone: "5551234567", Website: "example.com", Geometry: Geometry{Lat: 0, Lng: 0}},
		},
	}

	type placeholder struct {
		GooglePlace GooglePlace `json:"googlePlace"`
	}

	for _, tc := range testCases {
		var got placeholder
		err := json.Unmarshal([]byte(tc.inputJSON), &got)
		require.NoError(t, err)

		require.Equal(t, tc.want, got.GooglePlace)
	}

}
