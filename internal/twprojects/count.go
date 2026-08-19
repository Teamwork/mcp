package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTaskCount      toolsets.Method = "twprojects-count_tasks"
	MethodProjectCount   toolsets.Method = "twprojects-count_projects"
	MethodMilestoneCount toolsets.Method = "twprojects-count_milestones"
	MethodTimelogCount   toolsets.Method = "twprojects-count_timelogs"
)

// countToolDroppedParams are the list-tool parameters a count tool does not
// advertise: they all describe how rows come back, and it returns none.
var countToolDroppedParams = []string{
	"page", "page_size", "cursor", "limit", "verbose", "fields", "count_only",
	"order_by", "order_mode", "order_by_custom_field_id", "order_by_field_id",
}

// countToolResult is the response of a count tool — the same body count_only
// returns. It exists to generate the published output schema.
type countToolResult struct {
	// Count is the exact number of entities matching the filters.
	Count int64 `json:"count"`
}

var countToolOutputSchema *jsonschema.Schema

func init() {
	var err error

	// Strict on purpose: a count is a computed value, so there is no sparse
	// shape to relax for.
	countToolOutputSchema, err = jsonschema.For[countToolResult](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for countToolResult: %v", err))
	}
}

// TaskCount counts tasks in Teamwork.com.
func TaskCount(engine *twapi.Engine) toolsets.ToolWrapper {
	return countTool(TaskList(engine), MethodTaskCount, "Count Tasks",
		"Exact task count for any filter set: one call, one number, no rows. Use for \"how many\" questions "+
			"(totals, late, per person, per project, per tag, completed in a window) instead of paging "+
			"twprojects-list_tasks to count rows. Note due_before excludes tasks with no due date, unless a "+
			"related milestone due date matches. Use twprojects-list_tasks when the rows are needed.")
}

// ProjectCount counts projects in Teamwork.com.
func ProjectCount(engine *twapi.Engine) toolsets.ToolWrapper {
	return countTool(ProjectList(engine), MethodProjectCount, "Count Projects",
		"Exact project count for any filter set: one call, one number, no rows. Use for \"how many projects\" "+
			"questions (per category, per tag, matching a term) instead of paging twprojects-list_projects to "+
			"count rows. Use twprojects-list_projects when the rows are needed.")
}

// MilestoneCount counts milestones in Teamwork.com.
func MilestoneCount(engine *twapi.Engine) toolsets.ToolWrapper {
	return countTool(MilestoneList(engine), MethodMilestoneCount, "Count Milestones",
		"Exact milestone count for any filter set: one call, one number, no rows. Use for \"how many "+
			"milestones\" questions (per project, per tag, matching a term) instead of paging "+
			"twprojects-list_milestones to count rows. Use twprojects-list_milestones when the rows are needed.")
}

// TimelogCount counts timelogs in Teamwork.com.
func TimelogCount(engine *twapi.Engine) toolsets.ToolWrapper {
	return countTool(TimelogList(engine), MethodTimelogCount, "Count Timelogs",
		"Exact count of time entries for any filter set: one call, one number, no rows. This counts entries, "+
			"not hours — for logged, billable or billed time totals use twprojects-summarize_timelogs. Use for "+
			"\"how many time entries\" questions (per project, per task, per person, in a date window) instead of "+
			"paging twprojects-list_timelogs to count rows.")
}

// countTool derives a count tool from a list tool: the list tool called with
// count_only, so the filters, their validation and the endpoint stay defined
// once.
//
// Both surfaces exist because they are reached from opposite ends — count_only
// catches a model already in a list call path, count_tasks a model reasoning
// from "how many". Weaker models do not reliably apply a prompt rule telling
// them to pass a flag.
func countTool(
	list toolsets.ToolWrapper,
	method toolsets.Method,
	title string,
	description string,
) toolsets.ToolWrapper {
	listSchema, ok := list.Tool.InputSchema.(*jsonschema.Schema)
	if !ok {
		panic(fmt.Sprintf("cannot derive %s: %s has no JSON schema", method, list.Tool.Name))
	}

	properties := make(map[string]*jsonschema.Schema, len(listSchema.Properties))
	for name, property := range listSchema.Properties {
		if !slices.Contains(countToolDroppedParams, name) {
			properties[name] = property
		}
	}
	required := slices.DeleteFunc(slices.Clone(listSchema.Required), func(name string) bool {
		return slices.Contains(countToolDroppedParams, name)
	})

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(method),
			Description: description,
			Annotations: &mcp.ToolAnnotations{
				Title:           title,
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
				Required:   required,
			},
			OutputSchema: countToolOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if arguments == nil {
				arguments = make(map[string]any, 1)
			}
			arguments["count_only"] = true
			encoded, err := json.Marshal(arguments)
			if err != nil {
				return nil, fmt.Errorf("failed to encode request: %w", err)
			}
			params := *request.Params
			params.Arguments = encoded
			delegated := *request
			delegated.Params = &params

			// The SDK validated against this tool's schema, not the list tool's, so
			// count_only is injected past validation on purpose: it is not part of
			// this tool's surface.
			result, err := list.Handler(ctx, &delegated)
			if err != nil || result == nil || result.IsError {
				return result, err
			}
			// The count_only body is already this tool's body, so it passes through
			// unchanged. The check stops a broken delegation returning rows under a
			// schema that promises a count.
			if _, ok := helpers.CountFromToolResult(result); !ok {
				return nil, fmt.Errorf("%s did not report a count", list.Tool.Name)
			}
			return result, nil
		},
	}
}
