package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// CountOnlySchema returns the schema for the count_only flag of a list tool.
// entity is the plural noun the tool lists (e.g. "tasks").
func CountOnlySchema(entity string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf("If true, return only {\"count\": N}: the exact number of matching %s, no rows "+
			"— use for \"how many\". Ignores page, page_size, verbose, fields.", entity),
		AnyOf: []*jsonschema.Schema{
			{Type: "boolean"},
			{Type: "null"},
		},
		Default: []byte(`false`),
	}
}

// WithCountOnlySchema declares the count_only body on a list tool's output
// schema, which otherwise describes rows only. Mutates in place and returns for
// chaining:
//
//	OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(taskListOutputSchema)),
func WithCountOnlySchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return schema
	}
	if schema.Properties == nil {
		schema.Properties = make(map[string]*jsonschema.Schema)
	}
	schema.Properties["count"] = &jsonschema.Schema{
		Type:        "integer",
		Description: "Exact number of matches across every page. Returned instead of the rows when count_only.",
	}
	return schema
}

// countResult is the count_only response body.
type countResult struct {
	Count int64 `json:"count"`
}

// CountFromToolResult reads the count out of a NewCountToolResult result, for a
// tool wrapping a count_only call. Any other body reports false, so a wiring
// mistake surfaces instead of answering zero.
func CountFromToolResult(result *mcp.CallToolResult) (int64, bool) {
	if result == nil {
		return 0, false
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return 0, false
	}
	var decoded struct {
		Count *int64 `json:"count"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Count == nil {
		return 0, false
	}
	return *decoded.Count, true
}

// countResponse is the slice of a v3 list response the count path reads.
type countResponse struct {
	Meta struct {
		Page struct {
			Count *int64 `json:"count"`
		} `json:"page"`
	} `json:"meta"`
}

// NewCountToolResult answers a count_only request: it rewires requester to page
// 1, one row and an exact count, then returns {"count": N} from meta.page.count.
//
// Pinning the exact count is correctness, not economy: skipCounts=true keeps
// `count` in the response but replaces the total with the lower bound
// (page * pageSize) + 1 — 2 for any non-empty result at pageSize=1. Four
// endpoints default to skipping, and the SDK's ResolveCount only clears the
// bound for callers decoding the typed response, which list tools do not.
func NewCountToolResult[R twapi.HTTPRequester](
	ctx context.Context,
	engine *twapi.Engine,
	requester R,
	label string,
) (*mcp.CallToolResult, error) {
	counting, err := countOnlyRequest(requester)
	if err != nil {
		return NewToolResultTextError("%s: %s", label, err.Error()), nil
	}

	resp, err := twapi.ExecuteRaw(ctx, engine, counting)
	if err != nil {
		return HandleAPIError(err, label)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return HandleAPIError(twapi.NewHTTPError(resp, label), label)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var decoded countResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if decoded.Meta.Page.Count == nil {
		return NewToolResultTextError("%s: the API did not report a count for this listing", label), nil
	}
	return NewToolResultJSON(countResult{Count: *decoded.Meta.Page.Count})
}

// countOnlyRequest copies requester restricted to one row of page one with an
// exact count. Every v3 list request names these slots the same way, so they are
// set by name rather than threaded through 27 call sites. A missing slot is
// reported: left at its default it would answer with a lower bound.
func countOnlyRequest[R twapi.HTTPRequester](requester R) (R, error) {
	value := reflect.ValueOf(&requester).Elem()
	if value.Kind() != reflect.Struct {
		// Every SDK list request is a struct, which keeps the rewiring on the copy.
		return requester, fmt.Errorf("cannot count on a %s request", value.Kind())
	}
	filters := value.FieldByName("Filters")
	if !filters.IsValid() || filters.Kind() != reflect.Struct {
		return requester, fmt.Errorf("%s has no filters to count with", value.Type())
	}
	if err := setCountField(filters, "Page", int64(1)); err != nil {
		return requester, err
	}
	if err := setCountField(filters, "PageSize", int64(1)); err != nil {
		return requester, err
	}
	if err := setCountField(filters, "CountMode", twapi.ListCountModeExact); err != nil {
		return requester, err
	}
	// Per-row work the count throws away, plus the cursor paging that would
	// override the single row above. All optional, and none of them selects which
	// rows match, so clearing them cannot move the count. Include* filters are
	// left alone: IncludeCompleted reads like a sideload but is a filter.
	for _, name := range []string{"Include", "Fields", "Cursor", "Limit"} {
		if field := filters.FieldByName(name); field.IsValid() && field.CanSet() {
			field.Set(reflect.Zero(field.Type()))
		}
	}
	return requester, nil
}

// setCountField assigns value to the named filter field, reporting one that is
// absent, unexported or of the wrong type.
func setCountField(filters reflect.Value, name string, value any) error {
	field := filters.FieldByName(name)
	if !field.IsValid() {
		return fmt.Errorf("%s filters have no %s to count with", filters.Type(), name)
	}
	if !field.CanSet() {
		return fmt.Errorf("%s.%s cannot be set", filters.Type(), name)
	}
	assign := reflect.ValueOf(value)
	if !assign.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("%s.%s is a %s, expected %s", filters.Type(), name, field.Type(), assign.Type())
	}
	field.Set(assign)
	return nil
}
