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

// OpenAI's review cited these on a list_tasks row.
func TestRedactKeysDropsRecordAttribution(t *testing.T) {
	var value any
	body := `{"tasks":[{"id":12345,"name":"Task","createdBy":777,"createdByUserId":777,` +
		`"updatedBy":777,"completedBy":777,"commentFollowers":[{"id":777,"type":"users"}]}]}`
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	RedactKeys(value, RestrictedDataKeys)
	got, _ := json.Marshal(value)
	if want := `{"tasks":[{"id":12345,"name":"Task"}]}`; string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
