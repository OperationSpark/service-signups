package signup

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("Want:\n\n%v\n\nbut got:\n\n%v\n\n", want, got)
	}
}

func assertDeepEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	areEqual := reflect.DeepEqual(got, want)
	if !areEqual {
		t.Fatalf("Want:\n %s, but got:\n %s", prettyPrint(want), prettyPrint(got))
	}
}

func prettyPrint(i interface{}) string {
	s, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		return fmt.Sprintf("%+v", i)
	}
	return string(s)
}

func mustMakeTime(t *testing.T, layout, value string) time.Time {
	tiempo, err := time.Parse(layout, value)
	if err != nil {
		t.Fatal("could not make time.Time", err)
	}
	return tiempo
}

func assertNilError(t *testing.T, got error) {
	t.Helper()
	if got != nil {
		t.Fatalf("Expected nil error, but got:\n%v", got)
	}
}
