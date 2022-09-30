package signup

import (
	"encoding/json"
	"reflect"
	"testing"
)

func assertEqual(t *testing.T, got, want string) {
	if got != want {
		t.Fatalf("Want: %q, but got: %q", want, got)
	}
}

func assertDeepEqual(t *testing.T, got, want interface{}) {
	areEqual := reflect.DeepEqual(got, want)
	if !areEqual {
		t.Fatalf("Want:\n %s, but got:\n %s", prettyPrint(want), prettyPrint(got))
	}
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
