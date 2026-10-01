package helpers

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRetainKeys(t *testing.T) {
	var value any
	body := `{"id":1,"email":"john@example.com","lastLogin":"2026-01-01T00:00:00Z"}`
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	RetainKeys(value, NewKeySet("id", "email"))
	want := map[string]any{"id": float64(1), "email": "john@example.com"}
	if !reflect.DeepEqual(value, want) {
		t.Errorf("got %v, want %v", value, want)
	}
}

func TestRetainKeysEachHandlesArraysAndIDMaps(t *testing.T) {
	var value map[string]any
	body := `{"list":[{"id":1,"x":1}],"byID":{"7":{"id":7,"x":1}},"scalar":3}`
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	keep := NewKeySet("id")
	RetainKeysEach(value["list"], keep)
	RetainKeysEach(value["byID"], keep)
	RetainKeysEach(value["scalar"], keep)

	got, _ := json.Marshal(value)
	if want := `{"byID":{"7":{"id":7}},"list":[{"id":1}],"scalar":3}`; string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
