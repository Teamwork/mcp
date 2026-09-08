package toolsets

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// nullableInteger mirrors the anyOf: [{integer}, {null}] shape used across the
// tool schemas for optional numeric parameters.
func nullableInteger() *jsonschema.Schema {
	return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}}}
}

func nullableBoolean() *jsonschema.Schema {
	return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: "boolean"}, {Type: "null"}}}
}

func nullableString() *jsonschema.Schema {
	return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: "string"}, {Type: "null"}}}
}

func TestCoerceStringValues(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"project_id":  nullableInteger(),
			"page":        nullableInteger(),
			"verbose":     nullableBoolean(),
			"search_term": nullableString(),
			"tag_ids": {
				AnyOf: []*jsonschema.Schema{
					{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
					{Type: "null"},
				},
			},
			"groups": {
				AnyOf: []*jsonschema.Schema{
					{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"user_ids": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
						},
					},
					{Type: "null"},
				},
			},
		},
	}

	args := map[string]any{
		"project_id":  "911218",
		"page":        "2",
		"verbose":     "false",
		"search_term": "911218",        // must stay a string
		"tag_ids":     []any{"1", "2"}, // stringified array elements
		"groups":      map[string]any{"user_ids": []any{"7"}},
	}

	if !coerceStringValues(schema, args) {
		t.Fatalf("expected coerceStringValues to report a change")
	}

	if got := args["project_id"]; got != float64(911218) {
		t.Errorf("project_id = %#v, want float64(911218)", got)
	}
	if got := args["page"]; got != float64(2) {
		t.Errorf("page = %#v, want float64(2)", got)
	}
	if got := args["verbose"]; got != false {
		t.Errorf("verbose = %#v, want false", got)
	}
	if got := args["search_term"]; got != "911218" {
		t.Errorf("search_term = %#v, want untouched string \"911218\"", got)
	}
	tagIDs, ok := args["tag_ids"].([]any)
	if !ok || len(tagIDs) != 2 || tagIDs[0] != float64(1) || tagIDs[1] != float64(2) {
		t.Errorf("tag_ids = %#v, want [1 2] as numbers", args["tag_ids"])
	}
	groups, ok := args["groups"].(map[string]any)
	if !ok {
		t.Fatalf("groups = %#v, want map", args["groups"])
	}
	userIDs, ok := groups["user_ids"].([]any)
	if !ok || len(userIDs) != 1 || userIDs[0] != float64(7) {
		t.Errorf("groups.user_ids = %#v, want [7] as number", groups["user_ids"])
	}
}

func TestCoerceStringValuesNoChange(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"project_id":  nullableInteger(),
			"search_term": nullableString(),
		},
	}
	// Well-behaved client: native types already correct.
	args := map[string]any{"project_id": float64(911218), "search_term": "Inbox"}
	if coerceStringValues(schema, args) {
		t.Errorf("expected no change for already-typed arguments")
	}
}

// TestCoerceStringValuesComplex is the regression test for issue #402: a client
// that JSON-encodes a whole array or object parameter as a string (against an
// anyOf: [{array|object}, {null}] schema) must have it decoded to the native
// type so validation passes and the handler receives the real value.
func TestCoerceStringValuesComplex(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"tag_ids": {
				AnyOf: []*jsonschema.Schema{
					{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
					{Type: "null"},
				},
			},
			"assignees": {
				AnyOf: []*jsonschema.Schema{
					{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"user_ids": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
						},
					},
					{Type: "null"},
				},
			},
		},
	}

	args := map[string]any{
		"tag_ids":   `[1,2,3]`,              // whole array stringified
		"assignees": `{"user_ids":[1,2,3]}`, // whole object stringified
	}

	if !coerceStringValues(schema, args) {
		t.Fatalf("expected coerceStringValues to report a change")
	}

	tagIDs, ok := args["tag_ids"].([]any)
	if !ok || len(tagIDs) != 3 || tagIDs[0] != float64(1) {
		t.Errorf("tag_ids = %#v, want [1 2 3] as numbers", args["tag_ids"])
	}
	assignees, ok := args["assignees"].(map[string]any)
	if !ok {
		t.Fatalf("assignees = %#v, want map", args["assignees"])
	}
	if userIDs, ok := assignees["user_ids"].([]any); !ok || len(userIDs) != 3 {
		t.Errorf("assignees.user_ids = %#v, want [1 2 3]", assignees["user_ids"])
	}
}

// TestWithInputValidationCoercesStringifiedScalars is the regression test for
// issue #383: a client that sends project_id as the string "911218" against an
// anyOf: [{integer}, {null}] schema must pass validation, and the handler must
// receive it as a number.
func TestWithInputValidationCoercesStringifiedScalars(t *testing.T) {
	tool := &mcp.Tool{
		Name: "twprojects-list_tasklists",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"project_id": nullableInteger(),
				"verbose":    nullableBoolean(),
			},
		},
	}

	var received map[string]any
	handler := func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		received = map[string]any{}
		if err := json.Unmarshal(req.Params.Arguments, &received); err != nil {
			t.Fatalf("handler could not decode arguments: %v", err)
		}
		return &mcp.CallToolResult{}, nil
	}

	wrapped := withInputValidation(tool, handler)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{
		Arguments: json.RawMessage(`{"project_id":"911218","verbose":"false"}`),
	}}
	res, err := wrapped(context.Background(), req)
	if err != nil {
		t.Fatalf("wrapped handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected validation to pass, got error result: %+v", res.Content)
	}
	if received["project_id"] != float64(911218) {
		t.Errorf("handler received project_id = %#v, want float64(911218)", received["project_id"])
	}
	if received["verbose"] != false {
		t.Errorf("handler received verbose = %#v, want false", received["verbose"])
	}
}

// TestWithInputValidationRejectsInvalidInput ensures the coercion does not
// swallow genuinely invalid input: a non-numeric string for an integer field
// still fails validation.
func TestWithInputValidationRejectsInvalidInput(t *testing.T) {
	tool := &mcp.Tool{
		Name: "twprojects-list_tasklists",
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: map[string]*jsonschema.Schema{"project_id": nullableInteger()},
		},
	}
	wrapped := withInputValidation(tool, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t.Fatalf("handler must not be called for invalid input")
		return nil, nil
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{
		Arguments: json.RawMessage(`{"project_id":"not-a-number"}`),
	}}
	res, err := wrapped(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected validation error for non-numeric string")
	}
}

// TestScopes covers the list a server advertises as "scopes_supported": it must
// be deduplicated (several groups may share a scope) and stable, and it must
// leave out groups that declared no namespace, since those are never filtered.
func TestScopes(t *testing.T) {
	newGroup := func(toolPrefix, scope string) *ToolsetGroup {
		group := NewToolsetGroup(false)
		if toolPrefix != "" || scope != "" {
			group.SetNamespace(toolPrefix, scope)
		}
		return group
	}

	tests := []struct {
		name   string
		groups []*ToolsetGroup
		want   []string
	}{{
		name: "no groups",
		want: []string{},
	}, {
		name:   "a group without a namespace contributes nothing",
		groups: []*ToolsetGroup{newGroup("", "")},
		want:   []string{},
	}, {
		name:   "a group with a prefix but no scope contributes nothing",
		groups: []*ToolsetGroup{newGroup("twpro", "")},
		want:   []string{},
	}, {
		name:   "scopes are sorted",
		groups: []*ToolsetGroup{newGroup("twprojects", "projects"), newGroup("twdesk", "desk")},
		want:   []string{"desk", "projects"},
	}, {
		name:   "a shared scope is listed once",
		groups: []*ToolsetGroup{newGroup("twpro", "pro"), newGroup("twproplus", "pro")},
		want:   []string{"pro"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Scopes(tt.groups)
			if got == nil {
				t.Fatal("Scopes returned nil; it is marshalled straight into JSON metadata, " +
					"where nil renders as null rather than an empty array")
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Scopes = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDropNullArguments covers the argument-side half: the published schema no
// longer accepts a null, so the key is dropped before validation.
func TestDropNullArguments(t *testing.T) {
	schema := &jsonschema.Schema{
		Type:     "object",
		Required: []string{"id"},
		Properties: map[string]*jsonschema.Schema{
			"id":          {Type: "integer"},
			"page":        {Type: "integer"},
			"search_term": {Type: "string"},
			"nullable":    {Types: []string{"string", "null"}},
			"tag_ids":     {Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
			"assignees": {
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"user_ids": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				},
			},
		},
	}

	args := map[string]any{
		"id":          nil,           // required: kept, so validation names it
		"page":        nil,           // optional: dropped
		"search_term": "website",     // untouched
		"nullable":    nil,           // schema accepts null: kept
		"tag_ids":     []any{1, nil}, // inside an array: kept
		"assignees":   map[string]any{"user_ids": nil},
	}

	if !dropNullArguments(schema, args) {
		t.Fatal("expected dropNullArguments to report a change")
	}
	if _, ok := args["page"]; ok {
		t.Error("page was not dropped")
	}
	if _, ok := args["id"]; !ok {
		t.Error("id is required, so its null must be left for validation to report")
	}
	if _, ok := args["nullable"]; !ok {
		t.Error("nullable accepts null, so its value must be left alone")
	}
	if got := args["search_term"]; got != "website" {
		t.Errorf("search_term = %#v, want it untouched", got)
	}
	if tagIDs, ok := args["tag_ids"].([]any); !ok || len(tagIDs) != 2 {
		t.Errorf("tag_ids = %#v, want the array length preserved", args["tag_ids"])
	}
	assignees, ok := args["assignees"].(map[string]any)
	if !ok {
		t.Fatalf("assignees = %#v, want an object", args["assignees"])
	}
	if _, ok := assignees["user_ids"]; ok {
		t.Error("a null nested in an object was not dropped")
	}
}

// TestNormalizeInputSchemasOnRegistration checks the schema is rewritten as the
// tool enters a Toolset, so every reader sees the same shape.
func TestNormalizeInputSchemasOnRegistration(t *testing.T) {
	toolset := NewToolset("twprojects-tasks", "Tasks").AddReadTools(ToolWrapper{
		Tool: &mcp.Tool{
			Name:        "twprojects-list_tasks",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"page"},
				Properties: map[string]*jsonschema.Schema{
					"page": nullableInteger(),
				},
			},
		},
		Handler: func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{}, nil
		},
	})

	tools := toolset.GetAvailableTools()
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
	schema, ok := tools[0].Tool.InputSchema.(*jsonschema.Schema)
	if !ok {
		t.Fatalf("input schema is %T, want *jsonschema.Schema", tools[0].Tool.InputSchema)
	}
	page := schema.Properties["page"]
	if page.AnyOf != nil {
		t.Errorf("page anyOf = %v, want the null branch collapsed away", page.AnyOf)
	}
	if page.Type != "integer" {
		t.Errorf("page type = %q, want integer", page.Type)
	}
	if len(schema.Required) != 0 {
		t.Errorf("required = %v, want page dropped along with its null branch", schema.Required)
	}
}

// TestWithInputValidationAcceptsExplicitNulls is the end-to-end guarantee for
// the clients the null branches were added for: an explicit null still
// validates and reaches the handler as an absent key.
func TestWithInputValidationAcceptsExplicitNulls(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"project_id": nullableInteger(),
			"verbose":    nullableBoolean(),
		},
	}
	tool := &mcp.Tool{Name: "twprojects-list_tasklists", InputSchema: schema}
	// Publish it the way a Toolset would.
	normalizeInputSchemas([]ToolWrapper{{Tool: tool}})

	var received map[string]any
	wrapped := withInputValidation(tool, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		received = map[string]any{}
		if err := json.Unmarshal(req.Params.Arguments, &received); err != nil {
			t.Fatalf("handler could not decode arguments: %v", err)
		}
		return &mcp.CallToolResult{}, nil
	})

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{
		Arguments: json.RawMessage(`{"project_id":911218,"verbose":null}`),
	}}
	res, err := wrapped(context.Background(), req)
	if err != nil {
		t.Fatalf("wrapped handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("an explicit null must still validate, got: %+v", res.Content)
	}
	if received["project_id"] != float64(911218) {
		t.Errorf("handler received project_id = %#v, want float64(911218)", received["project_id"])
	}
	if _, ok := received["verbose"]; ok {
		t.Errorf("handler received verbose = %#v, want the key absent", received["verbose"])
	}
}
