package helpers

import (
	"encoding/json"
	"testing"
)

func TestRedactKeysIsCaseInsensitiveAndDeep(t *testing.T) {
	var value any
	body := `{"Email":"john@example.com","user":{"EMAILONE":"x","name":"John"},` +
		`"rows":[{"phone":"1","allocation":{"id":7}}]}`
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	RedactKeys(value, RestrictedDataKeys)
	got, _ := json.Marshal(value)
	if want := `{"rows":[{"allocation":{"id":7}}],"user":{"name":"John"}}`; string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
