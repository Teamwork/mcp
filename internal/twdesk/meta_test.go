package twdesk_test

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// maxDeskPageSize is the largest page any Desk endpoint honours.
const maxDeskPageSize = 200.0

// pageSizeMaximum returns the upper bound a tool advertises on pageSize, or nil
// if it advertises none.
func pageSizeMaximum(t *testing.T, tool toolsets.ToolWrapper) *float64 {
	t.Helper()

	schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
	if !ok {
		t.Fatalf("tool %q has an unexpected input schema type %T", tool.Tool.Name, tool.Tool.InputSchema)
	}
	pageSize, ok := schema.Properties["pageSize"]
	if !ok {
		return nil
	}
	for _, branch := range pageSize.AnyOf {
		if branch.Type == "integer" {
			return branch.Maximum
		}
	}
	return nil
}

// TestPageSizeCeilingsMatchTheAPI pins the two page-size ceilings Desk actually
// enforces. Desk deliberately does not use helpers.PageSizeSchema, whose 500 is
// the v3 API's limit: Desk rewrites an oversized page rather than rejecting it,
// so advertising a page the caller cannot have returns a short result set that
// looks like the end of the data.
//
// The two ceilings differ because the endpoints clamp differently — list
// endpoints cut to 100, the search endpoints reset anything over 200 back to 50,
// which makes a request for 250 return fewer rows than a request for 200.
func TestPageSizeCeilingsMatchTheAPI(t *testing.T) {
	tests := []struct {
		name string
		tool toolsets.ToolWrapper
		want float64
	}{
		{name: "search endpoint", tool: twdesk.TicketSearch(nil), want: 200},
		{name: "list endpoint", tool: twdesk.TagList(nil), want: 100},
		{name: "list endpoint with its own filters", tool: twdesk.UserList(nil), want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pageSizeMaximum(t, tt.tool)
			if got == nil {
				t.Fatalf("tool %q declares no maximum on pageSize, so any page is advertised as valid",
					tt.tool.Tool.Name)
			}
			if *got != tt.want {
				t.Errorf("tool %q pageSize maximum = %v, want %v", tt.tool.Tool.Name, *got, tt.want)
			}
		})
	}
}

// TestNoToolAdvertisesAnUnreachablePage guards the specific regression across
// the whole product: reaching for helpers.PageSizeSchema in a Desk tool compiles
// and validates fine, and only shows up as truncated results against the live
// API.
func TestNoToolAdvertisesAnUnreachablePage(t *testing.T) {
	group := twdesk.DefaultToolsetGroup(false, nil)
	if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}

	var checked int
	for method, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			maximum := pageSizeMaximum(t, tool)
			if maximum == nil {
				continue
			}
			checked++
			if *maximum > maxDeskPageSize {
				t.Errorf("%s: tool %q advertises pageSize up to %v; Desk honours at most %v",
					method, tool.Tool.Name, *maximum, maxDeskPageSize)
			}
		}
	}

	// Without this the sweep passes silently if the schema shape changes and
	// pageSizeMaximum stops finding anything.
	if checked == 0 {
		t.Fatal("no tool was checked; the sweep is not reaching any pageSize schema")
	}
}
