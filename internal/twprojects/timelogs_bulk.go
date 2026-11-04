package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
const (
	MethodTimelogCreateBatch  toolsets.Method = "twprojects-create_timelogs"
	MethodTimelogUpdateBatch  toolsets.Method = "twprojects-update_timelogs"
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

// TimelogCreateBatch creates many timelogs with a single request to the bulk
// time endpoint, which counts once against the rate limit.
func TimelogCreateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	itemSchema := batchItemSchema(TimelogCreate(engine).Tool.InputSchema)
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTimelogCreateBatch),
			Description: fmt.Sprintf("Create many timelogs in one request (at most %d per call). Use instead "+
				"of calling twprojects-create_timelog in a loop, e.g. to import time entries; split longer "+
				"lists across calls. Items are validated first and nothing is created if one is invalid; "+
				"after that each timelog succeeds or fails on its own, and the result lists the new timelog "+
				"ID of each item and the reason for each failure, by item number counting from 1.",
				batchMaxItems),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Timelogs",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"timelogs": {
						Type:     "array",
						Items:    itemSchema,
						MinItems: new(1),
						MaxItems: new(batchMaxItems),
						Description: fmt.Sprintf("The timelogs to create, at most %d, each with the parameters "+
							"of twprojects-create_timelog.", batchMaxItems),
					},
				},
				Required: []string{"timelogs"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "timelogs", batchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			timelogs := make([]projects.TimelogBulkCreateRequestTimelog, len(items))
			labels := make([]string, len(items))
			var invalid []string
			for i, item := range items {
				timelog, err := parseTimelogBulkCreateItem(item)
				labels[i] = timelogBatchLabel(i, timelog)
				if err != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, err.Error()))
					continue
				}
				timelogs[i] = timelog
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			response, err := projects.TimelogBulkCreate(ctx, engine, projects.NewTimelogBulkCreateRequest(timelogs))
			if err != nil {
				return timelogBulkCreateError(err, labels)
			}
			return timelogBulkCreateReport(labels, response), nil
		},
	}
}

// parseTimelogBulkCreateItem binds one item of create_timelogs with the same
// parameters as create_timelog.
func parseTimelogBulkCreateItem(item map[string]any) (projects.TimelogBulkCreateRequestTimelog, error) {
	var timelog projects.TimelogBulkCreateRequestTimelog
	err := helpers.ParamGroup(item,
		helpers.OptionalNumericPointerParam(&timelog.ProjectID, "project_id"),
		helpers.OptionalNumericPointerParam(&timelog.TaskID, "task_id"),
		helpers.OptionalPointerParam(&timelog.Description, "description"),
		helpers.RequiredDateParam(&timelog.Date, "date"),
		helpers.RequiredTimeOnlyParam(&timelog.Time, "time"),
		helpers.OptionalParam(&timelog.IsUTC, "is_utc"),
		helpers.RequiredNumericParam(&timelog.Hours, "hours"),
		helpers.RequiredNumericParam(&timelog.Minutes, "minutes"),
		helpers.OptionalPointerParam(&timelog.Billable, "billable"),
		helpers.OptionalNumericPointerParam(&timelog.UserID, "user_id"),
		helpers.OptionalNumericListParam(&timelog.TagIDs, "tag_ids"),
	)
	if err != nil {
		return timelog, err
	}
	// The endpoint would take a task's project over the project_id given, so
	// both at once is ambiguous.
	if (timelog.ProjectID == nil) == (timelog.TaskID == nil) {
		return timelog, errors.New("provide exactly one of project_id or task_id")
	}
	return timelog, nil
}

// timelogBatchLabel names an item by its position, target and date, so a
// failure can be traced back to the input row.
func timelogBatchLabel(i int, timelog projects.TimelogBulkCreateRequestTimelog) string {
	var target string
	switch {
	case timelog.TaskID != nil:
		target = fmt.Sprintf("task %d, ", *timelog.TaskID)
	case timelog.ProjectID != nil:
		target = fmt.Sprintf("project %d, ", *timelog.ProjectID)
	}
	return fmt.Sprintf("item %d (%s%s)", i+1, target, time.Time(timelog.Date).Format(time.DateOnly))
}

// timelogBulkCreateError reports a bulk create that did not answer 201. A
// rejection naming items means nothing was created; a server error or a lost
// answer leaves the outcome unknown, because each entry is committed as it is
// created.
func timelogBulkCreateError(err error, labels []string) (*mcp.CallToolResult, error) {
	if failures := projects.TimelogBulkCreateFailures(err); len(failures) > 0 {
		return timelogBulkCreateNothingCreated(labels, failures), nil
	}
	const recheck = "Some timelogs may have been created; call " + string(MethodTimelogList) +
		" to check before sending the list again."
	if _, ok := errors.AsType[*twapi.HTTPError](err); !ok {
		return helpers.NewToolResultTextError("failed to create timelogs: %s. %s", err.Error(), recheck), nil
	}
	result, handleErr := helpers.HandleAPIError(err, "failed to create timelogs")
	if result != nil && isServerError(err) {
		result.Content = append(result.Content, &mcp.TextContent{Text: recheck})
	}
	return result, handleErr
}

// timelogBulkCreateReport maps the created timelogs back to the items they came
// from. The endpoint returns them in request order, leaving out the failures.
func timelogBulkCreateReport(labels []string, response *projects.TimelogBulkCreateResponse) *mcp.CallToolResult {
	errs := make([]error, len(labels))
	for _, failure := range response.Failed {
		if failure.Index >= 0 && failure.Index < len(errs) {
			errs[failure.Index] = errors.New(bulkFailureReasons(failure.Errors))
		}
	}

	details := make([]string, len(labels))
	next := 0
	for i := range errs {
		if errs[i] != nil {
			continue
		}
		if next >= len(response.Timelogs) {
			errs[i] = errors.New("not reported as created or failed; check with " +
				string(MethodTimelogList) + " before sending it again")
			continue
		}
		details[i] = fmt.Sprintf("timelog %d", response.Timelogs[next].ID)
		next++
	}
	return batchReport("Created", "timelogs",
		"check with "+string(MethodTimelogList)+" before sending them again", labels, details, errs)
}

// timelogBulkCreateNothingCreated reports a request the endpoint rejected as a
// whole, naming each item that caused it.
func timelogBulkCreateNothingCreated(
	labels []string,
	failures []projects.TimelogBulkCreateFailure,
) *mcp.CallToolResult {
	var report strings.Builder
	fmt.Fprintf(&report, "Created 0 of %d timelogs. Nothing was created.", len(labels))
	for _, failure := range failures {
		label := fmt.Sprintf("item %d", failure.Index+1)
		if failure.Index >= 0 && failure.Index < len(labels) {
			label = labels[failure.Index]
		}
		fmt.Fprintf(&report, "\nFailed: %s: %s.", label, bulkFailureReasons(failure.Errors))
	}
	return helpers.NewToolResultTextError("%s", report.String())
}

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

// TimelogUpdateBatch updates many timelogs in one call. The API has no bulk
// update, so it fans out one update per timelog.
func TimelogUpdateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	itemSchema := batchItemSchema(TimelogUpdate(engine).Tool.InputSchema)
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTimelogUpdateBatch),
			Description: "Update many timelogs in one call, each with its own changes. Use instead of calling " +
				"twprojects-update_timelog in a loop. Items are validated first and nothing is updated if one " +
				"is invalid; after that each timelog succeeds or fails on its own, and the result lists the " +
				"error of each failure.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Timelogs",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"timelogs": {
						Type:     "array",
						Items:    itemSchema,
						MinItems: new(1),
						MaxItems: new(batchMaxItems),
						Description: "The updates, one per timelog, each with the parameters of " +
							"twprojects-update_timelog. A timelog may appear only once.",
					},
				},
				Required: []string{"timelogs"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "timelogs", batchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			requests := make([]projects.TimelogUpdateRequest, len(items))
			labels := make([]string, len(items))
			seen := make(map[int64]bool, len(items))
			var invalid []string
			for i, item := range items {
				req, toolResult := parseTimelogUpdateArguments(item)
				if toolResult != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, toolResultText(toolResult)))
					continue
				}
				if seen[req.Path.ID] {
					invalid = append(invalid, fmt.Sprintf("item %d: timelog %d is listed more than once; "+
						"merge its changes into one item", i+1, req.Path.ID))
					continue
				}
				seen[req.Path.ID] = true
				requests[i] = req
				labels[i] = fmt.Sprintf("timelog %d", req.Path.ID)
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			errs := runBatch(ctx, len(requests), batchConcurrency, func(ctx context.Context, i int) error {
				return retryIdempotent(ctx, func() error {
					_, err := projects.TimelogUpdate(ctx, engine, requests[i])
					return err
				})
			})
			return batchReport("Updated", "timelogs",
				"sending the same update again is safe", labels, nil, errs), nil
		},
	}
}
