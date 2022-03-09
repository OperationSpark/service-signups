package signups

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHandleJson(t *testing.T) {
	tests := []struct {
		json []byte
		want Signup
	}{
		{[]byte(`{"startDateTime": null}`), Signup{}},
		{
			[]byte(`{
			"nameFirst": "Henri",
			"nameLast": "Testaroni",
			"email": "henri@email.com",
			"cell": "555-123-4567",
			"referrer": "instagram",
			"referrerResponse": ""
		}
		`),
			Signup{
				NameFirst:        "Henri",
				NameLast:         "Testaroni",
				Email:            "henri@email.com",
				Cell:             "555-123-4567",
				Referrer:         "instagram",
				ReferrerResponse: "",
			}},
	}

	for _, test := range tests {
		got := Signup{}
		err := handleJson(&got, bytes.NewReader(test.json))
		if err != nil {
			t.Errorf("Error unmarshalling JSON: %s\n%v", string(test.json), err)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("handleJSON() mismatch (-want +got):\n%s", diff)
		}
	}
}
