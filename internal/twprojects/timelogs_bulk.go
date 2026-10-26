package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
const (
	MethodTimelogBulkMove     toolsets.Method = "twprojects-move_timelogs"
	MethodTimelogBillableBulk toolsets.Method = "twprojects-set_timelogs_billable"
)

const (
	// timelogBulkMoveMaxItems bounds the list one move_timelogs call sends. The
	// whole list is a single request, so the bound keeps one call reviewable
	// rather than protecting the rate limit.
	timelogBulkMoveMaxItems = 100

	// timelogBillableMaxUserIDs bounds the user filter of set_timelogs_billable.
	timelogBillableMaxUserIDs = 50
)

// TimelogBulkMove moves several timelogs from one project to another with a
// single request.
func TimelogBulkMove(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTimelogBulkMove),
			Description: "Move up to 100 timelogs from one project to another in a single request. Only moving " +
				"to another project is supported; for any other timelog change use twprojects-update_timelog. " +
				"Every listed timelog must be in source_project_id: the endpoint skips any that is not, along " +
				"with deleted and already invoiced ones, and reports only how many moved, not which. Moved " +
				"timelogs are detached from their task and lose their project tags, and land on the target " +
				"project itself; to move a timelog onto a task in the target project, use " +
				"twprojects-update_timelog with task_id instead. Needs permission to log time in both projects.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Move Timelogs",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"source_project_id": {
						Type:        "integer",
						Description: "The project the timelogs are currently logged against.",
					},
					"target_project_id": {
						Type:        "integer",
						Description: "The project that will receive the timelogs.",
					},
					"timelog_ids": {
						Type:        "array",
						Items:       &jsonschema.Schema{Type: "integer"},
						MinItems:    new(1),
						MaxItems:    new(timelogBulkMoveMaxItems),
						Description: "The IDs of the timelogs to move, at most 100, all in source_project_id.",
					},
				},
				Required: []string{"source_project_id", "target_project_id", "timelog_ids"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var sourceProjectID, targetProjectID int64
			var timelogIDs []int64
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&sourceProjectID, "source_project_id"),
				helpers.RequiredNumericParam(&targetProjectID, "target_project_id"),
				helpers.OptionalNumericListParam(&timelogIDs, "timelog_ids"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			switch {
			case sourceProjectID <= 0 || targetProjectID <= 0:
				return helpers.NewToolResultTextError("source_project_id and target_project_id must be positive"), nil
			case sourceProjectID == targetProjectID:
				return helpers.NewToolResultTextError("source_project_id and target_project_id must differ"), nil
			case len(timelogIDs) == 0:
				return helpers.NewToolResultTextError("timelog_ids must contain at least one timelog ID"), nil
			case len(timelogIDs) > timelogBulkMoveMaxItems:
				return helpers.NewToolResultTextError(
					"timelog_ids accepts at most %d timelogs, got %d; nothing was moved, split the list across "+
						"several calls", timelogBulkMoveMaxItems, len(timelogIDs)), nil
			}

			seen := make(map[int64]bool, len(timelogIDs))
			unique := make([]int64, 0, len(timelogIDs))
			for _, id := range timelogIDs {
				if id <= 0 {
					return helpers.NewToolResultTextError("timelog_ids must be positive, got %d", id), nil
				}
				if !seen[id] {
					seen[id] = true
					unique = append(unique, id)
				}
			}

			response, err := projects.TimelogBulkMove(ctx, engine,
				projects.NewTimelogBulkMoveRequest(sourceProjectID, targetProjectID, unique))
			if err != nil {
				// The endpoint answers 200 with an errMsg body when it moved nothing,
				// which the SDK reports as an HTTPError carrying that 200.
				if httpErr, ok := errors.AsType[*twapi.HTTPError](err); ok && httpErr.StatusCode == http.StatusOK {
					return helpers.NewToolResultTextError(
						"No timelogs were moved: %s. Check that every timelog is in project %d, is not invoiced, "+
							"and that you can log time in both projects.",
						httpErr.Details, sourceProjectID), nil
				}
				return helpers.HandleAPIError(err, "failed to move timelogs")
			}

			moved := int64(response.MovedCount)
			if moved < int64(len(unique)) {
				return helpers.NewToolResultText(
					"Moved %d of %d timelogs from project %d to project %d. The other %d were skipped because "+
						"they are not in project %d, are deleted, or are already invoiced; the endpoint does not "+
						"say which. List the timelogs of project %d to see what moved.",
					moved, len(unique), sourceProjectID, targetProjectID, int64(len(unique))-moved,
					sourceProjectID, targetProjectID), nil
			}
			return helpers.NewToolResultText("Moved %d timelogs from project %d to project %d.",
				moved, sourceProjectID, targetProjectID), nil
		},
	}
}

// TimelogBillableBulkUpdate sets the billable status of the timelogs matching
// a filter with a single request.
func TimelogBillableBulkUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTimelogBillableBulk),
			Description: "Set the billable status of every timelog in one project that matches a filter, in a " +
				"single request. Only billable status is supported; for any other timelog change use " +
				"twprojects-update_timelog. This tool works by filter, not by timelog ID: it changes every " +
				"timelog the filters match, which can be far more than the user has in mind. Before calling " +
				"it, call twprojects-count_timelogs with the same project_id, start_date, end_date, " +
				"assigned_user_ids (for user_ids), tag_ids and match_all_tags, billable_type set to the " +
				"opposite of the target status, and invoiced_type=noninvoiced unless include_invoiced is " +
				"true; tell the user that count and the filters, and call this tool only after they confirm. " +
				"Only timelogs that do not already have the target status are changed, and the result reports " +
				"how many. Only site administrators of the owner company may use it; anyone else is refused.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set Timelogs Billable",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type: "integer",
						Description: "The project whose timelogs are changed. Required, so one call never " +
							"reaches beyond a single project.",
					},
					"billable": {
						Type:        "boolean",
						Description: "The status to set: true makes the matching timelogs billable, false non-billable.",
					},
					"start_date": helpers.DateFilterSchema(
						"Only change timelogs logged on or after this day, in the user's timezone; the day itself " +
							"is included."),
					"end_date": helpers.DateFilterSchema(
						"Only change timelogs logged on or before this day, in the user's timezone; the whole day " +
							"is included."),
					"user_ids": {
						Description: "Only change timelogs logged for these users, at most 50.",
						AnyOf: []*jsonschema.Schema{
							{
								Type:     "array",
								Items:    &jsonschema.Schema{Type: "integer"},
								MaxItems: new(timelogBillableMaxUserIDs),
							},
							{Type: "null"},
						},
					},
					"tag_ids":        helpers.TagIDsFilterSchema("timelogs"),
					"match_all_tags": helpers.MatchAllTagsSchema(),
					"include_invoiced": {
						Description: "Also change timelogs already added to an invoice. Defaults to false, so " +
							"invoiced time is left alone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
						Default: []byte("false"),
					},
				},
				Required: []string{"project_id", "billable"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var projectID int64
			var billable, includeInvoiced bool
			var userIDs, tagIDs []int64
			var startDate, endDate *projects.LegacyDate
			var matchAllTags *bool
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectID, "project_id"),
				helpers.RequiredParam(&billable, "billable"),
				helpers.OptionalLegacyDatePointerParam(&startDate, "start_date"),
				helpers.OptionalLegacyDatePointerParam(&endDate, "end_date"),
				helpers.OptionalNumericListParam(&userIDs, "user_ids"),
				helpers.OptionalNumericListParam(&tagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&matchAllTags, "match_all_tags"),
				helpers.OptionalParam(&includeInvoiced, "include_invoiced"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			// A project of 0 is read by the endpoint as "every project", so it is
			// refused rather than sent.
			if projectID <= 0 {
				return helpers.NewToolResultTextError("project_id must be a positive project ID; nothing was changed"), nil
			}
			if len(userIDs) > timelogBillableMaxUserIDs {
				return helpers.NewToolResultTextError(
					"user_ids accepts at most %d users, got %d; nothing was changed",
					timelogBillableMaxUserIDs, len(userIDs)), nil
			}
			for _, id := range userIDs {
				if id <= 0 {
					return helpers.NewToolResultTextError("user_ids must be positive, got %d; nothing was changed", id), nil
				}
			}

			req := projects.NewTimelogBulkBillableUpdateRequest(billable, projectID)
			req.UserIDs = userIDs
			req.TagIDs = tagIDs
			req.MatchAllTags = matchAllTags
			req.FromDate = startDate
			req.ToDate = endDate
			// Select only the timelogs that change, so the count is the number
			// changed rather than the number matched.
			req.BillableType = projects.TimelogBillableTypeBillable
			if billable {
				req.BillableType = projects.TimelogBillableTypeNonBillable
			}
			req.InvoicedType = projects.TimelogInvoicedTypeNonInvoiced
			if includeInvoiced {
				req.InvoicedType = projects.TimelogInvoicedTypeAll
			}

			status := "non-billable"
			if billable {
				status = "billable"
			}

			response, err := projects.TimelogBulkBillableUpdate(ctx, engine, req)
			if err != nil {
				// The endpoint answers 422 when the filters match nothing.
				if httpErr, ok := errors.AsType[*twapi.HTTPError](err); ok &&
					httpErr.StatusCode == http.StatusUnprocessableEntity &&
					strings.Contains(httpErr.Details, "Nothing to modify") {
					return helpers.NewToolResultText(
						"Changed 0 timelogs: no timelog in project %d matching these filters is still waiting "+
							"to become %s. Nothing was changed.", projectID, status), nil
				}
				return helpers.HandleAPIError(err, "failed to set timelogs billable status")
			}
			if response.Count == nil {
				return helpers.NewToolResultText(
					"Made the matching timelogs in project %d %s; the endpoint did not report how many.",
					projectID, status), nil
			}
			return helpers.NewToolResultText("Changed %d timelogs in project %d to %s.",
				*response.Count, projectID, status), nil
		},
	}
}
