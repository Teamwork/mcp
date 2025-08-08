package helpers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/helpers"
)

//nolint:lll
func TestWebLinker(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		url     string
		want    []byte
		builder func(map[string]any) string
		options []helpers.WebLinkerOption
	}{{
		name:    "single entity",
		data:    []byte(`{"entity":{"id":123,"name":"Test"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"id":123,"name":"Test","meta":{"webLink":"https://example.com/entities/123"}}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "multiple entities",
		data:    []byte(`{"entities":[{"id":123,"name":"Test1"},{"id":456,"name":"Test2"}]}`),
		url:     "https://example.com/",
		want:    []byte(`{"entities":[{"id":123,"name":"Test1","meta":{"webLink":"https://example.com/entities/123"}},{"id":456,"name":"Test2","meta":{"webLink":"https://example.com/entities/456"}}]}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "with known root fields",
		data:    []byte(`{"meta":{"page":1},"included":[{"id":789,"name":"Included"}],"entity":{"id":123,"name":"Test"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"meta":{"page":1},"included":[{"id":789,"name":"Included"}],"entity":{"id":123,"name":"Test","meta":{"webLink":"https://example.com/entities/123"}}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "non-JSON data",
		data:    []byte(`Not a JSON`),
		url:     "https://example.com/",
		want:    []byte(`Not a JSON`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "JSON without id",
		data:    []byte(`{"entity":{"name":"Test"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"name":"Test"}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "JSON with empty id string",
		data:    []byte(`{"entity":{"id":"","name":"Test"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"id":"","name":"Test"}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "JSON with zero numeric id",
		data:    []byte(`{"entity":{"id":0,"name":"Zero"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"id":0,"name":"Zero"}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "array with mixed items",
		data:    []byte(`{"entities":[{"id":123,"name":"Test1"},{"name":"Test2"},{"id":456,"name":"Test3"}]}`),
		url:     "https://example.com/",
		want:    []byte(`{"entities":[{"id":123,"name":"Test1","meta":{"webLink":"https://example.com/entities/123"}},{"name":"Test2"},{"id":456,"name":"Test3","meta":{"webLink":"https://example.com/entities/456"}}]}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "array with non-object elements",
		data:    []byte(`{"entities":[{"id":1,"name":"One"},"string",2,{"id":2,"name":"Two"}]}`),
		url:     "https://example.com/",
		want:    []byte(`{"entities":[{"id":1,"name":"One","meta":{"webLink":"https://example.com/entities/1"}},"string",2,{"id":2,"name":"Two","meta":{"webLink":"https://example.com/entities/2"}}]}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "entity with existing meta without webLink adds new key",
		data:    []byte(`{"entity":{"id":123,"name":"Test","meta":{"page":1}}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"id":123,"name":"Test","meta":{"page":1,"webLink":"https://example.com/entities/123"}}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "entity with existing meta and webLink preserved",
		data:    []byte(`{"entity":{"id":123,"name":"Test","meta":{"webLink":"https://other.com/custom/123","note":"keep"}}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"id":123,"name":"Test","meta":{"webLink":"https://other.com/custom/123","note":"keep"}}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "additional ignore field skips injection",
		data:    []byte(`{"skipMe":{"id":9,"name":"Ignored"},"entity":{"id":1,"name":"One"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"skipMe":{"id":9,"name":"Ignored"},"entity":{"id":1,"name":"One","meta":{"webLink":"https://example.com/entities/1"}}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
		options: []helpers.WebLinkerOption{helpers.WebLinkerWithIgnoreFields("skipMe")},
	}, {
		name:    "missing customer URL in context returns original",
		data:    []byte(`{"entity":{"id":1,"name":"One"}}`),
		url:     "",
		want:    []byte(`{"entity":{"id":1,"name":"One"}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}, {
		name:    "nil path builder returns original",
		data:    []byte(`{"entity":{"id":1,"name":"One"}}`),
		url:     "https://example.com/",
		want:    []byte(`{"entity":{"id":1,"name":"One"}}`),
		builder: nil,
	}, {
		name:    "customer URL without trailing slash",
		data:    []byte(`{"entity":{"id":2,"name":"Two"}}`),
		url:     "https://example.com",
		want:    []byte(`{"entity":{"id":2,"name":"Two","meta":{"webLink":"https://example.com/entities/2"}}}`),
		builder: helpers.WebLinkerWithIDPathBuilder("entities"),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.url != "" { // only set when provided
				ctx = config.WithCustomerURL(ctx, tt.url)
			}
			got := helpers.WebLinker(ctx, tt.data, tt.builder, tt.options...)
			// we cannot compare the bytes because the order of fields in JSON may vary
			// so we compare the decoded maps instead
			var gotMap, wantMap map[string]any
			gotErr, wantErr := json.Unmarshal(got, &gotMap), json.Unmarshal(tt.want, &wantMap)
			if gotErr != nil || wantErr != nil {
				if !bytes.Equal(got, tt.want) {
					t.Errorf("unexpected result (non-json) %s, want %s", got, tt.want)
				}
				return
			}
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("unexpected result %s, want %s", got, tt.want)
			}
		})
	}
}

func TestWebLinkerWithIDPathBuilder(t *testing.T) {
	builder := helpers.WebLinkerWithIDPathBuilder("entities")

	tests := []struct {
		name   string
		object map[string]any
		want   string
	}{
		{
			name:   "missing id field returns empty",
			object: map[string]any{"name": "no id"},
			want:   "",
		},
		{
			name:   "int id builds path",
			object: map[string]any{"id": 123},
			want:   "entities/123",
		},
		{
			name:   "zero int id returns empty",
			object: map[string]any{"id": 0},
			want:   "",
		},
		{
			name:   "float id is truncated to int64 in path (no decimals)",
			object: map[string]any{"id": 123.0},
			want:   "entities/123",
		},
		{
			name:   "big float id is truncated to int64 in path (no decimals)",
			object: map[string]any{"id": 12345678901234.0},
			want:   "entities/12345678901234",
		},
		{
			name:   "float id with decimals is not truncated",
			object: map[string]any{"id": 123.9},
			want:   "entities/123.9",
		},
		{
			name:   "string id builds path",
			object: map[string]any{"id": "abc"},
			want:   "entities/abc",
		},
		{
			name:   "empty string id returns empty",
			object: map[string]any{"id": ""},
			want:   "",
		},
		{
			name:   "true bool id builds path",
			object: map[string]any{"id": true},
			want:   "entities/true",
		},
		{
			name:   "false bool id returns empty (zero value)",
			object: map[string]any{"id": false},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := builder(tt.object); got != tt.want {
				t.Errorf("builder returned %q, want %q (id=%#v)", got, tt.want, tt.object["id"])
			}
		})
	}
}
