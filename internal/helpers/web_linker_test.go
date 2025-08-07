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
		name string
		data []byte
		url  string
		want []byte
	}{{
		name: "single entity",
		data: []byte(`{"entity":{"id":123,"name":"Test"}}`),
		url:  "https://example.com/",
		want: []byte(`{"entity":{"id":123,"name":"Test","webLink":"https://example.com/entities/123"}}`),
	}, {
		name: "multiple entities",
		data: []byte(`{"entities":[{"id":123,"name":"Test1"},{"id":456,"name":"Test2"}]}`),
		url:  "https://example.com/",
		want: []byte(`{"entities":[{"id":123,"name":"Test1","webLink":"https://example.com/entities/123"},{"id":456,"name":"Test2","webLink":"https://example.com/entities/456"}]}`),
	}, {
		name: "with known root fields",
		data: []byte(`{"meta":{"page":1},"included":[{"id":789,"name":"Included"}],"entity":{"id":123,"name":"Test"}}`),
		url:  "https://example.com/",
		want: []byte(`{"meta":{"page":1},"included":[{"id":789,"name":"Included"}],"entity":{"id":123,"name":"Test","webLink":"https://example.com/entities/123"}}`),
	}, {
		name: "non-JSON data",
		data: []byte(`Not a JSON`),
		url:  "https://example.com/",
		want: []byte(`Not a JSON`),
	}, {
		name: "JSON without id",
		data: []byte(`{"entity":{"name":"Test"}}`),
		url:  "https://example.com/",
		want: []byte(`{"entity":{"name":"Test"}}`),
	}, {
		name: "JSON with empty id",
		data: []byte(`{"entity":{"id":"","name":"Test"}}`),
		url:  "https://example.com/",
		want: []byte(`{"entity":{"id":"","name":"Test"}}`),
	}, {
		name: "array with mixed items",
		data: []byte(`{"entities":[{"id":123,"name":"Test1"},{"name":"Test2"},{"id":456,"name":"Test3"}]}`),
		url:  "https://example.com/",
		want: []byte(`{"entities":[{"id":123,"name":"Test1","webLink":"https://example.com/entities/123"},{"name":"Test2"},{"id":456,"name":"Test3","webLink":"https://example.com/entities/456"}]}`),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := config.WithCustomerURL(context.Background(), tt.url)
			got := helpers.WebLinker(ctx, tt.data, helpers.WebLinkerWithIDPathBuilder("entities"))
			// we cannot compare the bytes because the order of fields in JSON may vary
			// so we compare the decoded maps instead
			var gotMap, wantMap map[string]any
			gotErr, wantErr := json.Unmarshal(got, &gotMap), json.Unmarshal(tt.want, &wantMap)
			if gotErr != nil || wantErr != nil {
				if !bytes.Equal(got, tt.want) {
					t.Errorf("unexpected result %s, want %s", got, tt.want)
				}
			}
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("unexpected result %s, want %s", got, tt.want)
			}
		})
	}
}
